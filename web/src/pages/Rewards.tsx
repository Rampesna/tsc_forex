import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/endpoints";
import type {
  CatalogItem,
  GamBadge,
  GamificationExperience,
  GamificationPrivacy,
  GamProfile,
  LessonStatus,
  PaperTrade,
} from "../lib/types";
import { Button, Card } from "../components/common";
import { useMarkets } from "../store/markets";

export function Rewards() {
  const qc = useQueryClient();
  const catalog = useMarkets((state) => state.catalog);
  const [toast, setToast] = useState<string | null>(null);
  const [celebrate, setCelebrate] = useState<GamBadge[]>([]);
  const [showPlan, setShowPlan] = useState(false);
  const [selectedBadge, setSelectedBadge] = useState<GamBadge | null>(null);

  const profileQ = useQuery({ queryKey: ["gamification"], queryFn: api.gamification, refetchOnWindowFocus: false });
  const experienceQ = useQuery({
    queryKey: ["gamification-experience"],
    queryFn: api.gamificationExperience,
    refetchOnWindowFocus: false,
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
          {experienceQ.data && <DevelopmentSummary experience={experienceQ.data} />}
          <Badges p={p} onSelect={setSelectedBadge} />
          {experienceQ.data && (
            <ExperienceCards
              profile={p}
              experience={experienceQ.data}
              onPlan={() => setShowPlan(true)}
              flash={flash}
              refresh={() => {
                qc.invalidateQueries({ queryKey: ["gamification-experience"] });
                qc.invalidateQueries({ queryKey: ["gamification"] });
              }}
            />
          )}
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
      {selectedBadge && <BadgeDetail badge={selectedBadge} onClose={() => setSelectedBadge(null)} />}
      {showPlan && (
        <PlanEditor
          experience={experienceQ.data}
          catalog={catalog}
          onClose={() => setShowPlan(false)}
          onSaved={(res) => {
            qc.setQueryData(["gamification"], res.profile);
            qc.invalidateQueries({ queryKey: ["gamification-experience"] });
            setShowPlan(false);
            flash(`Plan kaydedildi · +${res.xp_gained} XP · 🔥 ${res.profile.current_streak} gün`);
            if (res.new_badges.length) setCelebrate(res.new_badges);
          }}
          flash={flash}
        />
      )}
    </div>
  );
}

function PlanEditor({
  experience,
  catalog,
  onClose,
  onSaved,
  flash,
}: {
  experience?: GamificationExperience;
  catalog: CatalogItem[];
  onClose: () => void;
  onSaved: (result: Awaited<ReturnType<typeof api.saveDailyPlan>>) => void;
  flash: (message: string) => void;
}) {
  const lessons = experience?.lessons ?? [];
  const [risk, setRisk] = useState<"low" | "moderate" | "high">(
    experience?.daily_plan?.risk_appetite ?? "moderate",
  );
  const [codes, setCodes] = useState<string[]>(experience?.daily_plan?.watch_codes ?? []);
  const [assetKind, setAssetKind] = useState("gold");
  const [assetSearch, setAssetSearch] = useState("");
  const [goal, setGoal] = useState(
    experience?.daily_plan?.learning_goal ?? lessons[0]?.code ?? "risk_budget",
  );
  const [reflection, setReflection] = useState(experience?.daily_plan?.reflection ?? "");
  const save = useMutation({
    mutationFn: () =>
      api.saveDailyPlan({
        risk_appetite: risk,
        watch_codes: codes.slice(0, 3),
        learning_goal: goal,
        reflection,
      }),
    onSuccess: onSaved,
    onError: (error: Error) => flash(error.message || "Plan kaydedilemedi"),
  });
  const selectedAssets = codes
    .map((code) => catalog.find((item) => item.code === code))
    .filter((item): item is CatalogItem => Boolean(item));
  const matchingAssets = catalog
    .filter((item) => item.kind === assetKind)
    .filter((item) =>
      !assetSearch ||
      item.display_name.toLocaleLowerCase("tr-TR").includes(assetSearch.toLocaleLowerCase("tr-TR")) ||
      item.code.toLocaleLowerCase("tr-TR").includes(assetSearch.toLocaleLowerCase("tr-TR")),
    )
    .sort((a, b) => a.display_name.localeCompare(b.display_name, "tr"))
    .slice(0, 12);

  const toggleAsset = (code: string) => {
    setCodes((current) => {
      if (current.includes(code)) return current.filter((item) => item !== code);
      if (current.length >= 3) return current;
      return [...current, code];
    });
  };

  return (
    <Modal title="Bugünü planla" subtitle="Ödül, butona basmaya değil plan yapmaya bağlıdır." onClose={onClose}>
      <Field label="Risk yaklaşımı">
        <select value={risk} onChange={(event) => setRisk(event.target.value as typeof risk)} style={inputStyle}>
          <option value="low">Düşük — korumacı</option>
          <option value="moderate">Orta — dengeli</option>
          <option value="high">Yüksek — sınırları belirli</option>
        </select>
      </Field>
      <Field label="Bugün izleyeceğim varlıklar (en fazla 3)">
        {selectedAssets.length > 0 && (
          <div style={{ display: "flex", flexWrap: "wrap", gap: 7, marginBottom: 10 }}>
            {selectedAssets.map((item) => (
              <button
                key={item.code}
                type="button"
                onClick={() => toggleAsset(item.code)}
                style={{ ...chipStyle, color: "var(--accent)", borderColor: "var(--accent)" }}
              >
                {item.display_name} ×
              </button>
            ))}
          </div>
        )}
        <div style={{ display: "flex", flexWrap: "wrap", gap: 6, marginBottom: 9 }}>
          {PLAN_ASSET_KINDS.map((item) => (
            <button
              key={item.id}
              type="button"
              onClick={() => {
                setAssetKind(item.id);
                setAssetSearch("");
              }}
              style={{
                ...chipStyle,
                color: assetKind === item.id ? "var(--accent)" : "var(--text-dim)",
                borderColor: assetKind === item.id ? "var(--accent)" : "var(--stroke)",
              }}
            >
              {item.label}
            </button>
          ))}
        </div>
        <input
          value={assetSearch}
          onChange={(event) => setAssetSearch(event.target.value)}
          placeholder={`${PLAN_ASSET_KINDS.find((item) => item.id === assetKind)?.label ?? "Varlık"} ara`}
          style={inputStyle}
        />
        <div style={{ display: "grid", gap: 5, maxHeight: 230, overflowY: "auto", marginTop: 8 }}>
          {matchingAssets.map((item) => {
            const selected = codes.includes(item.code);
            return (
              <button
                key={item.code}
                type="button"
                onClick={() => toggleAsset(item.code)}
                style={{
                  ...pickerAssetButtonStyle,
                  borderColor: selected ? "var(--accent)" : "var(--stroke)",
                  background: selected ? "color-mix(in srgb, var(--accent) 12%, var(--surface-hi))" : "var(--surface-hi)",
                }}
              >
                <b>{item.display_name}</b>
                <span style={{ color: selected ? "var(--up)" : "var(--text-dim)" }}>{selected ? "✓ Seçildi" : "Seç"}</span>
              </button>
            );
          })}
        </div>
      </Field>
      <Field label="Öğrenme hedefi">
        <select value={goal} onChange={(event) => setGoal(event.target.value)} style={inputStyle}>
          {lessons.map((lesson) => <option key={lesson.code} value={lesson.code}>{lesson.title}</option>)}
        </select>
      </Field>
      <Field label="Dünkü plandan kısa not (isteğe bağlı)">
        <textarea value={reflection} onChange={(event) => setReflection(event.target.value)} rows={3} maxLength={500} style={inputStyle} />
      </Field>
      <Button variant="primary" full disabled={save.isPending || !goal || codes.length === 0} onClick={() => save.mutate()}>
        {save.isPending ? "Kaydediliyor…" : "Planı Kaydet ve Seriyi Sürdür"}
      </Button>
    </Modal>
  );
}

function ExperienceCards({
  profile,
  experience,
  onPlan,
  refresh,
  flash,
}: {
  profile: GamProfile;
  experience: GamificationExperience;
  onPlan: () => void;
  refresh: () => void;
  flash: (message: string) => void;
}) {
  const [activeTab, setActiveTab] = useState<ExperienceTab>("plan");
  const tabs: Array<{ id: ExperienceTab; label: string; description: string }> = [
    { id: "plan", label: "Bugünü planla", description: "Risk sınırı ve odak" },
    { id: "quests", label: "Günlük görevler", description: "Süreç görevleri" },
    { id: "learning", label: "Mikro dersler", description: "Bilgi ve tekrar" },
    { id: "discipline", label: "Disiplin", description: "Karar kalitesi" },
    { id: "season", label: "Sezon", description: "Kişisel ilerleme" },
    { id: "paper", label: "Karar günlüğü", description: "Paper senaryolar" },
    { id: "community", label: "Gruplar ve gizlilik", description: "Paylaşım tercihleri" },
  ];
  return (
    <section aria-labelledby="development-tools-title" style={{ display: "grid", gap: 12 }}>
      <div>
        <h2 id="development-tools-title" style={{ margin: 0, fontSize: 18 }}>Gelişim araçları</h2>
        <div style={{ marginTop: 3, color: "var(--text-dim)", fontSize: 12 }}>Her araç kendi çalışma alanında açılır; finansal getiri puanlamaya dahil edilmez.</div>
      </div>
      <Card style={{ padding: 7, overflow: "hidden" }}>
        <div role="tablist" aria-label="Gelişim araçları" style={{ display: "flex", gap: 6, overflowX: "auto" }}>
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              role="tab"
              aria-selected={activeTab === tab.id}
              aria-controls="rewards-feature-panel"
              onClick={() => setActiveTab(tab.id)}
              style={{
                flex: "0 0 auto",
                minWidth: 132,
                padding: "10px 11px",
                border: activeTab === tab.id ? "1px solid var(--stroke-strong)" : "1px solid transparent",
                borderRadius: 9,
                color: activeTab === tab.id ? "var(--text)" : "var(--text-dim)",
                background: activeTab === tab.id ? "var(--surface-hi)" : "transparent",
                textAlign: "left",
                cursor: "pointer",
              }}
            >
              <strong style={{ display: "block", fontSize: 11.5 }}>{tab.label}</strong>
              <small style={{ display: "block", marginTop: 3, color: "var(--text-faint)", fontSize: 9.5 }}>{tab.description}</small>
            </button>
          ))}
        </div>
      </Card>
      <div id="rewards-feature-panel" role="tabpanel" tabIndex={0}>
        {activeTab === "plan" && <DailyReward p={profile} planned={Boolean(experience.daily_plan)} onClaim={onPlan} />}
        {activeTab === "quests" && <Quests p={profile} />}
        {activeTab === "learning" && <LearningLab lessons={experience.lessons} refresh={refresh} flash={flash} />}
        {activeTab === "discipline" && <DisciplineLab quests={experience.discipline_quests} refresh={refresh} flash={flash} />}
        {activeTab === "season" && <SeasonCard experience={experience} />}
        {activeTab === "paper" && <PaperLab trades={experience.paper_trades} refresh={refresh} flash={flash} />}
        {activeTab === "community" && <PrivacyAndGroups experience={experience} refresh={refresh} flash={flash} />}
      </div>
    </section>
  );
}

