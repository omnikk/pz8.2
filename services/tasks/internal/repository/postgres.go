package repository

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // Р В РўвЂР РЋР вЂљР В Р’В°Р В РІвЂћвЂ“Р В Р вЂ Р В Р’ВµР РЋР вЂљ Р РЋР вЂљР В Р’ВµР В РЎвЂ“Р В РЎвЂР РЋР С“Р РЋРІР‚С™Р РЋР вЂљР В РЎвЂР РЋР вЂљР РЋРЎвЂњР В Р’ВµР РЋРІР‚С™Р РЋР С“Р РЋР РЏ Р РЋРІР‚РЋР В Р’ВµР РЋР вЂљР В Р’ВµР В Р’В· init()

	"github.com/omnikk/pz8/services/tasks/internal/service"
)

type PostgresRepo struct {
	db *sql.DB
}

// NewPostgres Р В РЎвЂўР РЋРІР‚С™Р В РЎвЂќР РЋР вЂљР РЋРІР‚в„–Р В Р вЂ Р В Р’В°Р В Р’ВµР РЋРІР‚С™ Р РЋР С“Р В РЎвЂўР В Р’ВµР В РўвЂР В РЎвЂР В Р вЂ¦Р В Р’ВµР В Р вЂ¦Р В РЎвЂР В Р’Вµ Р В РЎвЂ Р В РЎвЂ”Р РЋР вЂљР В РЎвЂўР В Р вЂ Р В Р’ВµР РЋР вЂљР РЋР РЏР В Р’ВµР РЋРІР‚С™ Р В Р’ВµР В РЎвЂ“Р В РЎвЂў Р В РЎвЂ”Р В РЎвЂР В Р вЂ¦Р В РЎвЂ“Р В РЎвЂўР В РЎВ.
// Р В РЎСљР В Р’В° Р В Р вЂ Р РЋРІР‚В¦Р В РЎвЂўР В РўвЂ Р Р†Р вЂљРІР‚Сњ Р В РЎвЂ“Р В РЎвЂўР РЋРІР‚С™Р В РЎвЂўР В Р вЂ Р РЋРІР‚в„–Р В РІвЂћвЂ“ DSN (Р РЋРІР‚С›Р В РЎвЂўР РЋР вЂљР В РЎВР В Р’В°Р РЋРІР‚С™ Р РЋР С“Р В РЎВ. Р В Р вЂ  main.go).
func NewPostgres(dsn string) (*PostgresRepo, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Р В РўвЂР В РЎвЂў Р В РЎвЂ”Р В Р’ВµР РЋР вЂљР В Р вЂ Р В РЎвЂўР В РЎвЂ“Р В РЎвЂў Р РЋР вЂљР В Р’ВµР В Р’В°Р В Р’В»Р РЋР Р‰Р В Р вЂ¦Р В РЎвЂўР В РЎвЂ“Р В РЎвЂў Р В Р’В·Р В Р’В°Р В РЎвЂ”Р РЋР вЂљР В РЎвЂўР РЋР С“Р В Р’В° Р РЋР С“Р В РЎвЂўР В Р’ВµР В РўвЂР В РЎвЂР В Р вЂ¦Р В Р’ВµР В Р вЂ¦Р В РЎвЂР В Р’Вµ Р В Р вЂ¦Р В Р’Вµ Р В РЎвЂ”Р РЋР вЂљР В РЎвЂўР В Р вЂ Р В Р’ВµР РЋР вЂљР РЋР РЏР В Р’ВµР РЋРІР‚С™Р РЋР С“Р РЋР РЏ Р Р†Р вЂљРІР‚Сњ Р В РЎвЂ”Р В РЎвЂР В Р вЂ¦Р В РЎвЂ“Р РЋРЎвЂњР В Р’ВµР В РЎВ Р РЋР С“Р В Р’В°Р В РЎВР В РЎвЂ
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(10)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}
	return &PostgresRepo{db: db}, nil
}

func (r *PostgresRepo) Close() error {
	return r.db.Close()
}

