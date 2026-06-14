package handlers

import (
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

const voucherChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O or 1/I

func generateVoucherCode() (string, error) {
	parts := make([]string, 3)
	for p := range parts {
		b := make([]byte, 4)
		for i := range b {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(voucherChars))))
			if err != nil {
				return "", err
			}
			b[i] = voucherChars[n.Int64()]
		}
		parts[p] = string(b)
	}
	return strings.Join(parts, "-"), nil
}

// GenerateVouchersRequest is the payload for batch voucher generation.
type GenerateVouchersRequest struct {
	Count        int    `json:"count" binding:"required,min=1,max=500"`
	BatchName    string `json:"batch_name"`
	DataLimitMB  *int64 `json:"data_limit_mb"`
	TimeLimitMin *int   `json:"time_limit_minutes"`
	ValidDays    int    `json:"valid_days"`
}

// ListVouchers returns paginated vouchers with optional filters.
func (h *Handler) ListVouchers(c *gin.Context) {
	offset, limit := paginationParams(c)
	status := c.Query("status")
	batch := c.Query("batch")

	where := "WHERE 1=1"
	args := []interface{}{}
	argN := 1

	if status != "" {
		where += fmt.Sprintf(" AND v.status = $%d", argN)
		args = append(args, status)
		argN++
	}
	if batch != "" {
		where += fmt.Sprintf(" AND v.batch_name ILIKE $%d", argN)
		args = append(args, "%"+batch+"%")
		argN++
	}

	var total int
	h.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM vouchers v %s", where), args...).Scan(&total)

	dataArgs := append(args, limit, offset)
	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT v.id, v.code, v.batch_name, v.status, v.data_limit_mb, v.time_limit_minutes,
		       v.valid_days, v.expires_at, v.redeemed_by, v.redeemed_at,
		       au.username, v.created_at
		FROM vouchers v
		LEFT JOIN app_users au ON au.id = v.created_by
		%s
		ORDER BY v.created_at DESC
		LIMIT $%d OFFSET $%d`, where, argN, argN+1), dataArgs...)
	if err != nil {
		h.log.WithError(err).Error("list vouchers failed")
		respondError(c, http.StatusInternalServerError, "failed to fetch vouchers")
		return
	}
	defer rows.Close()

	type VoucherRow struct {
		ID           int        `json:"id"`
		Code         string     `json:"code"`
		BatchName    *string    `json:"batch_name"`
		Status       string     `json:"status"`
		DataLimitMB  *int64     `json:"data_limit_mb"`
		TimeLimitMin *int       `json:"time_limit_minutes"`
		ValidDays    int        `json:"valid_days"`
		ExpiresAt    *time.Time `json:"expires_at"`
		RedeemedBy   *string    `json:"redeemed_by"`
		RedeemedAt   *time.Time `json:"redeemed_at"`
		CreatedBy    *string    `json:"created_by"`
		CreatedAt    time.Time  `json:"created_at"`
	}

	vouchers := []VoucherRow{}
	for rows.Next() {
		var v VoucherRow
		rows.Scan(&v.ID, &v.Code, &v.BatchName, &v.Status, &v.DataLimitMB, &v.TimeLimitMin,
			&v.ValidDays, &v.ExpiresAt, &v.RedeemedBy, &v.RedeemedAt, &v.CreatedBy, &v.CreatedAt)
		vouchers = append(vouchers, v)
	}

	c.JSON(http.StatusOK, gin.H{"data": vouchers, "total": total})
}

// GenerateVouchers creates a batch of voucher codes.
func (h *Handler) GenerateVouchers(c *gin.Context) {
	var req GenerateVouchersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.ValidDays == 0 {
		req.ValidDays = 30
	}

	claims, _ := middleware.GetClaims(c)
	expiresAt := time.Now().AddDate(0, 0, req.ValidDays)

	batchName := req.BatchName
	if batchName == "" {
		batchName = fmt.Sprintf("Batch-%s", time.Now().Format("20060102-1504"))
	}

	created := 0
	for i := 0; i < req.Count; i++ {
		for attempt := 0; attempt < 10; attempt++ {
			code, err := generateVoucherCode()
			if err != nil {
				break
			}
			_, dbErr := h.db.Exec(`
				INSERT INTO vouchers (code, batch_name, data_limit_mb, time_limit_minutes, valid_days, expires_at, created_by)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				code, batchName, req.DataLimitMB, req.TimeLimitMin, req.ValidDays, expiresAt, claims.UserID)
			if dbErr != nil {
				continue // collision, retry
			}
			// Register code in radcheck so FreeRADIUS accepts it
			h.db.Exec(`
				INSERT INTO radcheck (username, attribute, op, value)
				VALUES ($1, 'Cleartext-Password', ':=', $1)
				ON CONFLICT (username, attribute) DO NOTHING`, code)
			if req.DataLimitMB != nil {
				bytes := *req.DataLimitMB * 1024 * 1024
				h.db.Exec(`INSERT INTO radcheck (username, attribute, op, value)
					VALUES ($1, 'Max-Octets', ':=', $2)
					ON CONFLICT (username, attribute) DO UPDATE SET value = $2`,
					code, fmt.Sprintf("%d", bytes))
			}
			if req.TimeLimitMin != nil {
				h.db.Exec(`INSERT INTO radcheck (username, attribute, op, value)
					VALUES ($1, 'Session-Timeout', ':=', $2)
					ON CONFLICT (username, attribute) DO UPDATE SET value = $2`,
					code, fmt.Sprintf("%d", *req.TimeLimitMin*60))
			}
			created++
			break
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("Generated %d vouchers in batch '%s'", created, batchName),
		"count":   created,
		"batch":   batchName,
	})
}

