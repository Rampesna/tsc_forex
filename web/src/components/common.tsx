import type { CSSProperties, ReactNode } from "react";
import { Icon, type IconName } from "./Icon";

export function ChangePill({ pct, small }: { pct: number; small?: boolean }) {
  const up = pct >= 0;
  // Linear-style: quiet colored text with a small caret — no chunky pill.
  return (
    <span
      className="mono"
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 3,
        fontSize: small ? 12 : 12.5,
        fontWeight: 550,
        color: up ? "var(--up)" : "var(--down)",
        whiteSpace: "nowrap",
      }}
    >
      <span style={{ fontSize: small ? 8 : 9, lineHeight: 1 }}>{up ? "▲" : "▼"}</span>
      {Math.abs(pct).toFixed(2)}%
    </span>
  );
}

export function AssetBadge({ code, size = 34 }: { code: string; size?: number }) {
  // Monochrome, restrained — the ticker on a neutral surface. No decorative color.
  const initials = code.replace(/[^A-Z]/g, "").slice(0, 2) || code.slice(0, 2).toUpperCase();
  return (
    <div
      style={{
        width: size,
        height: size,
        borderRadius: size / 3,
        background: "var(--surface-hi)",
        border: "1px solid var(--stroke)",
        color: "var(--text-dim)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontSize: size * 0.32,
        fontWeight: 600,
        letterSpacing: "-0.02em",
        flexShrink: 0,
      }}
    >
      {initials}
    </div>
  );
}

export function ConnectionDot({ connected }: { connected: boolean }) {
  return (
    <span
      title={connected ? "Canlı" : "Bağlanıyor…"}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 6,
        fontSize: 12,
        color: connected ? "var(--up)" : "var(--text-faint)",
      }}
    >
      <span
        style={{
          width: 8,
          height: 8,
          borderRadius: 4,
          background: connected ? "var(--up)" : "var(--text-faint)",
          boxShadow: connected ? "0 0 0 3px rgba(14,203,129,0.15)" : "none",
        }}
      />
      {connected ? "Canlı" : "Bağlanıyor"}
    </span>
  );
}

export function Card({
  children,
  style,
  onClick,
}: {
  children: ReactNode;
  style?: CSSProperties;
  onClick?: () => void;
}) {
  return (
    <div
      onClick={onClick}
      style={{
        background: "var(--surface)",
        border: "1px solid var(--stroke)",
        borderRadius: "var(--radius)",
        ...style,
      }}
    >
      {children}
    </div>
  );
}

export function Button({
  children,
  onClick,
  disabled,
  variant = "primary",
  full,
  type = "button",
}: {
  children: ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  variant?: "primary" | "ghost" | "soft" | "danger";
  full?: boolean;
  type?: "button" | "submit";
}) {
  const styles: Record<string, CSSProperties> = {
    primary: { background: "var(--accent)", color: "#04222a", border: "1px solid transparent" },
    soft: { background: "var(--accent-soft)", color: "var(--accent-hi)", border: "1px solid transparent" },
    ghost: { background: "var(--surface-hi)", color: "var(--text)", border: "1px solid var(--stroke)" },
    danger: { background: "transparent", color: "var(--down)", border: "1px solid var(--stroke-strong)" },
  };
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      style={{
        ...styles[variant],
        borderRadius: 9,
        padding: "10px 15px",
        fontSize: 13.5,
        fontWeight: 600,
        letterSpacing: "-0.01em",
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.5 : 1,
        width: full ? "100%" : undefined,
        transition: "filter 0.15s, background 0.15s",
      }}
      onMouseDown={(e) => (e.currentTarget.style.filter = "brightness(0.93)")}
      onMouseUp={(e) => (e.currentTarget.style.filter = "none")}
      onMouseLeave={(e) => (e.currentTarget.style.filter = "none")}
    >
      {children}
    </button>
  );
}

export function EmptyState({ icon, title, hint }: { icon: IconName; title: string; hint?: string }) {
  return (
    <div style={{ textAlign: "center", padding: "60px 20px", color: "var(--text-dim)" }}>
      <div
        style={{
          width: 48,
          height: 48,
          borderRadius: 12,
          margin: "0 auto 14px",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          background: "var(--surface-hi)",
          border: "1px solid var(--stroke)",
          color: "var(--text-faint)",
        }}
      >
        <Icon name={icon} size={22} />
      </div>
      <div style={{ fontSize: 15, fontWeight: 600, color: "var(--text)" }}>{title}</div>
      {hint && <div style={{ fontSize: 13, marginTop: 6 }}>{hint}</div>}
    </div>
  );
}
