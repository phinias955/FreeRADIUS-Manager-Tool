package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Payments
// ─────────────────────────────────────────────────────────────────────────────

type Payment struct {
	ID            int        `json:"id"`
	InvoiceID     *int       `json:"invoice_id"`
	InvoiceNumber *string    `json:"invoice_number"`
	Username      *string    `json:"username"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	PaymentMethod string     `json:"payment_method"`
	Gateway       *string    `json:"gateway"`
	GatewayRef    *string    `json:"gateway_ref"`
	Status        string     `json:"status"`
	Notes         *string    `json:"notes"`
	ReceiptNumber string     `json:"receipt_number"`
	ProcessedBy   *string    `json:"processed_by"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ListPayments returns paginated payments with optional filters.
func (h *Handler) ListPayments(c *gin.Context) {
	offset, limit := paginationParams(c)
	username := c.Query("username")
	method := c.Query("method")

	rows, err := h.db.Query(`
		SELECT p.id, p.invoice_id, i.invoice_number, p.username,
		       p.amount, p.currency, p.payment_method, p.gateway,
		       p.gateway_ref, p.status, p.notes, p.receipt_number,
		       u.username AS processed_by_name, p.created_at
		FROM payments p
		LEFT JOIN invoices i ON i.id = p.invoice_id
		LEFT JOIN app_users u ON u.id = p.processed_by
		WHERE ($1='' OR p.username=$1)
		  AND ($2='' OR p.payment_method=$2)
		ORDER BY p.created_at DESC
		LIMIT $3 OFFSET $4`,
		username, method, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch payments")
		return
	}
	defer rows.Close()

	payments := []Payment{}
	for rows.Next() {
		var p Payment
		rows.Scan(&p.ID, &p.InvoiceID, &p.InvoiceNumber, &p.Username,
			&p.Amount, &p.Currency, &p.PaymentMethod, &p.Gateway,
			&p.GatewayRef, &p.Status, &p.Notes, &p.ReceiptNumber,
			&p.ProcessedBy, &p.CreatedAt)
		payments = append(payments, p)
	}

	var total int
	var totalAmount float64
	h.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(amount),0) FROM payments
		WHERE ($1='' OR username=$1) AND status='confirmed'`, username).
		Scan(&total, &totalAmount)

	c.JSON(http.StatusOK, gin.H{
		"data":         payments,
		"total":        total,
		"total_amount": totalAmount,
	})
}

// CreatePayment records a new payment and marks the linked invoice as paid.
func (h *Handler) CreatePayment(c *gin.Context) {
	var req struct {
		InvoiceID     *int    `json:"invoice_id"`
		Username      string  `json:"username"`
		Amount        float64 `json:"amount" binding:"required,gt=0"`
		Currency      string  `json:"currency"`
		PaymentMethod string  `json:"payment_method"`
		Gateway       string  `json:"gateway"`
		GatewayRef    string  `json:"gateway_ref"`
		Notes         string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "cash"
	}

	receipt := generateReceiptNumber()

	var id int
	err := h.db.QueryRow(`
		INSERT INTO payments (invoice_id, username, amount, currency, payment_method,
		                      gateway, gateway_ref, notes, receipt_number, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'confirmed') RETURNING id`,
		req.InvoiceID, nullableString(req.Username), req.Amount, req.Currency,
		req.PaymentMethod, nullableString(req.Gateway), nullableString(req.GatewayRef),
		nullableString(req.Notes), receipt).Scan(&id)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to record payment")
		return
	}

	// Mark linked invoice as paid
	if req.InvoiceID != nil {
		h.db.Exec(`UPDATE invoices SET status='paid', paid_at=NOW() WHERE id=$1`, *req.InvoiceID)
	}

	go h.dispatchWebhook("invoice.paid", gin.H{
		"payment_id": id, "username": req.Username,
		"amount": req.Amount, "receipt": receipt,
	})

	c.JSON(http.StatusCreated, gin.H{
		"id":             id,
		"receipt_number": receipt,
		"message":        "payment recorded",
	})
}

// GetPaymentReceipt returns HTML receipt for a payment.
func (h *Handler) GetPaymentReceipt(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var p Payment
	var orgName, orgEmail string
	h.db.QueryRow(`
		SELECT p.id, p.invoice_id, i.invoice_number, p.username,
		       p.amount, p.currency, p.payment_method, p.gateway,
		       p.gateway_ref, p.status, p.notes, p.receipt_number,
		       u.username, p.created_at
		FROM payments p
		LEFT JOIN invoices i ON i.id = p.invoice_id
		LEFT JOIN app_users u ON u.id = p.processed_by
		WHERE p.id=$1`, id).
		Scan(&p.ID, &p.InvoiceID, &p.InvoiceNumber, &p.Username,
			&p.Amount, &p.Currency, &p.PaymentMethod, &p.Gateway,
			&p.GatewayRef, &p.Status, &p.Notes, &p.ReceiptNumber,
			&p.ProcessedBy, &p.CreatedAt)

	// Get org name from settings
	h.db.QueryRow(`SELECT COALESCE(value,'RADIUS Manager ISP') FROM system_settings WHERE key='org_name'`).Scan(&orgName)
	h.db.QueryRow(`SELECT COALESCE(value,'') FROM system_settings WHERE key='org_email'`).Scan(&orgEmail)

	if p.ReceiptNumber == "" {
		respondError(c, http.StatusNotFound, "payment not found")
		return
	}

	html := buildReceiptHTML(p, orgName, orgEmail)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// DeletePayment removes a payment record.
func (h *Handler) DeletePayment(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM payments WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "payment deleted"})
}

// PaymentSummary returns revenue analytics by period.
func (h *Handler) PaymentSummary(c *gin.Context) {
	type PeriodStat struct {
		Period   string  `json:"period"`
		Total    float64 `json:"total"`
		Count    int     `json:"count"`
	}

	// Monthly revenue (last 12 months)
	monthly := []PeriodStat{}
	rows, _ := h.db.Query(`
		SELECT TO_CHAR(created_at,'YYYY-MM') AS period,
		       SUM(amount), COUNT(*)
		FROM payments
		WHERE status='confirmed' AND created_at >= NOW()-INTERVAL '12 months'
		GROUP BY period ORDER BY period`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var s PeriodStat
			rows.Scan(&s.Period, &s.Total, &s.Count)
			monthly = append(monthly, s)
		}
	}

	// Method breakdown
	type MethodStat struct {
		Method string  `json:"method"`
		Total  float64 `json:"total"`
		Count  int     `json:"count"`
	}
	methods := []MethodStat{}
	mrows, _ := h.db.Query(`
		SELECT payment_method, SUM(amount), COUNT(*)
		FROM payments WHERE status='confirmed'
		GROUP BY payment_method ORDER BY SUM(amount) DESC`)
	if mrows != nil {
		defer mrows.Close()
		for mrows.Next() {
			var s MethodStat
			mrows.Scan(&s.Method, &s.Total, &s.Count)
			methods = append(methods, s)
		}
	}

	var totalAll, totalMonth float64
	var countAll, countMonth int
	h.db.QueryRow(`SELECT COALESCE(SUM(amount),0), COUNT(*) FROM payments WHERE status='confirmed'`).Scan(&totalAll, &countAll)
	h.db.QueryRow(`SELECT COALESCE(SUM(amount),0), COUNT(*) FROM payments WHERE status='confirmed' AND created_at >= DATE_TRUNC('month',NOW())`).Scan(&totalMonth, &countMonth)

	c.JSON(http.StatusOK, gin.H{
		"monthly":       monthly,
		"by_method":     methods,
		"total_all":     totalAll,
		"count_all":     countAll,
		"total_month":   totalMonth,
		"count_month":   countMonth,
	})
}

func generateReceiptNumber() string {
	t := time.Now()
	r := rand.Intn(9000) + 1000
	return fmt.Sprintf("RCP-%d%02d%02d-%04d", t.Year(), t.Month(), t.Day(), r)
}

func buildReceiptHTML(p Payment, orgName, orgEmail string) string {
	username := "—"
	if p.Username != nil {
		username = *p.Username
	}
	invoice := "—"
	if p.InvoiceNumber != nil {
		invoice = *p.InvoiceNumber
	}
	notes := ""
	if p.Notes != nil {
		notes = fmt.Sprintf(`<tr><td style="color:#6b7280;">Notes</td><td style="text-align:right;">%s</td></tr>`, *p.Notes)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<title>Receipt %s</title>
<style>
  body{font-family:-apple-system,sans-serif;max-width:480px;margin:40px auto;padding:20px;color:#1e293b}
  .header{text-align:center;padding:24px 0;border-bottom:2px solid #e2e8f0;margin-bottom:24px}
  .logo{font-size:24px;font-weight:800;color:#3b82f6}
  .receipt-no{font-size:13px;color:#64748b;margin-top:4px}
  table{width:100%%;border-collapse:collapse}
  td{padding:10px 4px;border-bottom:1px solid #f1f5f9;font-size:14px}
  .total{font-size:20px;font-weight:700;color:#16a34a;text-align:right}
  .footer{text-align:center;margin-top:32px;padding-top:16px;border-top:1px solid #e2e8f0;color:#94a3b8;font-size:12px}
  .badge{display:inline-block;padding:4px 12px;background:#dcfce7;color:#16a34a;border-radius:20px;font-size:12px;font-weight:600}
  @media print{body{margin:0}}
</style>
</head><body>
<div class="header">
  <div class="logo">%s</div>
  <div class="receipt-no">PAYMENT RECEIPT</div>
  <div style="font-size:13px;color:#94a3b8;margin-top:4px">%s</div>
</div>
<table>
  <tr><td style="color:#6b7280;">Receipt No.</td><td style="text-align:right;font-weight:700;font-family:monospace">%s</td></tr>
  <tr><td style="color:#6b7280;">Date</td><td style="text-align:right;">%s</td></tr>
  <tr><td style="color:#6b7280;">Customer</td><td style="text-align:right;">%s</td></tr>
  <tr><td style="color:#6b7280;">Invoice</td><td style="text-align:right;">%s</td></tr>
  <tr><td style="color:#6b7280;">Payment Method</td><td style="text-align:right;text-transform:capitalize;">%s</td></tr>
  %s
  <tr><td style="font-weight:600;padding-top:16px;">Amount Paid</td><td class="total">%s %.2f</td></tr>
</table>
<div style="text-align:center;margin-top:24px;"><span class="badge">✓ Payment Confirmed</span></div>
<div class="footer">
  <p>%s</p>
  <p style="margin-top:4px">Thank you for your payment!</p>
  <p style="margin-top:8px"><button onclick="window.print()" style="background:#3b82f6;color:#fff;border:none;padding:8px 20px;border-radius:8px;cursor:pointer;">🖨 Print</button></p>
</div>
</body></html>`,
		p.ReceiptNumber, orgName, orgEmail,
		p.ReceiptNumber,
		p.CreatedAt.Format("Jan 2, 2006 15:04"),
		username, invoice,
		strings.Title(strings.ReplaceAll(p.PaymentMethod, "_", " ")),
		notes,
		p.Currency, p.Amount,
		orgName)
}
