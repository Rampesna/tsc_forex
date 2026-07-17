import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { Button, Card } from "../components/common";
import { api } from "../lib/endpoints";
import { ApiError } from "../lib/api";
import { BRAND } from "../lib/config";
import {
  clearPersonalBrowserDataAfterDeletion,
  hasCurrentAIConsent,
  withdrawAIConsent,
} from "../lib/privacy";
import type { BillingPlan } from "../lib/types";

export function Settings() {
  const { logout, profile, authed } = useAuth();
  const nav = useNavigate();
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  const [aiConsent, setAIConsent] = useState(hasCurrentAIConsent);
  const entitlementQ = useQuery({
    queryKey: ["billing-entitlement"],
    queryFn: api.billingStatus,
    enabled: authed,
    retry: false,
    refetchOnWindowFocus: false,
  });
  const plansQ = useQuery({
    queryKey: ["billing-plans"],
    queryFn: api.billingPlans,
    retry: false,
    refetchOnWindowFocus: false,
  });
  const billingUnavailable = entitlementQ.isError || entitlementQ.isRefetchError;
  const plansUnavailable = plansQ.isError || plansQ.isRefetchError;
  const billing = billingUnavailable ? undefined : entitlementQ.data;
  const entitlement = billing?.entitlement;
  const plans = plansQ.data?.plans ?? [];
  const freeTrialReady = plansQ.data?.free_trial.ready === true;

  const deleteAccount = async () => {
    if (!window.confirm("Hesabın ve kişisel verilerin kalıcı olarak silinecek. Bu işlem App Store aboneliğini iptal etmez; yinelenen ödemeyi durdurmak için aboneliği ayrıca Apple üzerinden yönetmelisin. Bu işlem geri alınamaz. Devam edilsin mi?")) {
      return;
    }
    setDeleteError("");
    setDeleting(true);
    try {
      await api.deleteAccount();
      clearPersonalBrowserDataAfterDeletion();
      logout();
      window.location.replace("/");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Hesap silinemedi.";
      setDeleteError(
        error instanceof ApiError && error.code === "apple_reauthentication_required"
          ? "Apple ile bağlı hesabın güvenli silinmesi için iOS uygulamasında yeniden Apple ile doğrulama yapmalısın."
          : error instanceof ApiError && error.code === "subscription_recovery_token_required"
            ? "Satın alma geçmişi bulunan bu hesabın abonelik bağını güvenle koruyarak silinmesi için iOS uygulamasında Ayarlar > Hesabı Sil yolunu kullanmalısın."
            : message,
      );
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 18, maxWidth: 980 }}>
      <h2 style={{ margin: 0, fontSize: 20, fontWeight: 600, letterSpacing: "-0.02em" }}>Ayarlar</h2>

      {authed && (
        <Card style={{ padding: 20 }}>
          <Row label="Hesap" value={profile?.email ?? "-"} />
          <Row label="Sürüm" value="1.0.0" />
        </Card>
      )}

      {authed && (
        <Card style={{ padding: 20 }}>
          <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 16, marginBottom: 14 }}>
            <div>
              <div style={{ color: "var(--text-faint)", fontSize: 10, fontWeight: 750, letterSpacing: ".09em", textTransform: "uppercase" }}>Üyelik ve kullanım</div>
              <h3 style={{ margin: "6px 0 3px", fontSize: 20 }}>
                {entitlementQ.isLoading ? "Doğrulanıyor…" : billingUnavailable || !billing ? "Plan bilgisi kullanılamıyor" : billing.plan.name}
              </h3>
              <div style={{ color: "var(--text-dim)", fontSize: 12 }}>
                {entitlementQ.isLoading
                  ? "Sunucu doğrulaması bekleniyor…"
                  : billingUnavailable || !billing
                    ? "Yanlış plan veya kota göstermemek için durum tahmin edilmedi."
                    : entitlement?.status === "grace_period"
                      ? "Faturalandırma ödemesiz döneminde"
                      : entitlement?.status === "active"
                        ? "App Store tarafından doğrulanmış etkin abonelik"
                        : entitlement?.status === "expired"
                          ? "Abonelik süresi dolmuş"
                          : entitlement?.status === "revoked"
                            ? "Abonelik erişimi geri alınmış"
                        : "Doğrulanmış ücretsiz plan"}
              </div>
            </div>
            <span className={`settings-plan-status ${entitlement?.status === "active" ? "premium" : ""}`}>
              {entitlementQ.isLoading
                ? "Kontrol ediliyor"
                : billingUnavailable || !billing
                  ? "Kullanılamıyor"
                  : entitlement?.status === "active"
                    ? "Aktif"
                    : entitlement?.status === "grace_period"
                      ? "Ödemesiz dönem"
                      : entitlement?.status === "expired"
                        ? "Süresi doldu"
                        : entitlement?.status === "revoked"
                          ? "Geri alındı"
                      : "Ücretsiz"}
            </span>
          </div>
          {billing && <QuotaRow
            label={billing.plan.code === "free" ? "Kaynaklı analiz (ücretsiz hak yalnız doğrulanmış iOS cihazında)" : "Kaynaklı analiz"}
            used={billing.usage?.ai_research?.used ?? 0}
            limit={billing.usage?.ai_research?.limit ?? billing.plan.limits.ai_research}
            remaining={billing.plan.code === "free" && !freeTrialReady ? 0 : (billing.usage?.ai_research?.remaining ?? 0)}
          />}
          {billing?.plan.code === "free" && !plansQ.isLoading && !plansUnavailable && !freeTrialReady && (
            <div role="status" style={{ marginTop: 10, padding: 10, border: "1px solid var(--stroke)", borderRadius: 8, color: "var(--text-dim)", fontSize: 12 }}>
              Cihaz doğrulama servisi hazır olmadığı için tek kullanımlık ücretsiz araştırma şu anda kullanılamıyor; kullanılabilir olmayan bir hak gösterilmiyor.
            </div>
          )}
          {billingUnavailable && <div role="alert" style={{ padding: 10, border: "1px solid var(--stroke)", borderRadius: 8, color: "var(--down)", fontSize: 12 }}>Üyelik servisine ulaşılamadı. Mevcut planın ücretsiz varsayılmadı; biraz sonra yeniden dene.</div>}
          {billing?.usage?.ai_research?.period_end && <div style={{ marginTop: 8, color: "var(--text-faint)", fontSize: 11 }}>Kota yenilenmesi: {new Date(billing.usage.ai_research.period_end).toLocaleDateString("tr-TR")}</div>}
          {entitlement?.expires_at && <div style={{ marginTop: 5, color: "var(--text-faint)", fontSize: 11 }}>Erişim bitişi: {new Date(entitlement.expires_at).toLocaleDateString("tr-TR")}</div>}
          <div style={{ marginTop: 12, display: "flex", gap: 10, alignItems: "center", flexWrap: "wrap" }}>
            <a href={billing?.manage_url || "https://apps.apple.com/account/subscriptions"} target="_blank" rel="noreferrer" style={{ color: "var(--accent-hi)", fontSize: 12, fontWeight: 700 }}>Apple aboneliğini yönet</a>
            <span style={{ color: "var(--text-faint)", fontSize: 10 }}>Satın alma ve plan değişikliği yalnız iOS/App Store üzerinden yapılır.</span>
          </div>
        </Card>
      )}

      {authed && (
        <Card style={{ padding: 20 }}>
          <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", gap: 16 }}>
            <div>
              <div style={{ fontWeight: 700 }}>AI veri paylaşım izni</div>
              <p style={{ margin: "5px 0 0", maxWidth: 680, color: "var(--text-dim)", fontSize: 12, lineHeight: 1.5 }}>
                {aiConsent
                  ? "OpenAI'a yalnız seçtiğin varlık ve ilgili piyasa bağlamının gönderilmesine izin verdin. İzni geri çekmek gelecekteki AI isteklerini durdurur."
                  : "İzin verilmedi. İlk araştırmadan önce gönderilecek verileri açıklayan onay ekranı gösterilir; temel piyasa özellikleri etkilenmez."}
              </p>
            </div>
            <span className={`settings-plan-status ${aiConsent ? "premium" : ""}`}>{aiConsent ? "Etkin" : "Verilmedi"}</span>
          </div>
          {aiConsent && (
            <button
              type="button"
              onClick={() => {
                withdrawAIConsent();
                setAIConsent(false);
              }}
              style={{ marginTop: 12, padding: 0, border: 0, borderBottom: "1px solid currentColor", color: "var(--text-dim)", background: "transparent", fontSize: 12, fontWeight: 700, cursor: "pointer" }}
            >
              İzni geri çek
            </button>
          )}
        </Card>
      )}

      <section>
        <div style={{ marginBottom: 10 }}>
          <div style={{ fontWeight: 700 }}>Planlar</div>
          <div style={{ marginTop: 3, color: "var(--text-dim)", fontSize: 12 }}>Satın alma ve abonelik yönetimi yalnızca iOS uygulamasından yapılır.</div>
        </div>
        {plansQ.isLoading && <Card style={{ padding: 18, color: "var(--text-dim)" }}>Plan kataloğu yükleniyor…</Card>}
        {plansUnavailable && <Card style={{ padding: 18, color: "var(--down)" }}>Plan kataloğu şu anda kullanılamıyor. Fiyat veya hak tahmini gösterilmiyor.</Card>}
        {!plansUnavailable && !!plans.length && <div className="settings-plan-grid">
          {plans.map((plan) => (
            <article key={plan.code} className={`settings-plan-card ${plan.code === "pro" ? "highlighted" : ""} ${billing?.plan?.code === plan.code ? "current" : ""}`}>
              <div className="settings-plan-name">{plan.name}{billing?.plan?.code === plan.code && <span>Mevcut</span>}</div>
              <p>{typeof plan.price_hint_try === "number" ? `Yaklaşık ${plan.price_hint_try.toLocaleString("tr-TR", { style: "currency", currency: "TRY" })} / ay` : plan.price_hint_try || "Fiyat iOS uygulamasında gösterilir"}</p>
              <strong>{plan.limits.ai_research} analiz / ay</strong>
              <ul>{planFeatures(plan).map((feature) => <li key={feature}>✓ {feature}</li>)}</ul>
              <button type="button" disabled>{billing?.plan?.code === plan.code ? "Mevcut plan" : "Yalnız iOS uygulamasından"}</button>
            </article>
          ))}
        </div>}
      </section>

      <Card style={{ padding: 20 }}>
        <div style={{ fontWeight: 600, marginBottom: 6 }}>Hakkında</div>
        <div style={{ fontSize: 13, color: "var(--text-dim)", lineHeight: 1.5 }}>
          {BRAND.name} — canlı döviz, altın, kripto ve hisse piyasası. Fiyatlar bilgilendirme
          amaçlıdır, yatırım tavsiyesi değildir.
        </div>
      </Card>

      {authed ? (
        <>
          <Button variant="ghost" onClick={logout}>Çıkış yap</Button>
          <Card style={{ padding: 18, borderColor: "rgba(241,107,125,.25)" }}>
            <div style={{ fontWeight: 750, color: "var(--down)" }}>Hesabı ve verileri sil</div>
            <p style={{ margin: "6px 0 12px", color: "var(--text-dim)", fontSize: 12, lineHeight: 1.55 }}>
              Bu işlem favori, alarm, portföy, ödül ve AI arşivini geri alınamaz biçimde siler.
              Hesap silme App Store aboneliğini iptal etmez. Yinelenen ödemeyi durdurmak için önce{" "}
              <a href="https://apps.apple.com/account/subscriptions" target="_blank" rel="noreferrer" style={{ color: "var(--accent-hi)" }}>Apple Abonelikleri</a>
              {" "}ekranını kullan.
            </p>
            {profile?.provider === "apple" && <div style={{ marginBottom: 10, color: "#e6b565", fontSize: 11 }}>Apple ile bağlı hesaplarda Apple kimliğinin yeniden doğrulanması gerekir. Web isteği reddedilirse hesabı iOS uygulamasındaki Ayarlar ekranından sil.</div>}
            <button
              onClick={deleteAccount}
              disabled={deleting}
              style={{
                background: "transparent",
                border: "1px solid rgba(241,107,125,.28)",
                borderRadius: 8,
                color: "var(--down)",
                fontSize: 12,
                fontWeight: 700,
                cursor: deleting ? "not-allowed" : "pointer",
                padding: "9px 11px",
                opacity: deleting ? 0.6 : 1,
              }}
            >
              {deleting ? "Siliniyor…" : "Hesabı kalıcı olarak sil"}
            </button>
            {deleteError && <div role="alert" style={{ marginTop: 9, color: "var(--down)", fontSize: 12 }}>{deleteError}</div>}
          </Card>
        </>
      ) : (
        <Button variant="primary" onClick={() => nav("/login")}>Giriş yap / Kayıt ol</Button>
      )}
    </div>
  );
}