type ExperienceTab = "plan" | "quests" | "learning" | "discipline" | "season" | "paper" | "community";

function DevelopmentSummary({ experience }: { experience: GamificationExperience }) {
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ fontWeight: 800, marginBottom: 12 }}>Gelişim Özeti</div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 10 }}>
        <MiniStat label="Ustalık" value={`${experience.weekly_summary.mastered_lessons}/${experience.weekly_summary.total_lessons}`} />
        <MiniStat label="Tekrar" value={`${experience.weekly_summary.due_reviews}`} />
        <MiniStat label="Sezon puanı" value={`${experience.weekly_summary.season_points}`} />
      </div>
      <div style={{ marginTop: 10, fontSize: 11, color: "var(--text-faint)" }}>
        Kural sürümü {experience.rule_version} · Finansal getiri puanlamaya dahil değildir.
      </div>
    </Card>
  );
}

function LearningLab({
  lessons,
  refresh,
  flash,
}: {
  lessons: LessonStatus[];
  refresh: () => void;
  flash: (message: string) => void;
}) {
  const [selected, setSelected] = useState(lessons[0]?.code ?? "");
  const [answer, setAnswer] = useState<number | null>(null);
  const lesson = lessons.find((item) => item.code === selected) ?? lessons[0];
  const submit = useMutation({
    mutationFn: () =>
      api.submitQuiz({
        lesson_code: lesson.code,
        question_code: lesson.question.code,
        answer: answer ?? -1,
        idempotency_key: crypto.randomUUID(),
      }),
    onSuccess: (result) => {
      flash(result.correct ? `Doğru · +${result.xp_gained} XP` : "Henüz değil; kartları yeniden gözden geçir.");
      setAnswer(null);
      refresh();
    },
    onError: (error: Error) => flash(error.message || "Yanıt gönderilemedi"),
  });
  if (!lesson) return null;
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", gap: 12, marginBottom: 12 }}>
        <div>
          <div style={{ fontWeight: 800 }}>Mikro Dersler</div>
          <div style={{ fontSize: 12, color: "var(--text-dim)" }}>Bilgi kontrolü ve aralıklı tekrar</div>
        </div>
        <span style={pillStyle}>{lesson.mastery}% ustalık</span>
      </div>
      <div style={{ display: "flex", gap: 8, overflowX: "auto", paddingBottom: 10 }}>
        {lessons.map((item) => (
          <button key={item.code} onClick={() => { setSelected(item.code); setAnswer(null); }} style={{
            ...chipStyle,
            borderColor: item.code === lesson.code ? "var(--accent)" : "var(--stroke)",
            color: item.code === lesson.code ? "var(--accent-hi)" : "var(--text-dim)",
          }}>
            {item.completed ? "✓ " : item.due_for_review ? "↻ " : ""}{item.title}
          </button>
        ))}
      </div>
      <div style={{ fontWeight: 700, margin: "4px 0" }}>{lesson.title}</div>
      <div style={{ fontSize: 13, color: "var(--text-dim)", marginBottom: 10 }}>{lesson.summary}</div>
      <div style={{ display: "grid", gap: 8, marginBottom: 14 }}>
        {lesson.cards.map((card, index) => (
          <div key={card} style={{ padding: 11, borderRadius: 10, background: "var(--surface-hi)", fontSize: 12 }}>
            <strong>{index + 1}.</strong> {card}
          </div>
        ))}
      </div>
      <div style={{ fontSize: 13, fontWeight: 700, marginBottom: 8 }}>{lesson.question.text}</div>
      <div style={{ display: "grid", gap: 7 }}>
        {lesson.question.choices.map((choice, index) => (
          <button key={choice} onClick={() => setAnswer(index)} style={{
            ...choiceStyle,
            borderColor: answer === index ? "var(--accent)" : "var(--stroke)",
            background: answer === index ? "var(--accent-soft)" : "var(--surface)",
          }}>{choice}</button>
        ))}
      </div>
      <Button variant="primary" full disabled={answer == null || submit.isPending} onClick={() => submit.mutate()}>
        {submit.isPending ? "Kontrol ediliyor…" : lesson.due_for_review ? "Bilgimi Kontrol Et" : "Tekrar Çöz"}
      </Button>
    </Card>
  );
}

