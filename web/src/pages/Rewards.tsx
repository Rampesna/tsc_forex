import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/endpoints";
import type { GamBadge, GamProfile, LeaderEntry } from "../lib/types";
import { Button, Card } from "../components/common";

export function Rewards() {
  const qc = useQueryClient();
  const [toast, setToast] = useState<string | null>(null);
  const [celebrate, setCelebrate] = useState<GamBadge[]>([]);

  const profileQ = useQuery({ queryKey: ["gamification"], queryFn: api.gamification, refetchOnWindowFocus: false });
  const boardQ = useQuery({ queryKey: ["leaderboard"], queryFn: () => api.leaderboard(20), refetchOnWindowFocus: false });

  const checkIn = useMutation({
    mutationFn: api.checkIn,
    onSuccess: (res) => {
      qc.setQueryData(["gamification"], res.profile);
      qc.invalidateQueries({ queryKey: ["leaderboard"] });
      if (res.already_checked_in) {
        flash("Bugün zaten giriş yaptın ✓");
      } else {
        flash(`+${res.xp_gained} XP  •  🔥 ${res.profile.current_streak} gün`);
        if (res.new_badges.length) setCelebrate(res.new_badges);
      }
    },
    onError: () => flash("Bağlantı hatası, tekrar dene"),
  });

  function flash(msg: string) {
    setToast(msg);
    setTimeout(() => setToast((t) => (t === msg ? null : t)), 2800);
  }

  const p = profileQ.data;

  return (
    <div>
      <div style={{ marginBottom: 20 }}>
        <h1 style={{ fontSize: 26, fontWeight: 800, margin: 0 }}>Ödüller</h1>
        <div style={{ color: "var(--text-dim)", fontSize: 14, marginTop: 2 }}>
          Serini koru, seviye atla, rozet topla
        </div>
      </div>

      {profileQ.isLoading || !p ? (
        <Skeleton />
      ) : (
        <div style={{ display: "grid", gap: 16 }}>
          <Hero p={p} />
          <StatsRow p={p} />
          <DailyReward
            p={p}
            claiming={checkIn.isPending}
            onClaim={() => checkIn.mutate()}
          />
          <Quests p={p} />
          <Badges p={p} />
          <LeaderboardCard entries={boardQ.data?.entries ?? []} me={boardQ.data?.me} />
        </div>
      )}

      {toast && (
        <div
          style={{
            position: "fixed",
            top: 70,
            left: "50%",
            transform: "translateX(-50%)",
            background: "var(--surface-hi)",
            border: "1px solid var(--stroke-strong)",
            borderRadius: 999,
            padding: "12px 20px",
            fontWeight: 700,
            fontSize: 14,
            boxShadow: "var(--shadow)",
            zIndex: 200,
          }}
        >
          {toast}
        </div>
      )}

      {celebrate.length > 0 && (
        <Celebration badges={celebrate} onClose={() => setCelebrate([])} />
      )}
    </div>
  );
}

function Hero({ p }: { p: GamProfile }) {
  return (
    <Card style={{ padding: 20, display: "flex", gap: 20, alignItems: "center" }}>
      <LevelRing level={p.level} progress={p.progress} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 20, fontWeight: 800 }}>Seviye {p.level}</div>
        <XPBar progress={p.progress} />
        <div style={{ fontSize: 12, color: "var(--text-dim)", marginTop: 6 }}>
          {p.xp_into_level} / {p.xp_for_next_level} XP • Sonraki seviyeye{" "}
          {Math.max(0, p.xp_for_next_level - p.xp_into_level)} XP
        </div>
      </div>
    </Card>
  );
}

function StatsRow({ p }: { p: GamProfile }) {
  return (
    <div style={{ display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 12 }}>
      <Tile label="🔥 Seri" value={`${p.current_streak} gün`} tint="var(--down)" />
      <Tile label="Haftalık XP" value={`${p.weekly_xp}`} tint="var(--accent-hi)" />
      <Tile label="Sıralama" value={p.rank > 0 ? `#${p.rank}` : "—"} />
    </div>
  );
}

function DailyReward({ p, claiming, onClaim }: { p: GamProfile; claiming: boolean; onClaim: () => void }) {
  if (p.checked_in_today) {
    return (
      <Card style={{ padding: 16, display: "flex", alignItems: "center", gap: 12 }}>
        <span style={{ fontSize: 24 }}>✅</span>
        <div>
          <div style={{ fontWeight: 700 }}>Günlük giriş tamamlandı</div>
          <div style={{ fontSize: 12, color: "var(--text-dim)" }}>
            Yarın tekrar gel, serini büyüt! En uzun serin: {p.longest_streak} gün
          </div>
        </div>
      </Card>
    );
  }
  return (
    <Button variant="primary" full onClick={onClaim} disabled={claiming}>
      {claiming ? "…" : "🎁 Günlük Ödülü Al"}
    </Button>
  );
}

