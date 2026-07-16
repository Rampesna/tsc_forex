BEGIN;

ALTER TABLE gamification_events ADD COLUMN IF NOT EXISTS source_type varchar(32);
ALTER TABLE gamification_events ADD COLUMN IF NOT EXISTS source_id varchar(96);
ALTER TABLE gamification_events ADD COLUMN IF NOT EXISTS rule_version varchar(24) NOT NULL DEFAULT '2026.1';

CREATE TABLE IF NOT EXISTS xp_ledger (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    idempotency_key varchar(120) NOT NULL,
    event_type varchar(48) NOT NULL,
    source_type varchar(32),
    source_id varchar(96),
    reason varchar(160) NOT NULL,
    rule_version varchar(24) NOT NULL,
    delta integer NOT NULL,
    day_key varchar(10),
    created_at timestamptz NOT NULL,
    CONSTRAINT uq_xp_ledger_user_key UNIQUE (user_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_xp_ledger_user_created ON xp_ledger (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS daily_plans (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    day_key varchar(10) NOT NULL,
    risk_appetite varchar(16) NOT NULL,
    watch_codes varchar(128),
    learning_goal varchar(64) NOT NULL,
    reflection varchar(500),
    created_at timestamptz,
    updated_at timestamptz,
    CONSTRAINT uq_daily_plan_user_day UNIQUE (user_id, day_key)
);

CREATE TABLE IF NOT EXISTS learning_progress (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    lesson_code varchar(48) NOT NULL,
    mastery integer NOT NULL DEFAULT 0,
    attempts integer NOT NULL DEFAULT 0,
    correct integer NOT NULL DEFAULT 0,
    completed_at timestamptz,
    next_review_at timestamptz,
    updated_at timestamptz,
    CONSTRAINT uq_learning_progress_user_lesson UNIQUE (user_id, lesson_code)
);

CREATE TABLE IF NOT EXISTS learning_attempts (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    idempotency_key varchar(120) NOT NULL,
    lesson_code varchar(48) NOT NULL,
    question_code varchar(64) NOT NULL,
    answer integer NOT NULL,
    correct boolean NOT NULL,
    xp_awarded integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL,
    CONSTRAINT uq_learning_attempt_user_key UNIQUE (user_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_learning_attempts_user_created ON learning_attempts (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS discipline_reviews (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    day_key varchar(10) NOT NULL,
    code varchar(48) NOT NULL,
    note varchar(1000) NOT NULL,
    created_at timestamptz,
    CONSTRAINT uq_discipline_review_user_key UNIQUE (user_id, day_key, code)
);

CREATE TABLE IF NOT EXISTS season_scores (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    season_code varchar(32) NOT NULL,
    points bigint NOT NULL DEFAULT 0,
    league varchar(24) NOT NULL DEFAULT 'Başlangıç',
    updated_at timestamptz,
    CONSTRAINT uq_season_score_user UNIQUE (user_id, season_code)
);
CREATE INDEX IF NOT EXISTS idx_season_scores_rank ON season_scores (season_code, points DESC, updated_at);

CREATE TABLE IF NOT EXISTS paper_trades (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    client_id varchar(80) NOT NULL,
    code varchar(32) NOT NULL,
    side varchar(8) NOT NULL,
    quantity double precision NOT NULL,
    entry_price double precision NOT NULL,
    thesis varchar(1000) NOT NULL,
    risk_budget_pct double precision NOT NULL,
    invalidation varchar(500) NOT NULL,
    occurred_at timestamptz NOT NULL,
    followed_plan boolean,
    retrospective varchar(1000),
    process_score integer NOT NULL DEFAULT 0,
    reviewed_at timestamptz,
    created_at timestamptz,
    updated_at timestamptz,
    CONSTRAINT uq_paper_trade_user_client UNIQUE (user_id, client_id)
);
CREATE INDEX IF NOT EXISTS idx_paper_trades_user_created ON paper_trades (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS gamification_privacy (
    user_id uuid PRIMARY KEY,
    alias varchar(40),
    league_opt_in boolean NOT NULL DEFAULT true,
    personalized_learning boolean NOT NULL DEFAULT true,
    social_opt_in boolean NOT NULL DEFAULT false,
    learning_notifications boolean NOT NULL DEFAULT true,
    updated_at timestamptz
);

CREATE TABLE IF NOT EXISTS study_groups (
    id uuid PRIMARY KEY,
    owner_id uuid NOT NULL,
    name varchar(60) NOT NULL,
    invite_code varchar(12) NOT NULL UNIQUE,
    created_at timestamptz
);

CREATE TABLE IF NOT EXISTS study_group_members (
    id uuid PRIMARY KEY,
    group_id uuid NOT NULL,
    user_id uuid NOT NULL,
    alias varchar(40) NOT NULL,
    joined_at timestamptz,
    CONSTRAINT uq_study_group_member UNIQUE (group_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_study_group_members_user ON study_group_members (user_id, joined_at DESC);

COMMIT;