function DisciplineLab({
  quests,
  refresh,
  flash,
}: {
  quests: GamificationExperience["discipline_quests"];
  refresh: () => void;
  flash: (message: string) => void;
}) {
  const [code, setCode] = useState(quests[0]?.code ?? "");
  const [note, setNote] = useState("");
  const submit = useMutation({
    mutationFn: () => api.completeDiscipline(code, note),
    onSuccess: (result) => {
      flash(result.awarded ? `Süreç değerlendirmesi tamam · +${result.xp_gained} XP` : "Bu değerlendirme bugün zaten tamamlandı.");
      setNote("");
      refresh();
    },
    onError: (error: Error) => flash(error.message || "Değerlendirme kaydedilemedi"),
  });
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ fontWeight: 800 }}>Portföy Disiplini</div>
      <div style={{ fontSize: 12, color: "var(--text-dim)", margin: "3px 0 12px" }}>
        Sonuca değil, karar sürecine puan verilir.
      </div>
      <select value={code} onChange={(event) => setCode(event.target.value)} style={inputStyle}>
        {quests.map((quest) => <option key={quest.code} value={quest.code}>{quest.title}</option>)}
      </select>
      <div style={{ fontSize: 11, color: "var(--text-faint)", margin: "8px 0" }}>
        {quests.find((quest) => quest.code === code)?.description}
      </div>
      <textarea value={note} onChange={(event) => setNote(event.target.value)} rows={3} maxLength={1000}
        placeholder="Kararındaki risk, tez veya plan sapmasını en az 20 karakterle değerlendir…" style={inputStyle} />
      <Button variant="primary" full disabled={note.trim().length < 20 || submit.isPending} onClick={() => submit.mutate()}>
        {submit.isPending ? "Kaydediliyor…" : "Değerlendirmeyi Tamamla"}
      </Button>
    </Card>
  );
}

