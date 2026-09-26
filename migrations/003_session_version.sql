-- Migration 003: server-side session invalidation (SEC-09 / owner decision D6).
--
-- The session store is a signed cookie, so logging out used to delete only the browser's copy: any
-- copy taken earlier stayed valid until it expired. Each teacher now has a session_version. Login
-- stores it in the cookie, every authenticated request compares it with this column, and logout
-- increments it, so every older cookie (other devices, copies) stops working.
--
-- Apply once against a database set up from an older soln_db.sql (a fresh load already has it):
--   mariadb -u$DB_USER -p$DB_PASSWORD $DB_NAME < migrations/003_session_version.sql
--
-- Sessions issued before this migration carry no version, so every teacher logs in once more.

ALTER TABLE users ADD COLUMN IF NOT EXISTS session_version INT NOT NULL DEFAULT 0;
