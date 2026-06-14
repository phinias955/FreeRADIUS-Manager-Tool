package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/freeradius-manager/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Invoice holds a billing record.
type Invoice struct {
	ID            int        `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	UserID        *int       `json:"user_id"`
	Username      string     `json:"username"`
	PlanID        *int       `json:"plan_id"`
	PlanName      *string    `json:"plan_name"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Status        string     `json:"status"`
	DueDate       *string    `json:"due_date"`
	PaidAt        *time.Time `json:"paid_at"`
	Notes         *string    `json:"notes"`
	CreatedBy     *string    `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ListInvoices returns paginated invoices with filters.
func (h *Handler) ListInvoices(c *gin.Context) {
	offset, limit := paginationParams(c)
	status := c.Query("status")
	username := c.Query("username")

	where := "WHERE 1=1"
	args := []interface{}{}
	argN := 1

	if status != "" {
		where += fmt.Sprintf(" AND i.status=$%d", argN)
		args = append(args, status)
		argN++
	}
	if username != "" {
		where += fmt.Sprintf(" AND i.username ILIKE $%d", argN)
		args = append(args, "%"+username+"%")
		argN++
	}

	var total int
	h.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM invoices i %s", where), args...).Scan(&total)

	dataArgs := append(args, limit, offset)
	rows, err := h.db.Query(fmt.Sprintf(`
		SELECT i.id, i.invoice_number, i.user_id, i.username, i.plan_id, i.plan_name,
		       i.amount, i.currency, i.status, i.due_date::text, i.paid_at, i.notes,
		       au.username, i.created_at
		FROM invoices i
		LEFT JOIN app_users au ON au.id = i.created_by
		%s
		ORDER BY i.created_at DESC
		LIMIT $%d OFFSET $%d`, where, argN, argN+1), dataArgs...)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch invoices")
		return
	}
	defer rows.Close()

	invoices := []Invoice{}
	for rows.Next() {
		var inv Invoice
		rows.Scan(&inv.ID, &inv.InvoiceNumber, &inv.UserID, &inv.Username, &inv.PlanID, &inv.PlanName,
			&inv.Amount, &inv.Currency, &inv.Status, &inv.DueDate, &inv.PaidAt, &inv.Notes,
			&inv.CreatedBy, &inv.CreatedAt)
		invoices = append(invoices, inv)
	}

	// Revenue summary
	type Summary struct {
		TotalRevenue float64 `json:"total_revenue"`
		Pending      int     `json:"pending"`
		Paid         int     `json:"paid"`
		Overdue      int     `json:"overdue"`
	}
	var sum Summary
	h.db.QueryRow(`SELECT COALESCE(SUM(amount) FILTER (WHERE status='paid'),0),
	               COUNT(*) FILTER (WHERE status='pending'),
	               COUNT(*) FILTER (WHERE status='paid'),
	               COUNT(*) FILTER (WHERE status='overdue')
	               FROM invoices`).Scan(&sum.TotalRevenue, &sum.Pending, &sum.Paid, &sum.Overdue)

	c.JSON(http.StatusOK, gin.H{"data": invoices, "total": total, "summary": sum})
}

// CreateInvoice creates a new invoice.
func (h *Handler) CreateInvoice(c *gin.Context) {
	var req struct {
		Username string  `json:"username" binding:"required"`
		PlanID   *int    `json:"plan_id"`
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
		DueDate  string  `json:"due_date"`
		Notes    string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	claims, _ := middleware.GetClaims(c)

	// Get user ID
	var userID *int
	var uid int
	if err := h.db.QueryRow(`SELECT id FROM radius_users WHERE username=$1`, req.Username).Scan(&uid); err == nil {
		userID = &uid
	}

	// Get plan name
	var planName *string
	if req.PlanID != nil {
		var pn string
		if err := h.db.QueryRow(`SELECT name FROM user_plans WHERE id=$1`, *req.PlanID).Scan(&pn); err == nil {
			planName = &pn
		}
	}

	invoiceNum := fmt.Sprintf("INV-%s-%06d", time.Now().Format("200601"), time.Now().UnixNano()%1000000)

	var dueDate interface{}
	if req.DueDate != "" {
		dueDate = req.DueDate
	}

	var id int
	err := h.db.QueryRow(`
		INSERT INTO invoices (invoice_number,user_id,username,plan_id,plan_name,amount,currency,due_date,notes,created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		invoiceNum, userID, req.Username, req.PlanID, planName, req.Amount, req.Currency,
		dueDate, nullableString(req.Notes), claims.UserID,
	).Scan(&id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to create invoice")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "invoice_number": invoiceNum, "message": "invoice created"})
}

// UpdateInvoice updates invoice status / payment.
func (h *Handler) UpdateInvoice(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Status  string  `json:"status"`
		Amount  float64 `json:"amount"`
		DueDate string  `json:"due_date"`
		Notes   string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var paidAt interface{}
	if req.Status == "paid" {
		paidAt = time.Now()
	}

	var dueDate interface{}
	if req.DueDate != "" {
		dueDate = req.DueDate
	}

	h.db.Exec(`UPDATE invoices SET status=COALESCE(NULLIF($1,''),status),
		amount=CASE WHEN $2>0 THEN $2 ELSE amount END,
		due_date=COALESCE($3::date,due_date),
		notes=COALESCE(NULLIF($4,''),notes),
		paid_at=COALESCE($5,paid_at), updated_at=NOW() WHERE id=$6`,
		req.Status, req.Amount, dueDate, req.Notes, paidAt, id)

	c.JSON(http.StatusOK, gin.H{"message": "invoice updated"})
}

// DeleteInvoice removes an invoice.
func (h *Handler) DeleteInvoice(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM invoices WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "invoice deleted"})
}
