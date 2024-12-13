package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type TimerEntry struct {
	ID        int       `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	IsRunning bool      `json:"is_running"`
	Duration  string    `json:"duration,omitempty"`
}

type TimerServer struct {
	db *sql.DB
}

func NewTimerServer(connStr string) *TimerServer {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Не удалось установить соединение с базой данных: %v", err)
	}

	return &TimerServer{db: db}
}

func (ts *TimerServer) StartTimer(employeeID string) error {
	// Сначала проверяем/создаем сотрудника
	_, err := ts.db.Exec(`
		INSERT INTO employees (employee_id) 
		VALUES ($1) 
		ON CONFLICT (employee_id) DO NOTHING
	`, employeeID)
	if err != nil {
		return fmt.Errorf("ошибка регистрации сотрудника: %v", err)
	}

	// Проверяем активные таймеры
	var activeTimers int
	err = ts.db.QueryRow(`
		SELECT COUNT(*) FROM timer_entries 
		WHERE employee_id = $1 AND is_running = true
	`, employeeID).Scan(&activeTimers)
	if err != nil {
		return fmt.Errorf("ошибка проверки активных таймеров: %v", err)
	}

	if activeTimers > 0 {
		return fmt.Errorf("таймер для сотрудника %s уже запущен", employeeID)
	}

	// Создаем новую запись таймера
	_, err = ts.db.Exec(`
		INSERT INTO timer_entries (employee_id, start_time, is_running) 
		VALUES ($1, NOW(), true)
	`, employeeID)
	if err != nil {
		return fmt.Errorf("ошибка запуска таймера: %v", err)
	}

	return nil
}

func (ts *TimerServer) StopTimer(employeeID string) error {
	// Останавливаем последний активный таймер
	result, err := ts.db.Exec(`
		UPDATE timer_entries 
		SET 
			end_time = NOW(), 
			is_running = false,
			duration = NOW() - start_time
		WHERE 
			employee_id = $1 AND 
			is_running = true
	`, employeeID)
	if err != nil {
		return fmt.Errorf("ошибка остановки таймера: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка определения количества измененных строк: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("нет активных таймеров для сотрудника %s", employeeID)
	}

	return nil
}

func (ts *TimerServer) GetTimerInfo(employeeID string) ([]TimerEntry, error) {
	rows, err := ts.db.Query(`
		SELECT 
			id, 
			start_time, 
			COALESCE(end_time, '1970-01-01'::timestamp), 
			is_running,
			COALESCE(to_char(duration, 'HH24:MI:SS'), '')
		FROM timer_entries 
		WHERE employee_id = $1 
		ORDER BY start_time DESC
	`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о таймерах: %v", err)
	}
	defer rows.Close()

	var entries []TimerEntry
	for rows.Next() {
		var entry TimerEntry
		var endTime time.Time
		err := rows.Scan(
			&entry.ID,
			&entry.StartTime,
			&endTime,
			&entry.IsRunning,
			&entry.Duration,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования результата: %v", err)
		}

		if !endTime.IsZero() && endTime.Year() != 1970 {
			entry.EndTime = endTime
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (ts *TimerServer) handleStart(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	if employeeID == "" {
		http.Error(w, "требуется ID сотрудника", http.StatusBadRequest)
		return
	}

	err := ts.StartTimer(employeeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "таймер запущен"})
}

func (ts *TimerServer) handleStop(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	if employeeID == "" {
		http.Error(w, "требуется ID сотрудника", http.StatusBadRequest)
		return
	}

	err := ts.StopTimer(employeeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "таймер остановлен"})
}

func (ts *TimerServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	if employeeID == "" {
		http.Error(w, "требуется ID сотрудника", http.StatusBadRequest)
		return
	}

	timerInfo, err := ts.GetTimerInfo(employeeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": timerInfo,
	})
}

func main() {
	// Параметры подключения к базе можно вынести в переменные окружения
	connStr := "postgres://username:password@localhost/timerdb?sslmode=disable"

	// Проверка переменных окружения
	if pgConnStr := os.Getenv("DATABASE_URL"); pgConnStr != "" {
		connStr = pgConnStr
	}

	ts := NewTimerServer(connStr)
	defer ts.db.Close()

	http.HandleFunc("/start", ts.handleStart)
	http.HandleFunc("/stop", ts.handleStop)
	http.HandleFunc("/info", ts.handleInfo)

	fmt.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
