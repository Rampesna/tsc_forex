import { useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { api, ApiError, setAccessToken } from "../api";
import { useAuth } from "../auth";
import { Icon } from "../Icon";
import type { SessionResponse } from "../types";

export function Settings() {
  const { user, replaceUser } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [repeat, setRepeat] = useState("");
  const [message, setMessage] = useState("");
  const mutation = useMutation({
    mutationFn: () => api<SessionResponse>("/admin/auth/password", {
      method: "POST",
      body: JSON.stringify({ current_password: current, new_password: next }),
    }),
    onSuccess: (session) => {
      setAccessToken(session.access_token);
      replaceUser(session.user); setCurrent(""); setNext(""); setRepeat("");
      setMessage("Şifre güncellendi; diğer tüm yönetici oturumları kapatıldı.");
    },
    onError: (error) => setMessage(error instanceof ApiError ? error.message : "Şifre değiştirilemedi"),
  });
  function submit(event: FormEvent) {
    event.preventDefault();
    setMessage("");
    if (next !== repeat) {
      setMessage("Yeni şifreler eşleşmiyor.");
      return;
    }
    mutation.mutate();
  }
  const strength = [next.length >= 12, /[A-Z]/.test(next), /[a-z]/.test(next), /\d/.test(next), /[^A-Za-z0-9]/.test(next)].filter(Boolean).length;
  return <div className="settings-page">
    <section className="panel settings-profile">
      <div className="settings-avatar">{(user?.display_name || user?.email || "TC").slice(0, 2).toUpperCase()}</div>
      <div><span className="kicker">YÖNETİCİ HESABI</span><h2>{user?.display_name || "Talha Can"}</h2><p>{user?.email}</p></div>
      <span className="status-pill active">Admin</span>
    </section>
    <section className="panel password-panel">
      <div className="panel-heading"><div><span className="kicker">KİMLİK DOĞRULAMA</span><h2>Şifre değiştir</h2><p>Değişiklikten sonra bu tarayıcı hariç tüm yenileme oturumları iptal edilir.</p></div><span className="list-icon"><Icon name="shield" /></span></div>
      <form onSubmit={submit}>
        <label>Mevcut şifre<input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" required /></label>
        <div className="form-split"><label>Yeni şifre<input type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" required minLength={12} /></label><label>Yeni şifre tekrar<input type="password" value={repeat} onChange={(e) => setRepeat(e.target.value)} autoComplete="new-password" required minLength={12} /></label></div>
        <div className="password-strength"><div>{[1, 2, 3, 4, 5].map((x) => <i className={strength >= x ? "on" : ""} key={x} />)}</div><span>En az 12 karakter; büyük/küçük harf, rakam ve özel karakter.</span></div>
        {message && <div className={`form-message ${message.startsWith("Şifre güncellendi") ? "success" : ""}`}>{message}</div>}
        <button className="primary-button" disabled={mutation.isPending || strength < 5 || next !== repeat}>{mutation.isPending ? <span className="spinner" /> : <Icon name="shield" size={16} />} Şifreyi güncelle</button>
      </form>
    </section>
    <section className="settings-cards">
      <article className="panel"><span className="list-icon green"><Icon name="shield" /></span><div><strong>Güvenli yenileme çerezi</strong><small>HttpOnly · Secure · SameSite Strict</small></div><span className="status-pill active">Etkin</span></article>
      <article className="panel"><span className="list-icon violet"><Icon name="audit" /></span><div><strong>Denetim günlüğü</strong><small>Ayrıcalıklı hareketler kalıcı olarak kaydedilir</small></div><span className="status-pill active">Etkin</span></article>
      <article className="panel"><span className="list-icon amber"><Icon name="sparkles" /></span><div><strong>AI güvenlik sınırı</strong><small>İzinli DSL · önizleme · açık insan onayı</small></div><span className="status-pill active">Etkin</span></article>
    </section>
  </div>;
}
