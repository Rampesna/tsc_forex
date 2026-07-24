export interface CatalogItem {
  code: string;
  display_name: string;
  kind: string;
  unit: string;
  currency?: string;
}

export interface InstrumentPrice {
  code: string;
  display_name: string;
  kind: string;
  price: number;
  bid: number;
  ask: number;
  change: number;
  change_pct: number;
  currency?: string; // native currency of `price`: "TRY" (default) or "USD"
  price_try?: number; // TRY equivalent (== price for TRY instruments)
  provider?: string;
  timestamp?: string;
}

export interface Stats {
  code: string;
  open: number;
  high: number;
  low: number;
  last: number;
  change: number;
  change_pct: number;
  count: number;
}

export interface Candle {
  bucket: string;
  open: number;
  high: number;
  low: number;
  close: number;
}

// Rich per-kind detail payload from GET /instruments/:code/detail. All fields
// are optional because they depend on the instrument kind + upstream coverage.
export interface InstrumentDetail {
  code: string;
  kind: string;
  price_try?: number;

  // crypto
  price_usd?: number;
  high24h?: number;
  low24h?: number;
  open24h?: number;
  prev_close?: number;
  weighted_avg?: number;
  volume_base?: number;
  quote_volume_usd?: number;
  trades?: number;
  change_pct24h?: number;
  market_cap_usd?: number;
  rank?: number;
  fdv_usd?: number;
  total_volume_usd?: number;
  circulating_supply?: number;
  total_supply?: number;
  max_supply?: number;
  ath_usd?: number;
  atl_usd?: number;
  ath_date?: string;
  atl_date?: string;
  dominance_pct?: number;

  // stock
  currency?: string; // "TRY" (BIST) or "USD" (global)
  day_high?: number;
  day_low?: number;
  week52_high?: number;
  week52_low?: number;
  volume?: number;
  exchange?: string;
  long_name?: string;

  // gold
  ons_usd?: number;
  ons_try?: number;
  usd_try?: number;
  gram_has_try?: number;
  gram_weight?: number;
  fineness?: number;
  purity_label?: string;

  // currency
  inverse?: number;
  usd_cross?: number;
  iso?: string;
}

export interface Profile {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string;
  provider: string;
  role: string;
  created_at: string;
}

export interface Alert {
  id: string;
  code: string;
  direction: "above" | "below";
  threshold: number;
  active: boolean;
  last_fired_at?: string;
  created_at: string;
}

export interface Notification {
  id: string;
  code: string;
  title: string;
  body: string;
  price: number;
  read_at?: string;
  created_at: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
}

export type WSMessageType =
  | "price_update"
  | "instrument_update"
  | "notification"
  | "heartbeat"
  | "error"
  | "subscribed";

export interface WSEnvelope<T = unknown> {
  type: WSMessageType;
  payload?: T;
  timestamp: string;
}

export interface InstrumentUpdatePayload {
  code: string;
  display_name: string;
  kind: string;
  price: number;
  bid: number;
  ask: number;
  change: number;
  change_pct: number;
  currency?: string;
  price_try?: number;
  provider?: string;
}

export interface NotificationPayload {
  id?: string;
  code: string;
  title: string;
  body: string;
  price: number;
}

export interface MarketInsight {
  code: string;
  title: string;
  stance: "positive" | "neutral" | "cautious";
  stance_label: string;
  summary: string;
  factors: string[];
  risk_note: string;
  confidence: "low" | "medium" | "high";
  time_horizons: Array<{ horizon: string; outlook: string; rationale: string }>;
  macro_drivers: string[];
  geopolitical_risks: string[];
  cross_asset_analysis: string[];
  scenarios: Array<{ name: string; likelihood: "low" | "medium" | "high"; trigger: string; implication: string }>;
  what_i_would_do?: {
    intro: string;
    short_term: string;
    medium_term: string;
    long_term: string;
  };
  data_context: Array<{
    code: string;
    display_name: string;
    price: number;
    currency: string;
    change_1d_pct?: number;
    change_1w_pct?: number;
    change_1m_pct?: number;
    change_3m_pct?: number;
    change_1y_pct?: number;
    timestamp: string;
  }>;
  sources: Array<{ title: string; url: string }>;
  research_mode: string;
  generated_at: string;
  disclaimer: string;
}

// Billing is owned by the App Store. The web client only reflects the
// server-verified entitlement and quota; it never starts or restores a sale.
export type BillingPlanCode = "free" | "plus" | "pro" | "expert" | "elite" | (string & {});

export interface UsageQuota {
  used: number;
  limit: number;
  remaining: number;
  period_end?: string;
}

export interface BillingPlanLimits {
  ai_research: number;
  active_alerts: number;
  watchlist: number;
  portfolios: number;
  portfolio_health: number;
  exports: number;
  research_archive_days: number;
}

export interface BillingPlan {
  code: BillingPlanCode;
  name: string;
  product_id: string;
  price_hint_try?: number | string;
  limits: BillingPlanLimits;
}

export interface StoreEntitlement {
  product_id: string;
  status: "active" | "grace_period" | "expired" | "revoked" | string;
  environment: "Sandbox" | "Production" | string;
  expires_at?: string;
  will_renew?: boolean;
}

export interface BillingStatus {
  plan: BillingPlan;
  entitlement: StoreEntitlement | null;
  usage: Record<string, UsageQuota>;
  free_research_available: boolean;
  purchase_channel: "ios_app_store";
  manage_url?: string;
}

export interface BillingPlansResponse {
  purchase_channel: "ios_app_store";
  free_trial: {
    ready: boolean;
    user_limit: number;
    device_limit: number;
  };
  plans: BillingPlan[];
}

// --- Gamification (streak / XP / badges / leaderboard) ---

