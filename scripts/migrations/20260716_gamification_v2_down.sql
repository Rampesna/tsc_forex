BEGIN;

DROP TABLE IF EXISTS study_group_members;
DROP TABLE IF EXISTS study_groups;
DROP TABLE IF EXISTS gamification_privacy;
DROP TABLE IF EXISTS paper_trades;
DROP TABLE IF EXISTS season_scores;
DROP TABLE IF EXISTS discipline_reviews;
DROP TABLE IF EXISTS learning_attempts;
DROP TABLE IF EXISTS learning_progress;
DROP TABLE IF EXISTS daily_plans;
DROP TABLE IF EXISTS xp_ledger;

ALTER TABLE gamification_events DROP COLUMN IF EXISTS rule_version;
ALTER TABLE gamification_events DROP COLUMN IF EXISTS source_id;
ALTER TABLE gamification_events DROP COLUMN IF EXISTS source_type;

COMMIT;
