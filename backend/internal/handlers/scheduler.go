package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/freeradius-manager/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ScheduledTask represents an automated background job.
type ScheduledTask struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	TaskType   string     `json:"task_type"`
	Schedule   string     `json:"schedule"`
	IsActive   bool       `json:"is_active"`
	LastRun    *time.Time `json:"last_run"`
	NextRun    *time.Time `json:"next_run"`
	LastResult *string    `json:"last_result"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ListScheduledTasks returns all configured tasks.
func (h *Handler) ListScheduledTasks(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, task_type, schedule, is_active, last_run, next_run, last_result, created_at
		FROM scheduled_tasks ORDER BY id`)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "failed to fetch tasks")
		return
	}
	defer rows.Close()

	tasks := []ScheduledTask{}
	for rows.Next() {
		var t ScheduledTask
		rows.Scan(&t.ID, &t.Name, &t.TaskType, &t.Schedule, &t.IsActive, &t.LastRun, &t.NextRun, &t.LastResult, &t.CreatedAt)
		tasks = append(tasks, t)
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

// ToggleTask enables or disables a scheduled task.
func (h *Handler) ToggleTask(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	h.db.Exec(`UPDATE scheduled_tasks SET is_active = NOT is_active WHERE id=$1`, id)
	c.JSON(http.StatusOK, gin.H{"message": "task toggled"})
}

// RunTaskNow forces immediate execution of a task.
func (h *Handler) RunTaskNow(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var taskType string
	if err := h.db.QueryRow(`SELECT task_type FROM scheduled_tasks WHERE id=$1`, id).Scan(&taskType); err != nil {
		respondError(c, http.StatusNotFound, "task not found")
		return
	}

	result, count := runTask(h.db, taskType)
	h.db.Exec(`UPDATE scheduled_tasks SET last_run=NOW(), last_result=$1 WHERE id=$2`,
		fmt.Sprintf("%s: affected %d rows", result, count), id)

	c.JSON(http.StatusOK, gin.H{
		"message":  result,
		"affected": count,
	})
}

// UpdateTaskSchedule changes the schedule of a task.
func (h *Handler) UpdateTaskSchedule(c *gin.Context) {
	id, err := mustInt(c, "id")
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid ID")
		return
	}
	var req struct {
		Schedule string `json:"schedule" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	nextRun := nextRunTime(req.Schedule)
	h.db.Exec(`UPDATE scheduled_tasks SET schedule=$1, next_run=$2 WHERE id=$3`, req.Schedule, nextRun, id)
	c.JSON(http.StatusOK, gin.H{"message": "schedule updated", "next_run": nextRun})
}

// StartScheduler launches the background task runner.
func StartScheduler(db *database.DB, log *logrus.Logger) {
	go func() {
		log.Info("Scheduler started — checking tasks every 10 minutes")
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		// Run once at startup
		runDueTasks(db, log)

		for range ticker.C {
			runDueTasks(db, log)
		}
	}()
}

func runDueTasks(db *database.DB, log *logrus.Logger) {
	rows, err := db.Query(`
		SELECT id, task_type, schedule FROM scheduled_tasks
		WHERE is_active=true AND (next_run IS NULL OR next_run <= NOW())`)
	if err != nil {
		return
	}
	defer rows.Close()

	type task struct {
		id       int
		taskType string
		schedule string
	}
	tasks := []task{}
	for rows.Next() {
		var t task
		rows.Scan(&t.id, &t.taskType, &t.schedule)
		tasks = append(tasks, t)
	}

	for _, t := range tasks {
		result, count := runTask(db, t.taskType)
		nextRun := nextRunTime(t.schedule)
		db.Exec(`UPDATE scheduled_tasks SET last_run=NOW(), next_run=$1, last_result=$2 WHERE id=$3`,
			nextRun, fmt.Sprintf("%s (%d rows)", result, count), t.id)
		log.Infof("Scheduler: ran '%s' — %s (%d rows)", t.taskType, result, count)
	}
}

// runTask executes the logic for a given task type.
// Returns a description and number of affected rows.
func runTask(db *database.DB, taskType string) (string, int) {
	switch taskType {

	case "expire_accounts":
		res, err := db.Exec(`
			UPDATE radius_users SET status='expired', updated_at=NOW()
			WHERE status='active'
			  AND account_expiry IS NOT NULL
			  AND account_expiry < CURRENT_DATE`)
		if err != nil {
			return "expire_accounts failed: " + err.Error(), 0
		}
		n, _ := res.RowsAffected()
		return "expired accounts", int(n)

	case "mark_overdue_invoices":
		res, err := db.Exec(`
			UPDATE invoices SET status='overdue', updated_at=NOW()
			WHERE status='pending'
			  AND due_date IS NOT NULL
			  AND due_date < CURRENT_DATE`)
		if err != nil {
			return "mark_overdue failed: " + err.Error(), 0
		}
		n, _ := res.RowsAffected()
		return "invoices marked overdue", int(n)

	case "cleanup_sessions":
		// Close radacct sessions that haven't had an update in >24h (stale)
		res, err := db.Exec(`
			UPDATE radacct SET acctstoptime=NOW(),
			       acctterminatecause='Session-Timeout'
			WHERE acctstoptime IS NULL
			  AND acctupdatetime < NOW() - INTERVAL '24 hours'`)
		if err != nil {
			return "cleanup_sessions failed: " + err.Error(), 0
		}
		n, _ := res.RowsAffected()
		return "stale sessions closed", int(n)

	case "daily_report":
		// Count key stats for the report
		var users, sessions, traffic int
		db.QueryRow(`SELECT COUNT(*) FROM radius_users WHERE status='active'`).Scan(&users)
		db.QueryRow(`SELECT COUNT(*) FROM radacct WHERE acctstoptime IS NULL`).Scan(&sessions)
		db.QueryRow(`SELECT COALESCE(SUM(acctinputoctets+acctoutputoctets),0) FROM radacct WHERE acctstarttime >= CURRENT_DATE`).Scan(&traffic)
		return fmt.Sprintf("daily report: %d active users, %d sessions, %d bytes today", users, sessions, traffic), 1

	case "renew_plans":
		// Find users with expired plans and valid billing — extend for another period
		res, err := db.Exec(`
			UPDATE radius_users ru SET
			  account_expiry = CURRENT_DATE + up.validity_days,
			  updated_at     = NOW()
			FROM user_plans up
			WHERE ru.plan_id = up.id
			  AND ru.status  = 'active'
			  AND ru.account_expiry BETWEEN CURRENT_DATE-1 AND CURRENT_DATE+1`)
		if err != nil {
			return "renew_plans failed: " + err.Error(), 0
		}
		n, _ := res.RowsAffected()
		return "plans renewed", int(n)
	}

	return "unknown task", 0
}

func nextRunTime(schedule string) time.Time {
	switch schedule {
	case "hourly":
		return time.Now().Add(1 * time.Hour)
	case "weekly":
		return time.Now().AddDate(0, 0, 7)
	default: // daily
		return time.Now().AddDate(0, 0, 1)
	}
}