export interface GamBadge {
  code: string;
  name: string;
  description: string;
  icon: string;
  earned: boolean;
  earned_at?: string;
}

export interface GamQuest {
  code: string;
  title: string;
  description: string;
  xp: number;
  completed: boolean;
}

export interface GamProfile {
  xp: number;
  level: number;
  xp_into_level: number;
  xp_for_next_level: number;
  progress: number;
  current_streak: number;
  longest_streak: number;
  checked_in_today: boolean;
  total_check_ins: number;
  weekly_xp: number;
  rank: number;
  badges: GamBadge[];
  quests: GamQuest[];
}

export interface GamCheckInResult {
  profile: GamProfile;
  xp_gained: number;
  new_badges: GamBadge[];
  already_checked_in: boolean;
}

export interface LeaderEntry {
  rank: number;
  user_id: string;
  display_name: string;
  avatar_url: string;
  xp: number;
  level: number;
}

export interface Leaderboard {
  entries: LeaderEntry[];
  me: LeaderEntry;
}

export interface DailyPlan {
  id: string;
  day_key: string;
  risk_appetite: "low" | "moderate" | "high";
  watch_codes: string[];
  learning_goal: string;
  reflection: string;
  created_at: string;
  updated_at: string;
}

export interface LessonQuestion {
  code: string;
  text: string;
  choices: string[];
}

export interface LessonStatus {
  code: string;
  title: string;
  summary: string;
  cards: string[];
  question: LessonQuestion;
  mastery: number;
  attempts: number;
  completed: boolean;
  due_for_review: boolean;
  next_review_at?: string;
}

export interface DisciplineQuest {
  code: string;
  title: string;
  description: string;
  xp: number;
  completed: boolean;
}

export interface SeasonView {
  code: string;
  name: string;
  league: string;
  points: number;
  rank: number;
  ends_at: string;
  standings: LeaderEntry[];
  scoring: string;
}

export interface PaperTrade {
  id: string;
  client_id: string;
  code: string;
  side: "buy" | "sell";
  quantity: number;
  entry_price: number;
  thesis: string;
  risk_budget_pct: number;
  invalidation: string;
  occurred_at: string;
  followed_plan?: boolean;
  retrospective: string;
  process_score: number;
  reviewed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface GamificationPrivacy {
  user_id: string;
  alias: string;
  league_opt_in: boolean;
  personalized_learning: boolean;
  social_opt_in: boolean;
  learning_notifications: boolean;
  updated_at: string;
}

export interface StudyGroup {
  id: string;
  owner_id: string;
  name: string;
  invite_code: string;
  created_at: string;
  member_count: number;
}

export interface GamificationExperience {
  rule_version: string;
  daily_plan?: DailyPlan;
  lessons: LessonStatus[];
  discipline_quests: DisciplineQuest[];
  season: SeasonView;
  paper_trades: PaperTrade[];
  privacy: GamificationPrivacy;
  study_groups: StudyGroup[];
  weekly_summary: {
    mastered_lessons: number;
    total_lessons: number;
    season_points: number;
    due_reviews: number;
  };
}

export interface QuizAttemptResult {
  correct: boolean;
  xp_gained: number;
  created: boolean;
  lesson: LessonStatus;
}

// --- Cloud portfolio ledger / analytics ---

export type DecimalValue = string | number;

export interface PortfolioMeta {
  id: string;
  user_id: string;
  name: string;
  base_currency: string;
  net_deposits_try: DecimalValue;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface PortfolioPosition {
  id: string;
  code: string;
  display_name: string;
  kind: string;
  currency: string;
  quantity: DecimalValue;
  average_cost_native: DecimalValue;
  average_cost_try: DecimalValue;
  current_price_native: DecimalValue;
  current_price_try: DecimalValue;
  value_native: DecimalValue;
  value_try: DecimalValue;
  cost_basis_try: DecimalValue;
  unrealized_pnl_try: DecimalValue;
  unrealized_pnl_pct: DecimalValue;
  realized_pnl_try: DecimalValue;
  day_change_try: DecimalValue;
  day_change_pct: DecimalValue;
  allocation_pct: DecimalValue;
  updated_at: string;
}

export interface PortfolioOverview {
  portfolio: PortfolioMeta;
  positions: PortfolioPosition[];
  total_value_try: DecimalValue;
  total_cost_try: DecimalValue;
  unrealized_pnl_try: DecimalValue;
  realized_pnl_try: DecimalValue;
  day_change_try: DecimalValue;
  day_change_pct: DecimalValue;
  net_deposits_try: DecimalValue;
  active_positions: number;
}

export interface PortfolioHealth {
  score?: number;
  grade?: string;
  summary?: string;
  findings?: Array<{ title?: string; detail?: string; severity?: "low" | "medium" | "high" | string }>;
  generated_at?: string;
  [key: string]: unknown;
}

export type PortfolioTransactionType =
  | "buy"
  | "sell"
  | "dividend"
  | "fee"
  | "deposit"
  | "withdraw";

export interface PortfolioTransaction {
  id: string;
  portfolio_id: string;
  user_id: string;
  client_id: string;
  code: string;
  type: PortfolioTransactionType;
  quantity: DecimalValue;
  unit_price_native: DecimalValue;
  amount_native: DecimalValue;
  fee_native: DecimalValue;
  currency: string;
  usdtry: DecimalValue;
  occurred_at: string;
  note: string;
  created_at: string;
  updated_at: string;
}

export interface PortfolioPerformancePoint {
  date: string;
  value_try: DecimalValue;
  cost_basis_try: DecimalValue;
  realized_pnl_try: DecimalValue;
  total_pnl_try: DecimalValue;
  net_deposits_try: DecimalValue;
}