function Quests({ p }: { p: GamProfile }) {
  const done = p.quests.filter((q) => q.completed).length;
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 12 }}>
        <div style={{ fontWeight: 700 }}>Günlük Görevler</div>
        <div style={{ fontSize: 12, color: "var(--text-dim)", fontWeight: 700 }}>
          {done}/{p.quests.length}
        </div>
      </div>
      <div style={{ display: "grid", gap: 12 }}>
        {p.quests.map((q) => (
          <div key={q.code} style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <span style={{ fontSize: 18, color: q.completed ? "var(--up)" : "var(--text-faint)" }}>
              {q.completed ? "☑" : "☐"}
            </span>
            <div style={{ flex: 1 }}>
              <div
                style={{
                  fontSize: 14,
                  fontWeight: 600,
                  color: q.completed ? "var(--text-dim)" : "var(--text)",
                  textDecoration: q.completed ? "line-through" : "none",
                }}
              >
                {q.title}
              </div>
              <div style={{ fontSize: 11, color: "var(--text-faint)" }}>{q.description}</div>
            </div>
            <div className="mono" style={{ fontWeight: 700, color: "var(--accent-hi)" }}>
              +{q.xp}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function Badges({ p }: { p: GamProfile }) {
  const earned = p.badges.filter((b) => b.earned).length;
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 14 }}>
        <div style={{ fontWeight: 700 }}>Rozetler</div>
        <div style={{ fontSize: 12, color: "var(--text-dim)", fontWeight: 700 }}>
          {earned}/{p.badges.length}
        </div>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill,minmax(76px,1fr))", gap: 16 }}>
        {p.badges.map((b) => (
          <div key={b.code} title={`${b.name} — ${b.description}`} style={{ textAlign: "center" }}>
            <div
              style={{
                width: 54,
                height: 54,
                margin: "0 auto",
                borderRadius: 27,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 26,
                position: "relative",
                background: b.earned ? "var(--accent-soft)" : "var(--surface-hi)",
                opacity: b.earned ? 1 : 0.55,
              }}
            >
              <span style={{ filter: b.earned ? "none" : "grayscale(1)" }}>{b.icon}</span>
              {!b.earned && (
                <span style={{ position: "absolute", right: -2, bottom: -2, fontSize: 12 }}>🔒</span>
              )}
            </div>
            <div
              style={{
                fontSize: 10,
                marginTop: 5,
                color: b.earned ? "var(--text)" : "var(--text-faint)",
                lineHeight: 1.2,
              }}
            >
              {b.name}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function LeaderboardCard({ entries, me }: { entries: LeaderEntry[]; me?: LeaderEntry }) {
  const inTop = me && entries.some((e) => e.user_id === me.user_id);
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 12 }}>
        <div style={{ fontWeight: 700 }}>Haftalık Liderlik 👑</div>
      </div>
      {entries.length === 0 ? (
        <div style={{ fontSize: 13, color: "var(--text-dim)", padding: "8px 0" }}>
          Bu hafta henüz sıralama yok. İlk XP'yi sen kazan!
        </div>
      ) : (
        <div style={{ display: "grid", gap: 6 }}>
          {entries.map((e) => (
            <LeaderRow key={e.user_id} e={e} mine={me?.user_id === e.user_id} />
          ))}
          {me && !inTop && me.xp > 0 && (
            <>
              <div style={{ height: 1, background: "var(--stroke)", margin: "4px 0" }} />
              <LeaderRow e={me} mine />
            </>
          )}
        </div>
      )}
    </Card>
  );
}

function LeaderRow({ e, mine }: { e: LeaderEntry; mine: boolean }) {
  const medal = e.rank === 1 ? "🥇" : e.rank === 2 ? "🥈" : e.rank === 3 ? "🥉" : `${e.rank}`;
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: mine ? "8px 10px" : "6px 0",
        borderRadius: 10,
        background: mine ? "var(--accent-soft)" : "transparent",
      }}
    >
      <div className="mono" style={{ width: 30, fontWeight: 700, color: e.rank <= 3 ? "var(--accent-hi)" : "var(--text-dim)" }}>
        {medal}
      </div>
      <Avatar e={e} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 14, fontWeight: 600, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>
          {mine ? "Sen" : e.display_name}
        </div>
        <div style={{ fontSize: 11, color: "var(--text-faint)" }}>Seviye {e.level}</div>
      </div>
      <div className="mono" style={{ fontWeight: 700, color: "var(--accent-hi)" }}>{e.xp} XP</div>
    </div>
  );
}