function SeasonCard({ experience }: { experience: GamificationExperience }) {
  const season = experience.season;
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "start" }}>
        <div>
          <div style={{ fontWeight: 800 }}>{season.name}</div>
          <div style={{ fontSize: 12, color: "var(--text-dim)" }}>{season.code} · kişisel sezon ilerlemesi</div>
        </div>
        <span style={pillStyle}>{season.points} puan</span>
      </div>
      <div style={{ fontSize: 11, color: "var(--text-faint)", margin: "10px 0" }}>{season.scoring}</div>
      <div style={{ color: "var(--text-dim)", fontSize: 11 }}>Sezon bitişi: {new Date(season.ends_at).toLocaleDateString("tr-TR", { day: "numeric", month: "long", year: "numeric" })}</div>
    </Card>
  );
}

function PaperLab({
  trades,
  refresh,
  flash,
}: {
  trades: PaperTrade[];
  refresh: () => void;
  flash: (message: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [code, setCode] = useState("GRAM_GOLD");
  const [side, setSide] = useState<"buy" | "sell">("buy");
  const [quantity, setQuantity] = useState("1");
  const [price, setPrice] = useState("");
  const [risk, setRisk] = useState("1");
  const [thesis, setThesis] = useState("");
  const [invalidation, setInvalidation] = useState("");
  const [reviewID, setReviewID] = useState("");
  const [followed, setFollowed] = useState(true);
  const [retrospective, setRetrospective] = useState("");
  const create = useMutation({
    mutationFn: () => api.createPaperTrade({
      client_id: crypto.randomUUID(), code, side, quantity: Number(quantity),
      entry_price: Number(price), thesis, risk_budget_pct: Number(risk),
      invalidation, occurred_at: new Date().toISOString(),
    }),
    onSuccess: () => {
      flash("Paper karar günlüğüne eklendi. Getiri değil süreç izlenecek.");
      setOpen(false); setThesis(""); setInvalidation(""); refresh();
    },
    onError: (error: Error) => flash(error.message || "Paper işlem eklenemedi"),
  });
  const review = useMutation({
    mutationFn: () => api.reviewPaperTrade(reviewID, followed, retrospective),
    onSuccess: (result) => {
      flash(`Retrospektif tamamlandı · +${result.xp_gained} XP`);
      setReviewID(""); setRetrospective(""); refresh();
    },
    onError: (error: Error) => flash(error.message || "Retrospektif kaydedilemedi"),
  });
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
        <div>
          <div style={{ fontWeight: 800 }}>Paper Karar Laboratuvarı</div>
          <div style={{ fontSize: 12, color: "var(--text-dim)" }}>Tez · risk · geçersizleşme · retrospektif</div>
        </div>
        <Button variant="soft" onClick={() => setOpen(!open)}>{open ? "Kapat" : "Yeni Karar"}</Button>
      </div>
      {open && (
        <div style={{ display: "grid", gap: 8, marginTop: 12 }}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
            <input value={code} onChange={(event) => setCode(event.target.value.toUpperCase())} placeholder="Kod" style={inputStyle} />
            <select value={side} onChange={(event) => setSide(event.target.value as typeof side)} style={inputStyle}>
              <option value="buy">Alış senaryosu</option><option value="sell">Satış senaryosu</option>
            </select>
            <input value={quantity} onChange={(event) => setQuantity(event.target.value)} type="number" min="0" placeholder="Miktar" style={inputStyle} />
            <input value={price} onChange={(event) => setPrice(event.target.value)} type="number" min="0" placeholder="Giriş fiyatı" style={inputStyle} />
            <input value={risk} onChange={(event) => setRisk(event.target.value)} type="number" min="0.1" max="5" step="0.1" placeholder="Risk %" style={inputStyle} />
          </div>
          <textarea value={thesis} onChange={(event) => setThesis(event.target.value)} rows={2} placeholder="Tez (en az 20 karakter)" style={inputStyle} />
          <textarea value={invalidation} onChange={(event) => setInvalidation(event.target.value)} rows={2} placeholder="Tezi geçersiz kılacak koşul" style={inputStyle} />
          <Button variant="primary" full disabled={create.isPending || thesis.trim().length < 20 || invalidation.trim().length < 10 || Number(price) <= 0}
            onClick={() => create.mutate()}>Kararı Kilitle</Button>
        </div>
      )}
      <div style={{ display: "grid", gap: 8, marginTop: 12 }}>
        {trades.slice(0, 5).map((trade) => (
          <div key={trade.id} style={{ padding: 11, borderRadius: 10, background: "var(--surface-hi)" }}>
            <div style={{ display: "flex", justifyContent: "space-between", gap: 8 }}>
              <strong>{trade.code} · {trade.side === "buy" ? "Alış" : "Satış"}</strong>
              <span style={{ fontSize: 11, color: trade.reviewed_at ? "var(--up)" : "var(--text-dim)" }}>
                {trade.reviewed_at ? `Süreç ${trade.process_score}/100` : "Retrospektif bekliyor"}
              </span>
            </div>
            <div style={{ fontSize: 11, color: "var(--text-dim)", marginTop: 3 }}>{trade.thesis}</div>
            {!trade.reviewed_at && (
              <button style={linkButtonStyle} onClick={() => setReviewID(reviewID === trade.id ? "" : trade.id)}>Kararı değerlendir</button>
            )}
            {reviewID === trade.id && (
              <div style={{ display: "grid", gap: 8, marginTop: 8 }}>
                <label style={{ fontSize: 12, color: "var(--text-dim)" }}>
                  <input type="checkbox" checked={followed} onChange={(event) => setFollowed(event.target.checked)} /> Planıma uydum
                </label>
                <textarea value={retrospective} onChange={(event) => setRetrospective(event.target.value)}
                  rows={2} placeholder="Sonuçtan bağımsız süreç değerlendirmesi…" style={inputStyle} />
                <Button variant="primary" full disabled={retrospective.trim().length < 20 || review.isPending} onClick={() => review.mutate()}>
                  Retrospektifi Tamamla
                </Button>
              </div>
            )}
          </div>
        ))}
        {trades.length === 0 && <div style={{ fontSize: 12, color: "var(--text-dim)" }}>Henüz paper karar kaydı yok.</div>}
      </div>
    </Card>
  );
}

function PrivacyAndGroups({
  experience,
  refresh,
  flash,
}: {
  experience: GamificationExperience;
  refresh: () => void;
  flash: (message: string) => void;
}) {
  const [privacy, setPrivacy] = useState<GamificationPrivacy>(experience.privacy);
  const [groupName, setGroupName] = useState("");
  const [invite, setInvite] = useState("");
  const save = useMutation({
    mutationFn: () => api.saveGamificationPrivacy(privacy),
    onSuccess: (result) => { setPrivacy(result); flash("Gizlilik tercihleri kaydedildi"); refresh(); },
    onError: (error: Error) => flash(error.message || "Tercihler kaydedilemedi"),
  });
  const create = useMutation({
    mutationFn: () => api.createStudyGroup(groupName),
    onSuccess: (group) => { flash(`Grup hazır · davet kodu ${group.invite_code}`); setGroupName(""); refresh(); },
    onError: (error: Error) => flash(error.message || "Grup oluşturulamadı"),
  });
  const join = useMutation({
    mutationFn: () => api.joinStudyGroup(invite),
    onSuccess: () => { flash("Çalışma grubuna katıldın"); setInvite(""); refresh(); },
    onError: (error: Error) => flash(error.message || "Gruba katılınamadı"),
  });
  const toggle = (key: keyof GamificationPrivacy) => {
    setPrivacy((current) => ({ ...current, [key]: !current[key] }));
  };
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ fontWeight: 800 }}>Gizlilik ve Çalışma Grupları</div>
      <div style={{ fontSize: 12, color: "var(--text-dim)", margin: "3px 0 12px" }}>
        Para, pozisyon ve getiri hiçbir sosyal görünümde paylaşılmaz.
      </div>
      <input value={privacy.alias} maxLength={40} onChange={(event) => setPrivacy({ ...privacy, alias: event.target.value })}
        placeholder="Görünecek takma ad" style={inputStyle} />
      <div style={{ display: "grid", gap: 8, margin: "10px 0" }}>
        <Toggle label="Kişiselleştirilmiş öğrenme yolu" checked={privacy.personalized_learning} onChange={() => toggle("personalized_learning")} />
        <Toggle label="Öğrenme bildirimleri" checked={privacy.learning_notifications} onChange={() => toggle("learning_notifications")} />
        <Toggle label="Gizlilik güvenli çalışma grupları" checked={privacy.social_opt_in} onChange={() => toggle("social_opt_in")} />
      </div>
      <Button variant="soft" full disabled={save.isPending} onClick={() => save.mutate()}>Tercihleri Kaydet</Button>
      {privacy.social_opt_in && (
        <>
          <div style={{ display: "grid", gridTemplateColumns: "1fr auto", gap: 8, marginTop: 12 }}>
            <input value={groupName} onChange={(event) => setGroupName(event.target.value)} placeholder="Yeni grup adı" style={inputStyle} />
            <Button variant="soft" disabled={groupName.trim().length < 3 || create.isPending} onClick={() => create.mutate()}>Oluştur</Button>
            <input value={invite} onChange={(event) => setInvite(event.target.value.toUpperCase())} placeholder="Davet kodu" style={inputStyle} />
            <Button variant="soft" disabled={invite.trim().length < 6 || join.isPending} onClick={() => join.mutate()}>Katıl</Button>
          </div>
          <div style={{ display: "grid", gap: 6, marginTop: 10 }}>
            {experience.study_groups.map((group) => (
              <div key={group.id} style={{ display: "flex", justifyContent: "space-between", fontSize: 12, padding: 9, background: "var(--surface-hi)", borderRadius: 8 }}>
                <span><strong>{group.name}</strong> · {group.member_count}/8 kişi</span>
                <span className="mono" style={{ color: "var(--accent-hi)" }}>{group.invite_code}</span>
              </div>
            ))}
          </div>
        </>
      )}
    </Card>
  );
}

