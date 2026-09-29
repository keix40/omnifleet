-- TimescaleDB hypertable for GPS history (extension pre-enabled in timescale image).

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

SELECT create_hypertable('gps_positions', 'recorded_at', if_not_exists => TRUE);