// ----- CRUD -----

func (r *PostgresRepo) Create(t service.Task) (service.Task, error) {
	var dueDate any
	if t.DueDate == "" {
		dueDate = nil
	} else {
		dueDate = t.DueDate
	}

	const q = `
		INSERT INTO tasks (id, title, description, due_date, done)
		VALUES ($1, $2, $3, $4::date, $5)
		RETURNING id, title, description, COALESCE(due_date::text, ''), done`

	var out service.Task
	err := r.db.QueryRow(q, t.ID, t.Title, t.Description, dueDate, t.Done).
		Scan(&out.ID, &out.Title, &out.Description, &out.DueDate, &out.Done)
	if err != nil {
		return service.Task{}, fmt.Errorf("insert task: %w", err)
	}
	return out, nil
}

func (r *PostgresRepo) List() ([]service.Task, error) {
	const q = `
		SELECT id, title, description, COALESCE(due_date::text, ''), done
		FROM tasks
		ORDER BY created_at`

	rows, err := r.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *PostgresRepo) Get(id string) (service.Task, bool, error) {
	const q = `
		SELECT id, title, description, COALESCE(due_date::text, ''), done
		FROM tasks WHERE id = $1`

	var t service.Task
	err := r.db.QueryRow(q, id).
		Scan(&t.ID, &t.Title, &t.Description, &t.DueDate, &t.Done)
	if err == sql.ErrNoRows {
		return service.Task{}, false, nil
	}
	if err != nil {
		return service.Task{}, false, fmt.Errorf("get task: %w", err)
	}
	return t, true, nil
}

func (r *PostgresRepo) Update(id string, title *string, done *bool) (service.Task, bool, error) {
	// COALESCE($1, title) Р Р†Р вЂљРІР‚Сњ Р В Р’ВµР РЋР С“Р В Р’В»Р В РЎвЂ title == nil, Р В РЎвЂўР РЋР С“Р РЋРІР‚С™Р В Р’В°Р В Р вЂ Р В Р’В»Р РЋР РЏР В Р’ВµР В РЎВ Р РЋР С“Р РЋРІР‚С™Р В Р’В°Р РЋР вЂљР В РЎвЂўР В Р’Вµ Р В Р’В·Р В Р вЂ¦Р В Р’В°Р РЋРІР‚РЋР В Р’ВµР В Р вЂ¦Р В РЎвЂР В Р’Вµ.
	// Р В Р’В­Р РЋРІР‚С™Р В РЎвЂў Р В РЎвЂ”Р В РЎвЂўР В Р’В·Р В Р вЂ Р В РЎвЂўР В Р’В»Р РЋР РЏР В Р’ВµР РЋРІР‚С™ Р В РЎвЂўР В Р’В±Р В Р вЂ¦Р В РЎвЂўР В Р вЂ Р В Р’В»Р РЋР РЏР РЋРІР‚С™Р РЋР Р‰ Р В Р вЂ Р РЋРІР‚в„–Р В Р’В±Р В РЎвЂўР РЋР вЂљР В РЎвЂўР РЋРІР‚РЋР В Р вЂ¦Р В РЎвЂў Р В РЎвЂўР В РўвЂР В Р вЂ¦Р В РЎвЂР В РЎВ Р В Р’В·Р В Р’В°Р В РЎвЂ”Р РЋР вЂљР В РЎвЂўР РЋР С“Р В РЎвЂўР В РЎВ.
	const q = `
		UPDATE tasks
		SET title = COALESCE($2, title),
		    done  = COALESCE($3, done),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, title, description, COALESCE(due_date::text, ''), done`

	var t service.Task
	err := r.db.QueryRow(q, id, title, done).
		Scan(&t.ID, &t.Title, &t.Description, &t.DueDate, &t.Done)
	if err == sql.ErrNoRows {
		return service.Task{}, false, nil
	}
	if err != nil {
		return service.Task{}, false, fmt.Errorf("update task: %w", err)
	}
	return t, true, nil
}

