import { useState, type FormEvent } from "react";
import { Navigate, useNavigate } from "../router";
import { ApiError } from "../api";
import { useAuth } from "../auth";
import { Icon } from "../Icon";

export function Login() {
  const { manager, signIn } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  if (manager) return <Navigate to="/" replace />;

  async function submit(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      await signIn(email, password);
      navigate("/", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Giriş yapılamadı");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="login-page">
      <div className="login-glow one" /><div className="login-glow two" />
      <section className="login-story">
        <div className="brand large"><div className="brand-mark"><span>T</span><span>F</span></div><div><strong>TSC Forex</strong><small>Operations Console</small></div></div>
        <div className="login-copy">
          <span className="eyebrow"><i /> TEK MERKEZDEN OPERASYON</span>
          <h1>Piyasayı değil,<br /><em>sistemi</em> yönet.</h1>
          <p>Kullanıcı davranışı, abonelik geliri, canlı oturumlar ve güvenli bildirim orkestrasyonu tek bir profesyonel çalışma alanında.</p>
          <div className="login-features">
            <span><Icon name="activity" /> Canlı ürün sinyalleri</span>
            <span><Icon name="sparkles" /> Kontrollü AI hedefleme</span>
            <span><Icon name="shield" /> Uçtan uca denetim kaydı</span>
          </div>
        </div>
        <small className="login-foot">Yalnızca yetkili TSC Forex yöneticileri içindir.</small>
      </section>
      <section className="login-form-shell">
        <form onSubmit={submit} className="login-card" autoComplete="off">
          <div className="login-icon"><Icon name="shield" size={24} /></div>
          <h2>Yönetim paneline giriş</h2>
          <p>Devam etmek için yönetici bilgilerini kullan.</p>
          <label>E-posta<input value={email} onChange={(e) => setEmail(e.target.value)} type="email" name="manager-email" autoComplete="off" spellCheck={false} required /></label>
          <label>Şifre<input value={password} onChange={(e) => setPassword(e.target.value)} type="password" autoComplete="current-password" required /></label>
          {error && <div className="form-error">{error}</div>}
          <button className="primary-button" disabled={loading}>{loading ? <span className="spinner" /> : "Güvenli giriş yap"}<Icon name="arrow" size={16} /></button>
          <div className="login-assurance"><Icon name="shield" size={14} /> Oturumun kısa ömürlü erişim anahtarı ve güvenli yenileme çereziyle korunur.</div>
        </form>
      </section>
    </div>
  );
}
