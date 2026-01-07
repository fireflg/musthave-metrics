-- migrations/000001_create_metrcs_table.down.sql
-- Откат создания таблицы метрик
DROP INDEX IF EXISTS idx_metrics_name;
DROP TABLE IF EXISTS metrics;