function Modal({ title, subtitle, onClose, children }: { title: string; subtitle: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div style={modalBackdropStyle} onClick={onClose}>
      <div style={modalStyle} onClick={(event) => event.stopPropagation()}>
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
          <div><div style={{ fontSize: 19, fontWeight: 800 }}>{title}</div><div style={{ fontSize: 12, color: "var(--text-dim)" }}>{subtitle}</div></div>
          <button onClick={onClose} style={linkButtonStyle}>Kapat</button>
        </div>
        <div style={{ display: "grid", gap: 12, marginTop: 16 }}>{children}</div>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <label style={{ display: "grid", gap: 6, fontSize: 12, color: "var(--text-dim)" }}>{label}{children}</label>;
}

function Toggle({ label, checked, onChange }: { label: string; checked: boolean; onChange: () => void }) {
  return <label style={{ display: "flex", justifyContent: "space-between", gap: 12, fontSize: 13 }}>
    <span>{label}</span><input type="checkbox" checked={checked} onChange={onChange} />
  </label>;
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return <div style={{ padding: 10, borderRadius: 10, background: "var(--surface-hi)" }}>
    <div style={{ fontSize: 10, color: "var(--text-faint)" }}>{label}</div>
    <div className="mono" style={{ fontSize: 17, fontWeight: 800, marginTop: 2 }}>{value}</div>
  </div>;
}

const inputStyle: React.CSSProperties = {
  width: "100%", boxSizing: "border-box", background: "var(--surface-hi)", color: "var(--text)",
  border: "1px solid var(--stroke)", borderRadius: 10, padding: "10px 11px", font: "inherit",
};
const PLAN_ASSET_KINDS = [
  { id: "gold", label: "Altın" },
  { id: "currency", label: "Döviz" },
  { id: "crypto", label: "Kripto Para" },
  { id: "stock", label: "Hisse Senedi" },
];
const pickerAssetButtonStyle: React.CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  gap: 10,
  padding: "10px 11px",
  border: "1px solid var(--stroke)",
  borderRadius: 9,
  color: "var(--text)",
  cursor: "pointer",
  textAlign: "left",
};
const chipStyle: React.CSSProperties = {
  whiteSpace: "nowrap", background: "var(--surface-hi)", border: "1px solid var(--stroke)",
  borderRadius: 999, padding: "7px 10px", fontSize: 11, cursor: "pointer",
};
const choiceStyle: React.CSSProperties = {
  textAlign: "left", color: "var(--text)", border: "1px solid var(--stroke)", borderRadius: 10,
  padding: "10px 11px", cursor: "pointer", marginBottom: 0,
};
const pillStyle: React.CSSProperties = {
  borderRadius: 999, padding: "6px 9px", background: "var(--accent-soft)",
  color: "var(--accent-hi)", fontSize: 11, fontWeight: 800, whiteSpace: "nowrap",
};
const linkButtonStyle: React.CSSProperties = {
  background: "transparent", color: "var(--accent-hi)", border: 0, padding: "6px 0",
  fontWeight: 700, fontSize: 12, cursor: "pointer",
};
const modalBackdropStyle: React.CSSProperties = {
  position: "fixed", inset: 0, zIndex: 400, background: "rgba(0,0,0,.7)", display: "grid",
  placeItems: "center", padding: 16, overflowY: "auto",
};
const modalStyle: React.CSSProperties = {
  width: "min(560px,100%)", maxHeight: "90vh", overflowY: "auto", background: "var(--surface)",
  border: "1px solid var(--stroke-strong)", borderRadius: 18, padding: 20, boxShadow: "var(--shadow)",
};

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
      <Tile label="Tamamlanan plan" value={`${p.total_check_ins}`} />
    </div>
  );
}

