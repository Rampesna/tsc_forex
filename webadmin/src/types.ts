export type AdminUser = {
  id: string;
  email: string;
  display_name: string;
  must_change_password: boolean;
};

export type SessionResponse = {
  access_token: string;
  token_type: string;
  user: AdminUser;
};

export type NamedValue = { name: string; value: number };
export type TimeValue = { time: string; value: number; count?: number };

export type OnlineUser = {
  user_id: string;
  email: string;
  display_name: string;
  platform: string;
  connections: number;
  connected_at: string;
  last_seen_at: string;
};

export type DashboardData = {
  generated_at: string;
  range_days: number;
  totals: {
    users: number;
    new_users: number;
    online_users: number;
    active_subscriptions: number;
    active_alerts: number;
    favorites: number;
    ai_research_uses: number;
  };
  revenue: {
    daily: RevenueWindow;
    weekly: RevenueWindow;
    monthly: RevenueWindow;
    series: TimeValue[];
    notice: string;
  };
  user_series: TimeValue[];
  activity_series: TimeValue[];
  platform_distribution: NamedValue[];
  plan_distribution: NamedValue[];
  provider_distribution: NamedValue[];
  sales_by_plan: NamedValue[];
  top_favorite_assets: NamedValue[];
  recently_online: OnlineUser[];
};

export type RevenueWindow = {
  sales: number;
  gross_try: number;
  label: string;
  is_estimated: boolean;
};

export type UserSummary = {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string;
  provider: string;
  plan_code: string;
  purchase_platform: string;
  device_platform: string;
  favorites: number;
  active_alerts: number;
  badges: number;
  ai_research_uses: number;
  created_at: string;
  last_active_at?: string;
  subscription_expiry?: string;
  online: boolean;
};

export type UsersPage = {
  items: UserSummary[];
  page: number;
  page_size: number;
  total: number;
};

export type Device = {
  id: string;
  platform: string;
  locale?: string;
  app_version?: string;
  device_model?: string;
  last_seen_at: string;
  created_at: string;
};

export type Favorite = { id: string; code: string; created_at: string };
export type Alert = {
  id: string;
  code: string;
  direction: string;
  threshold: number;
  active: boolean;
  last_fired_at?: string;
  created_at: string;
};
export type Badge = { id: string; code: string; earned_at: string };
export type DailyPlan = {
  id: string;
  day_key: string;
  risk_appetite: string;
  learning_goal: string;
  reflection?: string;
  created_at: string;
};
export type FeatureUsage = {
  id: string;
  feature: string;
  period_key: string;
  scope: string;
  quantity: number;
  created_at: string;
};
export type LearningProgress = {
  id: string;
  lesson_code: string;
  mastery: number;
  attempts: number;
  correct: number;
  completed_at?: string;
  next_review_at?: string;
  updated_at: string;
};
export type DisciplineReview = {
  id: string;
  day_key: string;
  code: string;
  note: string;
  created_at: string;
};
export type Portfolio = { id: string; name: string; base_currency: string; created_at: string; updated_at: string };
export type Position = {
  id?: string;
  portfolio_id: string;
  code: string;
  quantity: number | string;
  average_cost_native: number | string;
  average_cost_try: number | string;
  realized_pnl_try: number | string;
  updated_at?: string;
};
export type Activity = { type: string; label: string; reference?: string; created_at: string };

export type UserDetail = {
  user: UserSummary;
  devices: Device[];
  favorites: Favorite[];
  alerts: Alert[];
  badges: Badge[];
  gamification?: {
    xp: number;
    level: number;
    current_streak: number;
    longest_streak: number;
    total_check_ins: number;
  };
  daily_plans: DailyPlan[];
  learning_progress: LearningProgress[];
  discipline_reviews: DisciplineReview[];
  feature_usage: FeatureUsage[];
  portfolios: Portfolio[];
  positions: Position[];
  activity: Activity[];
};

export type PerformanceFilter = {
  direction: "decliners" | "gainers";
  lookback_days: number;
  top_n: number;
  min_abs_change_pct: number;
};

export type TargetingPlan = {
  audience: string;
  platforms: string[];
  plan_codes: string[];
  asset_codes: string[];
  asset_sources: string[];
  performance: PerformanceFilter | null;
  registered_within_days: number;
  inactive_for_days: number;
  max_recipients: number;
};

export type CampaignPreview = {
  campaign_id: string;
  name: string;
  summary: string;
  title: string;
  body: string;
  deep_link: string;
  plan: TargetingPlan;
  matched_assets: { code: string; change_pct: number }[];
  recipient_count: number;
  total_matched: number;
  sample_users: {
    id: string;
    email: string;
    display_name: string;
    device_platform: string;
  }[];
  approval_fingerprint: string;
};

export type Campaign = {
  id: string;
  name: string;
  prompt: string;
  status: string;
  title: string;
  body: string;
  estimated_recipients: number;
  resolved_recipients: number;
  processed_recipients: number;
  failed_recipients: number;
  approved_at?: string;
  completed_at?: string;
  created_at: string;
};

export type AuditItem = {
  id: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  metadata: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
};
