import type { CSSProperties, ReactNode } from "react";
import { signedPct } from "../lib/format";

export function ChangePill({ pct, small }: { pct: number; small?: boolean }) {
  const up = pct >= 0;
  return (
    <span
      className="mono"
      style={{
        display: "inline-block",
        padding: small ? "1px 6px" : "3px 8px",
        borderRadius: 6,
        fontSize: small ? 11 : 12,
        fontWeight: 600,
        color: up ? "var(--up)" : "var(--down)",
        background: up ? "rgba(14,203,129,0.12)" : "rgba(246,70,93,0.12)",
        whiteSpace: "nowrap",
      }}
    >
      {signedPct(pct)}
    </span>
  );
}

const PALETTE = ["#17a2b8", "#0ecb81", "#f0a63b", "#8b5cf6", "#e05299", "#3b82f6", "#14b8a6"];

export function AssetBadge({ code, size = 34 }: { code: string; size?: number }) {
  const initials = code.replace(/[^A-Z]/g, "").slice(0, 2) || code.slice(0, 2).toUpperCase();
  let hash = 0;
  for (let i = 0; i < code.length; i++) hash = (hash * 31 + code.charCodeAt(i)) & 0xffff;
  const color = PALETTE[hash % PALETTE.length];
  return (
    <div
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        background: `${color}22`,
        color,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontSize: size * 0.36,
        fontWeight: 700,
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
    primary: { background: "var(--accent)", color: "#04222a" },
    soft: { background: "var(--accent-soft)", color: "var(--accent-hi)" },
    ghost: { background: "var(--surface-hi)", color: "var(--text)" },
    danger: { background: "rgba(246,70,93,0.14)", color: "var(--down)" },
  };
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      style={{
        ...styles[variant],
        border: "none",
        borderRadius: 12,
        padding: "12px 16px",
        fontSize: 15,
        fontWeight: 700,
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.55 : 1,
        width: full ? "100%" : undefined,
        transition: "filter 0.15s",
      }}
      onMouseDown={(e) => (e.currentTarget.style.filter = "brightness(0.92)")}
      onMouseUp={(e) => (e.currentTarget.style.filter = "none")}
      onMouseLeave={(e) => (e.currentTarget.style.filter = "none")}
    >
      {children}
    </button>
  );
}

export function EmptyState({ icon, title, hint }: { icon: string; title: string; hint?: string }) {
  return (
    <div style={{ textAlign: "center", padding: "60px 20px", color: "var(--text-dim)" }}>
      <div style={{ fontSize: 40, marginBottom: 12 }}>{icon}</div>
      <div style={{ fontSize: 16, fontWeight: 600, color: "var(--text)" }}>{title}</div>
      {hint && <div style={{ fontSize: 13, marginTop: 6 }}>{hint}</div>}
    </div>
  );
}