function DailyReward({ p, planned, onClaim }: { p: GamProfile; planned: boolean; onClaim: () => void }) {
  if (planned) {
    return (
      <Card style={{ padding: 16, display: "flex", alignItems: "center", gap: 12 }}>
        <span style={{ fontSize: 24 }}>✅</span>
        <div>
          <div style={{ fontWeight: 700 }}>Bugünün planı hazır</div>
          <div style={{ fontSize: 12, color: "var(--text-dim)" }}>
            Risk sınırın ve öğrenme hedefin kaydedildi. Seri: {p.current_streak} gün
          </div>
        </div>
      </Card>
    );
  }
  return (
    <Button variant="primary" full onClick={onClaim}>
      🧭 Günü Planla
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

function Badges({ p, onSelect }: { p: GamProfile; onSelect: (badge: GamBadge) => void }) {
  const earned = p.badges.filter((b) => b.earned).length;
  return (
    <Card style={{ padding: 18 }}>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 14 }}>
        <div><div style={{ fontWeight: 700 }}>Rozet Koleksiyonu</div><div style={{ marginTop: 2, color: "var(--text-faint)", fontSize: 10 }}>Kazanma koşulunu görmek için bir rozete dokun</div></div>
        <div style={{ fontSize: 12, color: "var(--text-dim)", fontWeight: 700 }}>
          {earned}/{p.badges.length}
        </div>
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill,minmax(76px,1fr))", gap: 16 }}>
        {p.badges.map((b) => (
          <button key={b.code} type="button" onClick={() => onSelect(b)} aria-label={`${b.name} rozetinin detayları`} style={{ padding: 0, border: 0, color: "inherit", background: "transparent", textAlign: "center", cursor: "pointer" }}>
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
          </button>
        ))}
      </div>
    </Card>
  );
}

