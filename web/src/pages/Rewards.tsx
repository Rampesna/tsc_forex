import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../lib/endpoints";
import type {
  CatalogItem,
  GamBadge,
  GamificationExperience,
  GamificationPrivacy,
  GamProfile,
  LeaderEntry,
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

  const profileQ = useQuery({ queryKey: ["gamification"], queryFn: api.gamification, refetchOnWindowFocus: false });
  const boardQ = useQuery({ queryKey: ["leaderboard"], queryFn: () => api.leaderboard(20), refetchOnWindowFocus: false });
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
          <DailyReward
            p={p}
            planned={Boolean(experienceQ.data?.daily_plan)}
            onClaim={() => setShowPlan(true)}
          />
          {experienceQ.data && (
            <ExperienceCards
              experience={experienceQ.data}
              flash={flash}
              refresh={() => {
                qc.invalidateQueries({ queryKey: ["gamification-experience"] });
                qc.invalidateQueries({ queryKey: ["gamification"] });
                qc.invalidateQueries({ queryKey: ["leaderboard"] });
              }}
            />
          )}
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
      {showPlan && (
        <PlanEditor
          experience={experienceQ.data}
          catalog={catalog}
          onClose={() => setShowPlan(false)}
          onSaved={(res) => {
            qc.setQueryData(["gamification"], res.profile);
            qc.invalidateQueries({ queryKey: ["gamification-experience"] });
            qc.invalidateQueries({ queryKey: ["leaderboard"] });
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
  experience,
  refresh,
  flash,
}: {
  experience: GamificationExperience;
  refresh: () => void;
  flash: (message: string) => void;
}) {
  return (
    <>
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
      <LearningLab lessons={experience.lessons} refresh={refresh} flash={flash} />
      <DisciplineLab quests={experience.discipline_quests} refresh={refresh} flash={flash} />
      <SeasonCard experience={experience} />
      <PaperLab trades={experience.paper_trades} refresh={refresh} flash={flash} />
      <PrivacyAndGroups experience={experience} refresh={refresh} flash={flash} />
    </>
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
          <div style={{ fontSize: 12, color: "var(--text-dim)" }}>{season.code} · {season.league} Ligi</div>
        </div>
        <span style={pillStyle}>#{season.rank} · {season.points} puan</span>
      </div>
      <div style={{ fontSize: 11, color: "var(--text-faint)", margin: "10px 0" }}>{season.scoring}</div>
      <div style={{ display: "grid", gap: 6 }}>
        {season.standings.slice(0, 5).map((entry) => (
          <LeaderRow key={entry.user_id} e={entry} mine={false} />
        ))}
      </div>
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
        <Toggle label="Sezon sıralamasında görün" checked={privacy.league_opt_in} onChange={() => toggle("league_opt_in")} />
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
      <Tile label="Sıralama" value={p.rank > 0 ? `#${p.rank}` : "—"} />
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
