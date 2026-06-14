package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────────────────────
// Customers CRM
// ─────────────────────────────────────────────────────────────────────────────

type Customer struct {
	ID            int        `json:"id"`
	Username      *string    `json:"username"`
	FullName      *string    `json:"full_name"`
	Email         *string    `json:"email"`
	Phone         *string    `json:"phone"`
	IDNumber      *string    `json:"id_number"`
	Address       *string    `json:"address"`
	City          *string    `json:"city"`
	Country       string     `json:"country"`
	OrgID         *int       `json:"org_id"`
	ContractStart *string    `json:"contract_start"`
	ContractEnd   *string    `json:"contract_end"`
	Notes         *string    `json:"notes"`
	OpenTickets   int        `json:"open_tickets"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ListCustomers returns all customers with ticket count.
func (h *Handler) ListCustomers(c *gin.Context) {
	search := c.Query("search")
	offset, limit := paginationParams(c)

	query := `SELECT c.id, c.username, c.full_name, c.email, c.phone,
		c.id_number, c.address, c.city, COALESCE(c.country,'Zimbabwe'),
		c.org_id, c.contract_start::text, c.contract_end::text, c.notes, c.created_at,
		COUNT(t.id) FILTER (WHERE t.status NOT IN ('resolved','closed')) AS open_tickets
		FROM customers c
		LEFT JOIN tickets t ON t.customer_id = c.id
		WHERE ($1='' OR c.full_name ILIKE '%'||$1||'%'
		             OR c.email ILIKE '%'||$1||'%'
		             OR c.username ILIKE '%'||$1||'%'
		             OR c.phone ILIKE '%'||$1||'%')
		GROUP BY c.id
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := h.db.Query(query, search, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch customers")
		return
	}
	defer rows.Close()

	customers := []Customer{}
	for rows.Next() {
		var cust Customer
		rows.Scan(&cust.ID, &cust.Username, &cust.FullName, &cust.Email, &cust.Phone,
			&cust.IDNumber, &cust.Address, &cust.City, &cust.Country,
			&cust.OrgID, &cust.ContractStart, &cust.ContractEnd, &cust.Notes, &cust.CreatedAt,
			&cust.OpenTickets)
		customers = append(customers, cust)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM customers WHERE ($1='' OR full_name ILIKE '%'||$1||'%' OR email ILIKE '%'||$1||'%')`, search).Scan(&total)

	c.JSON(http.StatusOK, gin.H{"data": customers, "total": total})
}

// GetCustomer returns a single customer with usage stats and tickets.
func (h *Handler) GetCustomer(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}

	var cust Customer
	err = h.db.QueryRow(`SELECT c.id, c.username, c.full_name, c.email, c.phone,
		c.id_number, c.address, c.city, COALESCE(c.country,'Zimbabwe'),
		c.org_id, c.contract_start::text, c.contract_end::text, c.notes, c.created_at,
		0 FROM customers c WHERE c.id=$1`, id).
		Scan(&cust.ID, &cust.Username, &cust.FullName, &cust.Email, &cust.Phone,
			&cust.IDNumber, &cust.Address, &cust.City, &cust.Country,
			&cust.OrgID, &cust.ContractStart, &cust.ContractEnd, &cust.Notes, &cust.CreatedAt,
			&cust.OpenTickets)
	if err != nil {
		respondError(c, http.StatusNotFound, "customer not found")
		return
	}

	// Tickets
	tickets := listTicketsForCustomer(h, id)

	// Usage (last 30 days)
	type Usage struct {
		TotalMB   float64 `json:"total_mb"`
		Sessions  int     `json:"sessions"`
		PaidTotal float64 `json:"paid_total"`
	}
	var usage Usage
	if cust.Username != nil {
		h.db.QueryRow(`SELECT
			COALESCE(SUM(acctinputoctets+acctoutputoctets),0)/(1024.0*1024.0), COUNT(*)
			FROM radacct WHERE username=$1 AND acctstarttime >= NOW()-INTERVAL '30 days'`,
			*cust.Username).Scan(&usage.TotalMB, &usage.Sessions)
		h.db.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM invoices WHERE username=$1 AND status='paid'`,
			*cust.Username).Scan(&usage.PaidTotal)
	}

	c.JSON(http.StatusOK, gin.H{"customer": cust, "tickets": tickets, "usage": usage})
}