// GetVoucherBatches lists all voucher batches with summary counts.
func (h *Handler) GetVoucherBatches(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT batch_name,
		       COUNT(*) as total,
		       SUM(CASE WHEN status='active'   THEN 1 ELSE 0 END) as active,
		       SUM(CASE WHEN status='used'     THEN 1 ELSE 0 END) as used,
		       SUM(CASE WHEN status='disabled' THEN 1 ELSE 0 END) as disabled,
		       MIN(created_at) as created_at
		FROM vouchers
		WHERE batch_name IS NOT NULL
		GROUP BY batch_name
		ORDER BY MIN(created_at) DESC`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch batches")
		return
	}
	defer rows.Close()

	type Batch struct {
		Name      *string   `json:"name"`
		Total     int       `json:"total"`
		Active    int       `json:"active"`
		Used      int       `json:"used"`
		Disabled  int       `json:"disabled"`
		CreatedAt time.Time `json:"created_at"`
	}

	batches := []Batch{}
	for rows.Next() {
		var b Batch
		rows.Scan(&b.Name, &b.Total, &b.Active, &b.Used, &b.Disabled, &b.CreatedAt)
		batches = append(batches, b)
	}
	c.JSON(http.StatusOK, gin.H{"data": batches})
}

// DisableVoucher sets a voucher's status to disabled and blocks RADIUS auth.
func (h *Handler) DisableVoucher(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var code string
	if err := h.db.QueryRow(`SELECT code FROM vouchers WHERE id = $1`, id).Scan(&code); err != nil {
		respondError(c, http.StatusNotFound, "voucher not found")
		return
	}
	h.db.Exec(`UPDATE vouchers SET status = 'disabled', updated_at = NOW() WHERE id = $1`, id)
	h.db.Exec(`INSERT INTO radcheck (username, attribute, op, value)
		VALUES ($1, 'Auth-Type', ':=', 'Reject')
		ON CONFLICT (username, attribute) DO UPDATE SET value = 'Reject'`, code)
	c.JSON(http.StatusOK, gin.H{"message": "voucher disabled"})
}

// DeleteVoucher removes a voucher and its RADIUS entries.
func (h *Handler) DeleteVoucher(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var code string
	if err := h.db.QueryRow(`SELECT code FROM vouchers WHERE id = $1`, id).Scan(&code); err != nil {
		respondError(c, http.StatusNotFound, "voucher not found")
		return
	}
	h.db.Exec(`DELETE FROM radcheck WHERE username = $1`, code)
	h.db.Exec(`DELETE FROM vouchers WHERE id = $1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "voucher deleted"})
}

// ExportVouchers exports vouchers as CSV.
func (h *Handler) ExportVouchers(c *gin.Context) {
	batch := c.Query("batch")
	where := "WHERE 1=1"
	args := []interface{}{}
	if batch != "" {
		where += " AND batch_name = $1"
		args = append(args, batch)
	}

	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT code, batch_name, status, data_limit_mb, time_limit_minutes,
		       valid_days, expires_at, redeemed_by, created_at
		FROM vouchers %s ORDER BY batch_name, created_at`, where), args...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "export failed")
		return
	}
	defer rows.Close()

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="vouchers_export.csv"`)

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"code", "batch", "status", "data_limit_mb", "time_limit_min", "valid_days", "expires_at", "redeemed_by", "created_at"})

	ns := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	ni64 := func(i *int64) string {
		if i == nil {
			return ""
		}
		return fmt.Sprintf("%d", *i)
	}
	ni := func(i *int) string {
		if i == nil {
			return ""
		}
		return fmt.Sprintf("%d", *i)
	}
	nt := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format("2006-01-02 15:04")
	}

	for rows.Next() {
		var code, status string
		var batch, redeemedBy *string
		var dataLimitMB *int64
		var timeLimitMins, validDays *int
		var expiresAt *time.Time
		var createdAt time.Time

		rows.Scan(&code, &batch, &status, &dataLimitMB, &timeLimitMins, &validDays, &expiresAt, &redeemedBy, &createdAt)
		vd := 30
		if validDays != nil {
			vd = *validDays
		}
		w.Write([]string{
			code, ns(batch), status, ni64(dataLimitMB), ni(timeLimitMins),
			fmt.Sprintf("%d", vd), nt(expiresAt), ns(redeemedBy), createdAt.Format("2006-01-02 15:04"),
		})
	}
	w.Flush()
}
