-- Optional Timescale hypertable for Neon/plain Postgres (Apache-2 Timescale features only; no compression).

DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS timescaledb;
    PERFORM create_hypertable('gps_positions', 'recorded_at', if_not_exists => TRUE);
EXCEPTION
    WHEN OTHERS THEN
        RAISE NOTICE 'omnifleet: Timescale hypertable skipped (%). gps_positions remains a regular table.', SQLERRM;
END
$$;
