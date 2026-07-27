import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, ApiError } from "../api";
import { trDate } from "../format";
import { Icon } from "../Icon";
import type { Campaign, CampaignPreview, PerformanceFilter, TargetingPlan } from "../types";
import { EmptyState, Modal, StatusPill } from "../UI";

type CampaignsResponse = {
  items: Campaign[] | null;
  total: number;
};

function stringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === "string");
}

function finiteNumber(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

function normalizePerformance(value: unknown): PerformanceFilter | null {
  if (!value || typeof value !== "object") return null;
  const filter = value as Partial<PerformanceFilter>;
  if (filter.direction !== "decliners" && filter.direction !== "gainers") return null;
  return {
    direction: filter.direction,
    lookback_days: finiteNumber(filter.lookback_days),
    top_n: finiteNumber(filter.top_n),
    min_abs_change_pct: finiteNumber(filter.min_abs_change_pct),
  };
}

function normalizePreview(value: CampaignPreview): CampaignPreview {
  const plan = (value?.plan || {}) as Partial<TargetingPlan>;
  return {
    ...value,
    name: typeof value?.name === "string" ? value.name : "Bildirim kampanyası",
    summary: typeof value?.summary === "string" ? value.summary : "",
    title: typeof value?.title === "string" ? value.title : "",
    body: typeof value?.body === "string" ? value.body : "",
    deep_link: typeof value?.deep_link === "string" ? value.deep_link : "",
    recipient_count: finiteNumber(value?.recipient_count),
    total_matched: finiteNumber(value?.total_matched),
    approval_fingerprint: typeof value?.approval_fingerprint === "string" ? value.approval_fingerprint : "",
    plan: {
      audience: typeof plan.audience === "string" ? plan.audience : "all_users",
      platforms: stringArray(plan.platforms),
      plan_codes: stringArray(plan.plan_codes),
      asset_codes: stringArray(plan.asset_codes),
      asset_sources: stringArray(plan.asset_sources),
      performance: normalizePerformance(plan.performance),
      registered_within_days: finiteNumber(plan.registered_within_days),
      inactive_for_days: finiteNumber(plan.inactive_for_days),
      max_recipients: finiteNumber(plan.max_recipients),
    },
    matched_assets: Array.isArray(value?.matched_assets)
      ? value.matched_assets
        .filter((asset) => asset && typeof asset.code === "string")
        .map((asset) => ({ code: asset.code, change_pct: finiteNumber(asset.change_pct) }))
      : [],
    sample_users: Array.isArray(value?.sample_users)
      ? value.sample_users
        .filter((user) => user && typeof user.id === "string")
        .map((user) => ({
          id: user.id,
          email: typeof user.email === "string" ? user.email : "",
          display_name: typeof user.display_name === "string" ? user.display_name : "",
          device_platform: typeof user.device_platform === "string" ? user.device_platform : "unknown",
        }))
      : [],
  };
}

export function Campaigns() {
  const queryClient = useQueryClient();
  const [prompt, setPrompt] = useState("");
  const [preview, setPreview] = useState<CampaignPreview | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [confirm, setConfirm] = useState(false);
  const [cancelCandidate, setCancelCandidate] = useState<Campaign | null>(null);
  const [confirmationText, setConfirmationText] = useState("");
  const [error, setError] = useState("");

  const campaigns = useQuery({
    queryKey: ["admin-campaigns"],
    queryFn: async () => {
      const value = await api<CampaignsResponse>("/admin/campaigns?page_size=50");
      return {
        items: Array.isArray(value.items) ? value.items : [],
        total: finiteNumber(value.total),
      };
    },
    refetchInterval: 10_000,
  });
  const previewMutation = useMutation({
    mutationFn: () => api<CampaignPreview>("/admin/campaigns/preview", { method: "POST", body: JSON.stringify({ prompt }) }),
    onSuccess: (value) => {
      const safeValue = normalizePreview(value);
      setPreview(safeValue); setTitle(safeValue.title); setBody(safeValue.body); setError("");
      queryClient.invalidateQueries({ queryKey: ["admin-campaigns"] });
    },
    onError: (value) => setError(value instanceof ApiError ? value.message : "Önizleme oluşturulamadı"),
  });
  const resumeMutation = useMutation({
    mutationFn: (campaign: Pick<Campaign, "id" | "prompt">) =>
      api<CampaignPreview>(`/admin/campaigns/${campaign.id}/preview`, { method: "POST" }),
    onSuccess: (value, campaign) => {
      const safeValue = normalizePreview(value);
      setPreview(safeValue);
      setPrompt(campaign.prompt);
      setTitle(safeValue.title);
      setBody(safeValue.body);
      setError("");
      queryClient.invalidateQueries({ queryKey: ["admin-campaigns"] });
      requestAnimationFrame(() => document.querySelector(".preview-workspace")?.scrollIntoView({ behavior: "smooth", block: "start" }));
    },
    onError: (value) => setError(value instanceof ApiError ? value.message : "Kampanya taslağı açılamadı"),
  });
  const approveMutation = useMutation({
    mutationFn: () => api<Campaign>(`/admin/campaigns/${preview!.campaign_id}/approve`, {
      method: "POST",
      body: JSON.stringify({
        expected_recipients: preview!.recipient_count,
        fingerprint: preview!.approval_fingerprint,
        title, body, deep_link: preview!.deep_link,
      }),
    }),
    onSuccess: () => {
      setPreview(null); setPrompt(""); setConfirm(false); setConfirmationText("");
      queryClient.invalidateQueries({ queryKey: ["admin-campaigns"] });
    },
    onError: (value) => {
      setConfirm(false);
      setError(value instanceof ApiError ? value.message : "Kampanya onaylanamadı");
    },
  });
  const cancelMutation = useMutation({
    mutationFn: (campaignID: string) => api<void>(`/admin/campaigns/${campaignID}/cancel`, { method: "POST" }),
    onSuccess: () => {
      setCancelCandidate(null);
      queryClient.invalidateQueries({ queryKey: ["admin-campaigns"] });
    },
    onError: (value) => {
      setCancelCandidate(null);
      setError(value instanceof ApiError ? value.message : "Kampanya iptal edilemedi");
    },
  });
  useEffect(() => {
    if (!confirm) setConfirmationText("");
  }, [confirm]);

  return <div className="campaign-page">
    <section className="campaign-composer panel">
      <div className="composer-intro">
        <span className="ai-orb"><Icon name="sparkles" size={24} /></span>
        <div><span className="kicker">GÜVENLİ AI ORKESTRASYONU</span><h2>Talebini anlat, hedef kitleyi birlikte doğrulayalım</h2><p>AI yalnızca izinli hedefleme kurallarını üretir. SQL çalıştırmaz; gönderim daima önizleme ve açık onay gerektirir.</p></div>
      </div>
      <label className="prompt-box">
        <textarea value={prompt} onChange={(e) => setPrompt(e.target.value)} maxLength={2000} placeholder="Örn: Son bir haftada en çok düşen beş varlığı favorilerine ekleyen kullanıcılara dikkatli olmalarını söyleyen bir bildirim hazırla." />
        <span>{prompt.length}/2000</span>
        <button disabled={prompt.trim().length < 8 || previewMutation.isPending} onClick={() => previewMutation.mutate()}>{previewMutation.isPending ? <span className="spinner" /> : <Icon name="sparkles" size={16} />} Hedef kitleyi analiz et</button>
      </label>
      {error && <div className="form-error wide">{error}</div>}
    </section>

    {preview && <section className="preview-workspace">
      <div className="panel preview-summary">
        <div className="panel-heading">
          <div><span className="kicker">GÖNDERİM ÖNİZLEMESİ</span><h2>{preview.name}</h2></div>
          <div className="preview-heading-actions">
            <button className="draft-refresh" disabled={resumeMutation.isPending} onClick={() => resumeMutation.mutate({ id: preview.campaign_id, prompt })}>
              {resumeMutation.isPending ? <span className="spinner" /> : <Icon name="activity" size={14} />} Hedefi yenile
            </button>
            <button className="preview-close" aria-label="Taslağı kapat" onClick={() => setPreview(null)}><Icon name="close" size={14} /></button>
            <span className="status-pill draft">Onay bekliyor</span>
          </div>
        </div>
        <p>{preview.summary}</p>
        <div className="audience-number"><span><small>Hedef alıcı</small><strong>{preview.recipient_count.toLocaleString("tr-TR")}</strong></span><span><small>Toplam eşleşme</small><strong>{preview.total_matched.toLocaleString("tr-TR")}</strong></span><span><small>Üst sınır</small><strong>{preview.plan.max_recipients.toLocaleString("tr-TR")}</strong></span></div>
        <div className="rule-chips">
          <span>Kitle: {preview.plan.audience}</span>
          {preview.plan.platforms.map((x) => <span key={x}>Platform: {x}</span>)}
          {preview.plan.plan_codes.map((x) => <span key={x}>Paket: {x}</span>)}
          {preview.plan.asset_sources.map((x) => <span key={x}>Kaynak: {x}</span>)}
          {preview.plan.performance && <span>{preview.plan.performance.lookback_days} gün · {preview.plan.performance.direction} · ilk {preview.plan.performance.top_n}</span>}
        </div>
        {preview.matched_assets.length > 0 && <div className="asset-preview">{preview.matched_assets.map((asset) => <div key={asset.code}><strong>{asset.code}</strong><span className={asset.change_pct < 0 ? "negative" : "positive"}>{asset.change_pct > 0 ? "+" : ""}{asset.change_pct.toFixed(2)}%</span></div>)}</div>}
      </div>
      <div className="panel message-editor">
        <div className="panel-heading"><div><span className="kicker">İÇERİK</span><h2>Bildirim metni</h2></div></div>
        <label>Başlık<input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={128} /><small>{title.length}/128</small></label>
        <label>Mesaj<textarea value={body} onChange={(e) => setBody(e.target.value)} maxLength={512} /><small>{body.length}/512</small></label>
        <div className="phone-preview"><div className="phone-app"><span className="brand-mark mini"><span>T</span><span>F</span></span><span><strong>{title}</strong><small>{body}</small></span><time>şimdi</time></div></div>
        <button className="primary-button danger-confirm" disabled={!preview.recipient_count || !title.trim() || !body.trim()} onClick={() => setConfirm(true)}><Icon name="campaign" size={16} /> Onaya geç</button>
      </div>
      <div className="panel sample-users">
        <div className="panel-heading"><div><span className="kicker">ÖRNEKLEM</span><h2>Eşleşen kullanıcılar</h2></div></div>
        {preview.sample_users.length === 0
          ? <p className="sample-empty">Mevcut kurallarla eşleşen kullanıcı yok. Hedef kitleyi yenileyebilir veya yeni bir taleple farklı kurallar oluşturabilirsin.</p>
          : preview.sample_users.map((user) => <div key={user.id}><span className="table-avatar">{(user.display_name || user.email).slice(0, 2).toUpperCase()}</span><span><strong>{user.display_name || "İsimsiz kullanıcı"}</strong><small>{user.email}</small></span><StatusPill value={user.device_platform} /></div>)}
      </div>
    </section>}

    <section className="panel campaign-history">
      <div className="panel-heading"><div><span className="kicker">GEÇMİŞ</span><h2>Bildirim kampanyaları</h2></div><span className="panel-chip">{campaigns.data?.total || 0} kampanya</span></div>
      {!campaigns.data?.items.length ? <EmptyState icon="campaign" title="Henüz kampanya yok" text="İlk hedefli bildirimi yukarıdaki alandan planlayın." /> : <div className="data-table campaign-table">
        <div className="table-head"><span>Kampanya</span><span>Durum</span><span>Alıcı</span><span>İşlendi / hata</span><span>Tarih</span><span>İşlemler</span></div>
        {campaigns.data.items.map((item) => {
          const isDraft = item.status === "draft";
          const isOpening = resumeMutation.isPending && resumeMutation.variables?.id === item.id;
          return <div
            className={`table-row campaign-row${isDraft ? " resumable" : ""}`}
            key={item.id}
            role={isDraft ? "button" : undefined}
            tabIndex={isDraft ? 0 : undefined}
            onClick={() => isDraft && !resumeMutation.isPending && resumeMutation.mutate(item)}
            onKeyDown={(event) => {
              if (isDraft && !resumeMutation.isPending && (event.key === "Enter" || event.key === " ")) {
                event.preventDefault();
                resumeMutation.mutate(item);
              }
            }}
          >
            <span><strong>{item.name}</strong><small>{item.title}</small></span>
            <span><StatusPill value={item.status} /></span>
            <span><strong>{(item.resolved_recipients || item.estimated_recipients).toLocaleString("tr-TR")}</strong><small>kullanıcı</small></span>
            <span><strong>{item.processed_recipients} / {item.failed_recipients}</strong><small>gelen kutusu / hata</small></span>
            <span><strong>{trDate(item.created_at)}</strong><small>{item.completed_at ? `Bitti: ${trDate(item.completed_at)}` : isDraft ? "Devam etmeye hazır" : "—"}</small></span>
            <span className="campaign-actions">
              {isDraft && <button className="resume-inline" disabled={resumeMutation.isPending} onClick={(event) => { event.stopPropagation(); resumeMutation.mutate(item); }}>
                {isOpening ? <span className="spinner" /> : <Icon name="arrow" size={12} />} Devam et
              </button>}
              {["draft", "queued", "running"].includes(item.status) && <button className="cancel-inline" onClick={(event) => { event.stopPropagation(); setCancelCandidate(item); }}>İptal</button>}
            </span>
          </div>;
        })}
      </div>}
    </section>

    {confirm && preview && <Modal onClose={() => setConfirm(false)}>
      <span className="modal-icon warning"><Icon name="campaign" size={24} /></span>
      <h2>Gönderimi açıkça onayla</h2>
      <p><strong>{preview.recipient_count.toLocaleString("tr-TR")}</strong> kullanıcının uygulama gelen kutusuna bildirim yazılacak ve uygun cihazlara APNs/FCM gönderimi denenecek. Bu işlem geri alınamaz.</p>
      <label className="confirm-label">Devam etmek için <b>GÖNDER</b> yaz<input autoFocus value={confirmationText} onChange={(e) => setConfirmationText(e.target.value)} /></label>
      <div className="modal-actions"><button className="secondary-button" onClick={() => setConfirm(false)}>Vazgeç</button><button className="primary-button" disabled={confirmationText !== "GÖNDER" || approveMutation.isPending} onClick={() => approveMutation.mutate()}>{approveMutation.isPending ? <span className="spinner" /> : "Kuyruğa al ve gönder"}</button></div>
    </Modal>}
    {cancelCandidate && <Modal onClose={() => setCancelCandidate(null)}>
      <span className="modal-icon warning"><Icon name="close" size={24} /></span>
      <h2>Kampanyayı iptal et</h2>
      <p><strong>{cancelCandidate.name}</strong> için henüz işlenmemiş alıcılar durdurulacak. Daha önce kullanıcı gelen kutusuna yazılan bildirimler geri alınamaz.</p>
      <div className="modal-actions"><button className="secondary-button" onClick={() => setCancelCandidate(null)}>Vazgeç</button><button className="primary-button destructive" disabled={cancelMutation.isPending} onClick={() => cancelMutation.mutate(cancelCandidate.id)}>{cancelMutation.isPending ? <span className="spinner" /> : "İptal etmeyi onayla"}</button></div>
    </Modal>}
  </div>;
}