// CreateCustomer creates a new CRM customer record.
func (h *Handler) CreateCustomer(c *gin.Context) {
	var req struct {
		Username      string `json:"username"`
		FullName      string `json:"full_name" binding:"required"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		IDNumber      string `json:"id_number"`
		Address       string `json:"address"`
		City          string `json:"city"`
		Country       string `json:"country"`
		OrgID         *int   `json:"org_id"`
		ContractStart string `json:"contract_start"`
		ContractEnd   string `json:"contract_end"`
		Notes         string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Country == "" {
		req.Country = "Zimbabwe"
	}
	var id int
	err := h.db.QueryRow(`
		INSERT INTO customers (username,full_name,email,phone,id_number,address,city,country,org_id,contract_start,contract_end,notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,
		        NULLIF($10,'')::date, NULLIF($11,'')::date,$12)
		RETURNING id`,
		nullableString(req.Username), req.FullName,
		nullableString(req.Email), nullableString(req.Phone),
		nullableString(req.IDNumber), nullableString(req.Address),
		nullableString(req.City), req.Country, req.OrgID,
		req.ContractStart, req.ContractEnd, nullableString(req.Notes)).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			respondError(c, http.StatusConflict, "customer with that username already exists")
			return
		}
		respondError(c, http.StatusInternalServerError, fmt.Sprintf("failed: %v", err))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "customer created"})
}

// UpdateCustomer updates CRM details.
func (h *Handler) UpdateCustomer(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		FullName      string `json:"full_name"`
		Email         string `json:"email"`
		Phone         string `json:"phone"`
		IDNumber      string `json:"id_number"`
		Address       string `json:"address"`
		City          string `json:"city"`
		Country       string `json:"country"`
		ContractStart string `json:"contract_start"`
		ContractEnd   string `json:"contract_end"`
		Notes         string `json:"notes"`
		OrgID         *int   `json:"org_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	h.db.Exec(`UPDATE customers SET
		full_name=COALESCE(NULLIF($1,''),full_name),
		email=COALESCE(NULLIF($2,''),email),
		phone=COALESCE(NULLIF($3,''),phone),
		id_number=COALESCE(NULLIF($4,''),id_number),
		address=COALESCE(NULLIF($5,''),address),
		city=COALESCE(NULLIF($6,''),city),
		country=COALESCE(NULLIF($7,''),country),
		contract_start=COALESCE(NULLIF($8,'')::date,contract_start),
		contract_end=COALESCE(NULLIF($9,'')::date,contract_end),
		notes=COALESCE(NULLIF($10,''),notes),
		org_id=COALESCE($11,org_id),
		updated_at=NOW()
		WHERE id=$12`,
		req.FullName, req.Email, req.Phone, req.IDNumber, req.Address,
		req.City, req.Country, req.ContractStart, req.ContractEnd, req.Notes, req.OrgID, id)
	c.JSON(http.StatusOK, gin.H{"message": "customer updated"})
}

// DeleteCustomer removes a CRM record.
func (h *Handler) DeleteCustomer(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM customers WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "customer deleted"})
}

// ─────────────────────────────────────────────────────────────────────────────
// Support Tickets
// ─────────────────────────────────────────────────────────────────────────────

type Ticket struct {
	ID          int        `json:"id"`
	CustomerID  *int       `json:"customer_id"`
	CustomerName *string   `json:"customer_name"`
	Username    *string    `json:"username"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	AssignedTo  *int       `json:"assigned_to"`
	AssigneeName *string   `json:"assignee_name"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListTickets returns all tickets (filterable by status/priority).
func (h *Handler) ListTickets(c *gin.Context) {
	statusFilter := c.Query("status")
	priorityFilter := c.Query("priority")
	offset, limit := paginationParams(c)

	rows, err := h.db.Query(`
		SELECT t.id, t.customer_id, c.full_name, t.username,
		       t.title, t.description, t.status, t.priority,
		       t.assigned_to, u.full_name, t.resolved_at, t.created_at, t.updated_at
		FROM tickets t
		LEFT JOIN customers c ON c.id = t.customer_id
		LEFT JOIN app_users u ON u.id = t.assigned_to
		WHERE ($1='' OR t.status=$1)
		  AND ($2='' OR t.priority=$2)
		ORDER BY
		  CASE t.priority WHEN 'urgent' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
		  t.created_at DESC
		LIMIT $3 OFFSET $4`,
		statusFilter, priorityFilter, limit, offset)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch tickets")
		return
	}
	defer rows.Close()

	tickets := []Ticket{}
	for rows.Next() {
		var t Ticket
		rows.Scan(&t.ID, &t.CustomerID, &t.CustomerName, &t.Username,
			&t.Title, &t.Description, &t.Status, &t.Priority,
			&t.AssignedTo, &t.AssigneeName, &t.ResolvedAt, &t.CreatedAt, &t.UpdatedAt)
		tickets = append(tickets, t)
	}

	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM tickets WHERE ($1='' OR status=$1) AND ($2='' OR priority=$2)`,
		statusFilter, priorityFilter).Scan(&total)

	// Counts by status
	type StatusCount struct {
		Open       int `json:"open"`
		InProgress int `json:"in_progress"`
		Resolved   int `json:"resolved"`
		Closed     int `json:"closed"`
	}
	var sc StatusCount
	h.db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE status='open'),
		COUNT(*) FILTER (WHERE status='in_progress'),
		COUNT(*) FILTER (WHERE status='resolved'),
		COUNT(*) FILTER (WHERE status='closed')
		FROM tickets`).Scan(&sc.Open, &sc.InProgress, &sc.Resolved, &sc.Closed)

	c.JSON(http.StatusOK, gin.H{"data": tickets, "total": total, "counts": sc})
}

// CreateTicket opens a new support ticket.
func (h *Handler) CreateTicket(c *gin.Context) {
	var req struct {
		CustomerID  *int   `json:"customer_id"`
		Username    string `json:"username"`
		Title       string `json:"title" binding:"required,min=3"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		AssignedTo  *int   `json:"assigned_to"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	var id int
	h.db.QueryRow(`INSERT INTO tickets (customer_id,username,title,description,priority,assigned_to)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		req.CustomerID, nullableString(req.Username), req.Title,
		nullableString(req.Description), req.Priority, req.AssignedTo).Scan(&id)

	go h.dispatchWebhook("ticket.created", gin.H{"id": id, "title": req.Title, "priority": req.Priority})

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "ticket created"})
}

