import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../auth/AuthContext";
import { googleSignIn } from "../auth/google";
import { passkeySupported } from "../auth/passkey";
import { BRAND } from "../lib/config";
import { Icon, type IconName } from "../components/Icon";

export function Login() {
  const auth = useAuth();
  const nav = useNavigate();
  const [isRegister, setRegister] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const googleRef = useRef<HTMLDivElement>(null);

  // Once signed in, leave the login screen and return to the app.
  useEffect(() => {
    if (auth.authed) nav("/", { replace: true });
  }, [auth.authed, nav]);

  useEffect(() => {
    if (!googleRef.current) return;
    googleSignIn(googleRef.current)
      .then((idToken) => auth.loginWithGoogleToken(idToken).catch(() => {}))
      .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isRegister) auth.register(email, password).catch(() => {});
    else auth.login(email, password).catch(() => {});
  };

  const disabled = auth.busy || !email || password.length < 8;

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: 20,
        background:
          "radial-gradient(1000px 500px at 50% -10%, rgba(0,61,77,0.55), transparent 60%), var(--bg)",
      }}
    >
      <div style={{ width: "100%", maxWidth: 400 }}>
        <button
          onClick={() => nav("/")}
          style={{
            background: "none",
            border: "none",
            color: "var(--text-dim)",
            fontSize: 13,
            fontWeight: 600,
            cursor: "pointer",
            marginBottom: 8,
            padding: 4,
          }}
        >
          ← Piyasalara dön
        </button>
        <div style={{ textAlign: "center", marginBottom: 28 }}>
          <img
            src="/logo.png"
            width={84}
            height={84}
            style={{ borderRadius: 20, boxShadow: "0 12px 30px rgba(0,61,77,0.6)" }}
            alt=""
          />
          <h1 style={{ fontSize: 28, fontWeight: 800, margin: "16px 0 4px" }}>{BRAND.name}</h1>
          <div style={{ color: "var(--text-dim)", fontSize: 14 }}>{BRAND.tagline}</div>
        </div>

        <form onSubmit={submit} style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          <Field icon="mail">
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="E-posta"
              autoComplete="email username webauthn"
              style={inputStyle}
            />
          </Field>
          <Field icon="lock">
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Parola (min 8)"
              autoComplete={isRegister ? "new-password" : "current-password"}
              style={inputStyle}
            />
          </Field>

          {auth.error && (
            <div style={{ color: "var(--down)", fontSize: 13, textAlign: "center" }}>{auth.error}</div>
          )}

          <button type="submit" disabled={disabled} style={primaryBtn(disabled)}>
            {auth.busy ? <span className="spinner" /> : isRegister ? "Kayıt Ol" : "Giriş Yap"}
          </button>
        </form>

        <div style={dividerStyle}>
          <span>veya</span>
        </div>

        <div style={{ display: "flex", justifyContent: "center", marginBottom: 12 }} ref={googleRef} />

        <button onClick={() => auth.loginWithApple().catch(() => {})} style={appleBtn}>
           Apple ile devam et
        </button>

        {passkeySupported() && (
          <button onClick={() => auth.loginWithPasskey().catch(() => {})} style={passkeyBtn}>
            <Icon name="key" size={16} /> Geçiş anahtarı ile giriş
          </button>
        )}

        <button
          onClick={() => {
            setRegister((v) => !v);
            auth.clearError();
          }}
          style={{
            marginTop: 16,
            width: "100%",
            background: "none",
            border: "none",
            color: "var(--accent-hi)",
            fontSize: 13,
            fontWeight: 600,
            cursor: "pointer",
          }}
        >
          {isRegister ? "Zaten hesabım var • Giriş yap" : "Hesabın yok mu? • Kayıt ol"}
        </button>
      </div>
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  flex: 1,
  background: "transparent",
  border: "none",
  outline: "none",
  color: "var(--text)",
  fontSize: 15,
};

function Field({ icon, children }: { icon: IconName; children: React.ReactNode }) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        background: "var(--surface)",
        border: "1px solid var(--stroke)",
        borderRadius: 12,
        padding: "13px 14px",
      }}
    >
      <span style={{ color: "var(--text-faint)", display: "flex" }}>
        <Icon name={icon} size={17} />
      </span>
      {children}
    </div>
  );
}

function primaryBtn(disabled: boolean): React.CSSProperties {
  return {
    background: "var(--accent)",
    color: "#04222a",
    border: "none",
    borderRadius: 12,
    padding: 14,
    fontSize: 16,
    fontWeight: 800,
    cursor: disabled ? "not-allowed" : "pointer",
    opacity: disabled ? 0.55 : 1,
    display: "flex",
    justifyContent: "center",
  };
}

const dividerStyle: React.CSSProperties = {
  textAlign: "center",
  color: "var(--text-faint)",
  fontSize: 12,
  margin: "18px 0",
  position: "relative",
};

const appleBtn: React.CSSProperties = {
  width: "100%",
  background: "#000",
  color: "#fff",
  border: "1px solid rgba(255,255,255,0.18)",
  borderRadius: 12,
  padding: 13,
  fontSize: 15,
  fontWeight: 600,
  cursor: "pointer",
  marginBottom: 10,
};

const passkeyBtn: React.CSSProperties = {
  width: "100%",
  background: "var(--surface)",
  color: "var(--text)",
  border: "1px solid var(--stroke)",
  borderRadius: 12,
  padding: 13,
  fontSize: 15,
  fontWeight: 600,
  cursor: "pointer",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  gap: 8,
};
