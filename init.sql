-- Создаем схему
CREATE SCHEMA IF NOT EXISTS timer_app;

-- Таблица сотрудников
CREATE TABLE IF NOT EXISTS timer_app.employees (
    id SERIAL PRIMARY KEY,
    employee_id VARCHAR(100) UNIQUE NOT NULL
);

-- Таблица записей таймера
CREATE TABLE IF NOT EXISTS timer_app.timer_entries (
    id SERIAL PRIMARY KEY,
    employee_id VARCHAR(100) NOT NULL,
    start_time TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    end_time TIMESTAMP WITHOUT TIME ZONE,
    is_running BOOLEAN DEFAULT TRUE,
    duration INTERVAL,
    FOREIGN KEY (employee_id) REFERENCES timer_app.employees(employee_id)
);

-- Индекс для оптимизации поиска
CREATE INDEX IF NOT EXISTS idx_timer_entries_employee ON timer_app.timer_entries(employee_id);
