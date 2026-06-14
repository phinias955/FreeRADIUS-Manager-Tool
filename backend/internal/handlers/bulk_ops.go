package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Bulk User Operations
// ─────────────────────────────────────────────────────────────────────────────

// BulkOperation performs an action on multiple RADIUS users.
func (h *Handler) BulkOperation(c *gin.Context) {
	var req struct {
		Usernames []string               `json:"usernames" binding:"required,min=1"`
		Action    string                 `json:"action"    binding:"required"`
		Params    map[string]interface{} `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	claims, _ := middleware.GetClaims(c)
	var performedBy *int
	if claims != nil {
		id := claims.UserID
		performedBy = &id
	}

	type Result struct {
		Username string `json:"username"`
		Success  bool   `json:"success"`
		Message  string `json:"message"`
	}

	results := []Result{}
	successCount, failCount := 0, 0

	for _, username := range req.Usernames {
		var msg string
		var opErr error

		switch req.Action {
		case "suspend":
			_, opErr = h.db.Exec(`UPDATE radius_users SET status='suspended' WHERE username=$1`, username)
			if opErr == nil {
				h.db.Exec(`DELETE FROM radcheck WHERE username=$1 AND attribute='Auth-Type'`, username)
				h.db.Exec(`INSERT INTO radcheck (username,attribute,op,value) VALUES ($1,'Auth-Type',':=','Reject')`, username)
				msg = "suspended"
			}

		case "activate":
			_, opErr = h.db.Exec(`UPDATE radius_users SET status='active' WHERE username=$1`, username)
			if opErr == nil {
				h.db.Exec(`DELETE FROM radcheck WHERE username=$1 AND attribute='Auth-Type' AND value='Reject'`, username)
				msg = "activated"
			}

		case "delete":
			h.db.Exec(`DELETE FROM radcheck WHERE username=$1`, username)
			h.db.Exec(`DELETE FROM radreply WHERE username=$1`, username)
			h.db.Exec(`DELETE FROM radusergroup WHERE username=$1`, username)
			_, opErr = h.db.Exec(`DELETE FROM radius_users WHERE username=$1`, username)
			msg = "deleted"

		case "change_plan":
			planID := 0
			if req.Params != nil {
				if v, ok := req.Params["plan_id"]; ok {
					if fv, ok := v.(float64); ok {
						planID = int(fv)
					}
				}
			}
			if planID == 0 {
				opErr = fmt.Errorf("plan_id required in params")
			} else {
				_, opErr = h.db.Exec(`UPDATE radius_users SET plan_id=$1 WHERE username=$2`, planID, username)
				msg = fmt.Sprintf("plan changed to #%d", planID)
			}

		case "set_expiry":
			expiry := ""
			if req.Params != nil {
				if v, ok := req.Params["expiry"]; ok {
					expiry, _ = v.(string)
				}
			}
			if expiry == "" {
				opErr = fmt.Errorf("expiry required in params")
			} else {
				_, opErr = h.db.Exec(`UPDATE radius_users SET account_expiry=$1 WHERE username=$2`, expiry, username)
				if opErr == nil {
					h.db.Exec(`INSERT INTO radcheck (username,attribute,op,value) VALUES ($1,'Expiration',':=',$2)
						ON CONFLICT (username,attribute) DO UPDATE SET value=$2`, username, expiry)
					msg = "expiry updated to " + expiry
				}
			}

		case "apply_template":
			templateID := 0
			if req.Params != nil {
				if v, ok := req.Params["template_id"]; ok {
					if fv, ok := v.(float64); ok {
						templateID = int(fv)
					}
				}
			}
			if templateID == 0 {
				opErr = fmt.Errorf("template_id required in params")
			} else {
				var attrsRaw []byte
				opErr = h.db.QueryRow(`SELECT attributes FROM radius_templates WHERE id=$1`, templateID).Scan(&attrsRaw)
				if opErr == nil {
					var attrs []AttributeEntry
					json.Unmarshal(attrsRaw, &attrs)
					for _, attr := range attrs {
						if attr.Table == "reply" || attr.Table == "" {
							h.db.Exec(`INSERT INTO radreply (username,attribute,op,value) VALUES ($1,$2,$3,$4)
								ON CONFLICT (username,attribute) DO UPDATE SET value=$4`,
								username, attr.Attribute, attr.Op, attr.Value)
						} else {
							h.db.Exec(`INSERT INTO radcheck (username,attribute,op,value) VALUES ($1,$2,$3,$4)
								ON CONFLICT (username,attribute) DO UPDATE SET value=$4`,
								username, attr.Attribute, attr.Op, attr.Value)
						}
					}
					msg = fmt.Sprintf("template #%d applied", templateID)
				}
			}

		case "reset_attributes":
			h.db.Exec(`DELETE FROM radreply WHERE username=$1 AND attribute IN
				('Mikrotik-Rate-Limit','WISPr-Bandwidth-Max-Up','WISPr-Bandwidth-Max-Down','Framed-IP-Address')`, username)
			h.db.Exec(`DELETE FROM radcheck WHERE username=$1 AND attribute IN
				('Max-Octets','Session-Timeout','Expiration','Simultaneous-Use')`, username)
			_, opErr = h.db.Exec(`SELECT 1 FROM radius_users WHERE username=$1`, username)
			msg = "RADIUS attributes reset"

		default:
			opErr = fmt.Errorf("unknown action: %s", req.Action)
		}

		success := opErr == nil
		if success {
			successCount++
			if msg == "" {
				msg = "ok"
			}
		} else {
			failCount++
			msg = opErr.Error()
		}
		results = append(results, Result{Username: username, Success: success, Message: msg})
	}

	// Log bulk operation
	paramsJSON, _ := json.Marshal(req.Params)
	h.db.Exec(`INSERT INTO bulk_operations (operation,target_count,success_count,fail_count,params,performed_by)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6)`,
		req.Action, len(req.Usernames), successCount, failCount, string(paramsJSON), performedBy)

	c.JSON(http.StatusOK, gin.H{
		"action":        req.Action,
		"total":         len(req.Usernames),
		"success_count": successCount,
		"fail_count":    failCount,
		"results":       results,
	})
}

// ListBulkOpHistory returns recent bulk operation history.
func (h *Handler) ListBulkOpHistory(c *gin.Context) {
	type BulkOp struct {
		ID           int       `json:"id"`
		Operation    string    `json:"operation"`
		TargetCount  int       `json:"target_count"`
		SuccessCount int       `json:"success_count"`
		FailCount    int       `json:"fail_count"`
		PerformedBy  *string   `json:"performed_by"`
		CreatedAt    time.Time `json:"created_at"`
	}
	rows, _ := h.db.Query(`
		SELECT bo.id, bo.operation, bo.target_count, bo.success_count, bo.fail_count,
		       u.username, bo.created_at
		FROM bulk_operations bo
		LEFT JOIN app_users u ON u.id = bo.performed_by
		ORDER BY bo.created_at DESC LIMIT 50`)

	ops := []BulkOp{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var op BulkOp
			rows.Scan(&op.ID, &op.Operation, &op.TargetCount, &op.SuccessCount, &op.FailCount, &op.PerformedBy, &op.CreatedAt)
			ops = append(ops, op)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": ops})
}