function Avatar({ e }: { e: LeaderEntry }) {
  if (e.avatar_url) {
    return <img src={e.avatar_url} width={30} height={30} style={{ borderRadius: 15, objectFit: "cover" }} alt="" />;
  }
  const initials = (e.display_name || "?").slice(0, 1).toUpperCase();
  return (
    <div
      style={{
        width: 30,
        height: 30,
        borderRadius: 15,
        background: "var(--brand-deep)",
        color: "#fff",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontSize: 13,
        fontWeight: 700,
      }}
    >
      {initials}
    </div>
  );
}

function Celebration({ badges, onClose }: { badges: GamBadge[]; onClose: () => void }) {
  return (
    <div
      onClick={onClose}
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.6)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 300,
        padding: 20,
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: "var(--surface)",
          border: "1px solid var(--stroke-strong)",
          borderRadius: 18,
          padding: 26,
          maxWidth: 380,
          width: "100%",
          textAlign: "center",
        }}
      >
        <div style={{ fontSize: 44 }}>🎉</div>
        <div style={{ fontSize: 18, fontWeight: 800, margin: "8px 0 16px" }}>
          {badges.length > 1 ? "Yeni Rozetler!" : "Yeni Rozet!"}
        </div>
        <div style={{ display: "grid", gap: 12, marginBottom: 18 }}>
          {badges.map((b) => (
            <div key={b.code} style={{ display: "flex", alignItems: "center", gap: 12, textAlign: "left" }}>
              <span style={{ fontSize: 30 }}>{b.icon}</span>
              <div>
                <div style={{ fontWeight: 700 }}>{b.name}</div>
                <div style={{ fontSize: 12, color: "var(--text-dim)" }}>{b.description}</div>
              </div>
            </div>
          ))}
        </div>
        <Button variant="primary" full onClick={onClose}>
          Harika!
        </Button>
      </div>
    </div>
  );
}

// --- small pieces ---

function Tile({ label, value, tint }: { label: string; value: string; tint?: string }) {
  return (
    <Card style={{ padding: 14 }}>
      <div style={{ fontSize: 11, color: "var(--text-dim)", marginBottom: 4 }}>{label}</div>
      <div className="mono" style={{ fontSize: 18, fontWeight: 800, color: tint ?? "var(--text)" }}>
        {value}
      </div>
    </Card>
  );
}

function LevelRing({ level, progress }: { level: number; progress: number }) {
  const size = 84;
  const stroke = 8;
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const filled = Math.max(0.02, Math.min(1, progress)) * c;
  return (
    <svg width={size} height={size} style={{ flexShrink: 0 }}>
      <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--surface-hi)" strokeWidth={stroke} />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={r}
        fill="none"
        stroke="var(--accent)"
        strokeWidth={stroke}
        strokeLinecap="round"
        strokeDasharray={`${filled} ${c}`}
        transform={`rotate(-90 ${size / 2} ${size / 2})`}
      />
      <text x="50%" y="44%" textAnchor="middle" fontSize="9" fontWeight="700" fill="var(--text-dim)">
        SEVİYE
      </text>
      <text x="50%" y="66%" textAnchor="middle" fontSize="24" fontWeight="800" fill="var(--text)">
        {level}
      </text>
    </svg>
  );
}

function XPBar({ progress }: { progress: number }) {
  return (
    <div style={{ height: 10, borderRadius: 6, background: "var(--surface-hi)", overflow: "hidden", marginTop: 8 }}>
      <div
        style={{
          height: "100%",
          width: `${Math.max(4, Math.min(1, progress) * 100)}%`,
          background: "linear-gradient(90deg, var(--accent), var(--up))",
          borderRadius: 6,
        }}
      />
    </div>
  );
}

function Skeleton() {
  return (
    <div style={{ display: "grid", gap: 16 }}>
      {[110, 70, 150, 180].map((h, i) => (
        <div
          key={i}
          style={{
            height: h,
            borderRadius: "var(--radius)",
            background: "linear-gradient(90deg, var(--surface) 25%, var(--surface-hi) 37%, var(--surface) 63%)",
            backgroundSize: "400% 100%",
            animation: "tscShimmer 1.4s ease infinite",
          }}
        />
      ))}
      <style>{`@keyframes tscShimmer { 0% { background-position: 100% 0 } 100% { background-position: -100% 0 } }`}</style>
    </div>
  );
}