function BadgeDetail({ badge, onClose }: { badge: GamBadge; onClose: () => void }) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    closeRef.current?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onClose();
        return;
      }
      if (event.key !== "Tab" || !dialogRef.current) return;
      const focusable = Array.from(
        dialogRef.current.querySelectorAll<HTMLElement>(
          "button, [href], input, select, textarea, [tabindex]:not([tabindex='-1'])",
        ),
      ).filter((element) => !element.hasAttribute("disabled"));
      if (!focusable.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      previous?.focus();
    };
  }, [onClose]);

  return (
    <div onClick={onClose} style={{ position: "fixed", inset: 0, zIndex: 280, display: "grid", placeItems: "center", padding: 20, background: "rgba(0,0,0,.68)" }}>
      <div ref={dialogRef} role="dialog" aria-modal="true" aria-labelledby="badge-detail-title" aria-describedby="badge-detail-description" onClick={(event) => event.stopPropagation()} style={{ width: "min(410px,100%)", padding: 24, border: "1px solid var(--stroke-strong)", borderRadius: 16, background: "var(--surface)", boxShadow: "var(--shadow)", textAlign: "center" }}>
        <div style={{ width: 72, height: 72, display: "grid", placeItems: "center", margin: "0 auto 14px", borderRadius: 36, background: badge.earned ? "var(--accent-soft)" : "var(--surface-hi)", fontSize: 36, filter: badge.earned ? "none" : "grayscale(1)" }}>{badge.icon}</div>
        <h2 id="badge-detail-title" style={{ margin: 0, fontSize: 20 }}>{badge.name}</h2>
        <div style={{ margin: "7px 0 16px", color: badge.earned ? "var(--up)" : "var(--text-faint)", fontSize: 11, fontWeight: 750, textTransform: "uppercase", letterSpacing: ".08em" }}>{badge.earned ? "Kazanıldı" : "Henüz kilitli"}</div>
        <div id="badge-detail-description" style={{ padding: 13, borderRadius: 10, background: "var(--bg-elev)", color: "var(--text-dim)", fontSize: 13, lineHeight: 1.55, textAlign: "left" }}>
          <strong style={{ display: "block", marginBottom: 4, color: "var(--text)", fontSize: 11 }}>{badge.earned ? "Rozet koşulu" : "Nasıl kazanılır?"}</strong>
          {badge.description}
        </div>
        {badge.earned_at && <div style={{ marginTop: 12, color: "var(--text-faint)", fontSize: 11 }}>Kazanma tarihi: {new Date(badge.earned_at).toLocaleDateString("tr-TR", { day: "numeric", month: "long", year: "numeric" })}</div>}
        <div style={{ marginTop: 18 }}><button ref={closeRef} type="button" onClick={onClose} style={{ width: "100%", minHeight: 40, border: "1px solid var(--stroke)", borderRadius: 9, color: "var(--text)", background: "var(--surface-hi)", fontWeight: 700, cursor: "pointer" }}>Kapat</button></div>
      </div>
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