func (r *PostgresRepo) Delete(id string) (bool, error) {
	res, err := r.db.Exec(`DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ----- Search: Р В Р’В±Р В Р’ВµР В Р’В·Р В РЎвЂўР В РЎвЂ”Р В Р’В°Р РЋР С“Р В Р вЂ¦Р РЋРІР‚в„–Р В РІвЂћвЂ“ Р В РЎвЂ Р РЋРЎвЂњР РЋР РЏР В Р’В·Р В Р вЂ Р В РЎвЂР В РЎВР РЋРІР‚в„–Р В РІвЂћвЂ“ -----

// Search Р Р†Р вЂљРІР‚Сњ Р В РЎвЂ”Р В Р’В°Р РЋР вЂљР В Р’В°Р В РЎВР В Р’ВµР РЋРІР‚С™Р РЋР вЂљР В РЎвЂР В Р’В·Р В РЎвЂўР В Р вЂ Р В Р’В°Р В Р вЂ¦Р В Р вЂ¦Р РЋРІР‚в„–Р В РІвЂћвЂ“ Р В Р’В·Р В Р’В°Р В РЎвЂ”Р РЋР вЂљР В РЎвЂўР РЋР С“. Р В РІР‚в„ўР В Р вЂ Р В РЎвЂўР В РўвЂ Р В РЎвЂ”Р В РЎвЂўР В Р’В»Р РЋР Р‰Р В Р’В·Р В РЎвЂўР В Р вЂ Р В Р’В°Р РЋРІР‚С™Р В Р’ВµР В Р’В»Р РЋР РЏ Р В РЎСљР В РІР‚Сћ Р В РЎвЂР В Р вЂ¦Р РЋРІР‚С™Р В Р’ВµР РЋР вЂљР В РЎвЂ”Р РЋР вЂљР В Р’ВµР РЋРІР‚С™Р В РЎвЂР РЋР вЂљР РЋРЎвЂњР В Р’ВµР РЋРІР‚С™Р РЋР С“Р РЋР РЏ
// Р В РЎвЂќР В Р’В°Р В РЎвЂќ Р РЋРІР‚РЋР В Р’В°Р РЋР С“Р РЋРІР‚С™Р РЋР Р‰ SQL: Р В РўвЂР РЋР вЂљР В Р’В°Р В РІвЂћвЂ“Р В Р вЂ Р В Р’ВµР РЋР вЂљ Р В РЎвЂ”Р В Р’ВµР РЋР вЂљР В Р’ВµР В РўвЂР В Р’В°Р РЋРІР‚ВР РЋРІР‚С™ Р В Р’ВµР В РЎвЂ“Р В РЎвЂў Р В РЎвЂќР В Р’В°Р В РЎвЂќ Р В Р’В·Р В Р вЂ¦Р В Р’В°Р РЋРІР‚РЋР В Р’ВµР В Р вЂ¦Р В РЎвЂР В Р’Вµ Р В РЎвЂ”Р В Р’В°Р РЋР вЂљР В Р’В°Р В РЎВР В Р’ВµР РЋРІР‚С™Р РЋР вЂљР В Р’В°.
func (r *PostgresRepo) Search(title string) ([]service.Task, error) {
	const q = `
		SELECT id, title, description, COALESCE(due_date::text, ''), done
		FROM tasks WHERE title = $1`

	rows, err := r.db.Query(q, title)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	return scanTasks(rows)
}

// SearchVulnerable Р Р†Р вЂљРІР‚Сњ Р В РЎСљР В РЎвЂ™Р В РЎС™Р В РІР‚СћР В Р’В Р В РІР‚СћР В РЎСљР В РЎСљР В РЎвЂє Р РЋРЎвЂњР РЋР РЏР В Р’В·Р В Р вЂ Р В РЎвЂР В РЎВР В Р’В°Р РЋР РЏ Р РЋР вЂљР В Р’ВµР В Р’В°Р В Р’В»Р В РЎвЂР В Р’В·Р В Р’В°Р РЋРІР‚В Р В РЎвЂР РЋР РЏ Р В РўвЂР В Р’В»Р РЋР РЏ Р В РўвЂР В Р’ВµР В РЎВР В РЎвЂўР В Р вЂ¦Р РЋР С“Р РЋРІР‚С™Р РЋР вЂљР В Р’В°Р РЋРІР‚В Р В РЎвЂР В РЎвЂ.
// Р В РЎв„ўР В РЎвЂўР В Р вЂ¦Р В РЎвЂќР В Р’В°Р РЋРІР‚С™Р В Р’ВµР В Р вЂ¦Р В Р’В°Р РЋРІР‚В Р В РЎвЂР РЋР РЏ Р В Р вЂ Р В Р вЂ Р В РЎвЂўР В РўвЂР В Р’В° Р В РЎвЂ”Р В РЎвЂўР В Р’В»Р РЋР Р‰Р В Р’В·Р В РЎвЂўР В Р вЂ Р В Р’В°Р РЋРІР‚С™Р В Р’ВµР В Р’В»Р РЋР РЏ Р РЋР С“ SQL = Р В РЎвЂќР В Р’В»Р В Р’В°Р РЋР С“Р РЋР С“Р В РЎвЂР РЋРІР‚РЋР В Р’ВµР РЋР С“Р В РЎвЂќР В Р’В°Р РЋР РЏ SQL-Р В РЎвЂР В Р вЂ¦Р РЋР вЂ°Р В Р’ВµР В РЎвЂќР РЋРІР‚В Р В РЎвЂР РЋР РЏ.
// Р В РЎСџР В Р’ВµР В РІвЂћвЂ“Р В Р’В»Р В РЎвЂўР В Р’В°Р В РўвЂ `' OR '1'='1` Р В РЎвЂ”Р РЋР вЂљР В Р’ВµР В Р вЂ Р РЋР вЂљР В Р’В°Р РЋРІР‚С™Р В РЎвЂР РЋРІР‚С™ Р В Р’В·Р В Р’В°Р В РЎвЂ”Р РЋР вЂљР В РЎвЂўР РЋР С“ Р В Р вЂ :
//
//	SELECT ... WHERE title = '' OR '1'='1'
//
// Р В РЎвЂ Р В Р вЂ Р В Р’ВµР РЋР вЂљР В Р вЂ¦Р РЋРІР‚ВР РЋРІР‚С™ Р В РІР‚в„ўР В Р Р‹Р В РІР‚Сћ Р РЋР С“Р РЋРІР‚С™Р РЋР вЂљР В РЎвЂўР В РЎвЂќР В РЎвЂ.
func (r *PostgresRepo) SearchVulnerable(title string) ([]service.Task, error) {
	q := "SELECT id, title, description, COALESCE(due_date::text, ''), done " +
		"FROM tasks WHERE title = '" + title + "'"

	rows, err := r.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("vulnerable search: %w", err)
	}
	defer rows.Close()
	return scanTasks(rows)
}

// scanTasks Р Р†Р вЂљРІР‚Сњ Р В РЎвЂўР В Р’В±Р РЋРІР‚В°Р В РЎвЂР В РІвЂћвЂ“ Р В РЎвЂ”Р В РЎвЂўР В РЎВР В РЎвЂўР РЋРІР‚В°Р В Р вЂ¦Р В РЎвЂР В РЎвЂќ Р В РўвЂР В Р’В»Р РЋР РЏ Р РЋР С“Р РЋРІР‚С™Р РЋР вЂљР В РЎвЂўР В РЎвЂќ-Р РЋР вЂљР В Р’ВµР В Р’В·Р РЋРЎвЂњР В Р’В»Р РЋР Р‰Р РЋРІР‚С™Р В Р’В°Р РЋРІР‚С™Р В РЎвЂўР В Р вЂ .
func scanTasks(rows *sql.Rows) ([]service.Task, error) {
	result := make([]service.Task, 0)
	for rows.Next() {
		var t service.Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.DueDate, &t.Done); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return result, nil
}