function planFeatures(plan: BillingPlan): string[] {
  const limits = plan.limits;
  const features = [
    `${limits.active_alerts} aktif alarm`,
    `${limits.watchlist} favori`,
    `${limits.portfolios} portföy`,
  ];
  if (limits.portfolio_health > 0) features.push(`${limits.portfolio_health} portföy sağlık analizi`);
  if (limits.exports > 0) features.push(`${limits.exports} dışa aktarma`);
  if (limits.research_archive_days > 0) features.push(`${limits.research_archive_days} günlük araştırma arşivi`);
  return features;
}

function QuotaRow({ label, used, limit, remaining }: { label: string; used: number; limit: number; remaining: number }) {
  const safeLimit = Math.max(1, limit);
  const ratio = Math.min(100, Math.max(0, (used / safeLimit) * 100));
  return <div>
    <div style={{ display: "flex", justifyContent: "space-between", gap: 12, marginBottom: 7, fontSize: 12 }}>
      <span style={{ color: "var(--text-dim)" }}>{label}</span>
      <strong>{remaining} kaldı · {used}/{limit}</strong>
    </div>
    <div className="settings-quota-track"><span style={{ width: `${ratio}%` }} /></div>
  </div>;
}

function Row({ label, value, valueColor }: { label: string; value: string; valueColor?: string }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", padding: "8px 0", fontSize: 14 }}>
      <span style={{ color: "var(--text-dim)" }}>{label}</span>
      <span style={{ fontWeight: 600, color: valueColor }}>{value}</span>
    </div>
  );
}
