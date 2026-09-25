-- Migration 001: add created_at to the three tables the game writes to, and index it
-- alongside classroom_id (02 §D1). This unlocks time-ordered UI (Recent activity, T6.2)
-- without any game-client change - created_at is a server-side default.
--
-- Apply against an existing database that was set up from an older soln_db.sql
-- (a fresh load already has these from soln_db.sql itself):
--   mariadb -u$DB_USER -p$DB_PASSWORD $DB_NAME < migrations/001_created_at.sql
--
-- Existing rows get the migration's timestamp, not their real creation time - anything
-- that cares about "how long ago" should treat rows from before this ran as unknown
-- ("—"), not as having just happened.

ALTER TABLE fraction_responses        ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE multiple_choice_responses ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE multiple_choice_scores    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_fr_created  ON fraction_responses (classroom_id, created_at);
CREATE INDEX IF NOT EXISTS idx_mcs_created ON multiple_choice_scores (classroom_id, created_at);