// UpdateTicket changes status, priority, or assignee.
func (h *Handler) UpdateTicket(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Status      string `json:"status"`
		Priority    string `json:"priority"`
		AssignedTo  *int   `json:"assigned_to"`
		Description string `json:"description"`
		Title       string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	resolvedAt := "NULL"
	if req.Status == "resolved" || req.Status == "closed" {
		resolvedAt = "NOW()"
	}
	h.db.Exec(fmt.Sprintf(`UPDATE tickets SET
		status=COALESCE(NULLIF($1,''),status),
		priority=COALESCE(NULLIF($2,''),priority),
		assigned_to=COALESCE($3,assigned_to),
		title=COALESCE(NULLIF($4,''),title),
		description=COALESCE(NULLIF($5,''),description),
		resolved_at=CASE WHEN $1 IN ('resolved','closed') THEN NOW() ELSE %s END,
		updated_at=NOW()
		WHERE id=$6`, resolvedAt),
		req.Status, req.Priority, req.AssignedTo, req.Title, req.Description, id)

	c.JSON(http.StatusOK, gin.H{"message": "ticket updated"})
}

// DeleteTicket removes a ticket.
func (h *Handler) DeleteTicket(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`DELETE FROM tickets WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "ticket deleted"})
}

func listTicketsForCustomer(h *Handler, customerID int) []Ticket {
	rows, _ := h.db.Query(`SELECT t.id, t.customer_id, NULL::text, t.username,
		t.title, t.description, t.status, t.priority,
		t.assigned_to, u.full_name, t.resolved_at, t.created_at, t.updated_at
		FROM tickets t
		LEFT JOIN app_users u ON u.id = t.assigned_to
		WHERE t.customer_id=$1
		ORDER BY t.created_at DESC LIMIT 20`, customerID)
	tickets := []Ticket{}
	if rows == nil {
		return tickets
	}
	defer rows.Close()
	for rows.Next() {
		var t Ticket
		rows.Scan(&t.ID, &t.CustomerID, &t.CustomerName, &t.Username,
			&t.Title, &t.Description, &t.Status, &t.Priority,
			&t.AssignedTo, &t.AssigneeName, &t.ResolvedAt, &t.CreatedAt, &t.UpdatedAt)
		tickets = append(tickets, t)
	}
	return tickets
}
