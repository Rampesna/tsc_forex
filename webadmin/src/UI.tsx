import type { ReactNode } from "react";
import { Icon } from "./Icon";

export function LoadingBlock({ height = 220 }: { height?: number }) {
  return <div className="loading-block" style={{ height }}><span className="spinner" /></div>;
}

export function EmptyState({ icon = "activity", title, text }: { icon?: "activity" | "users" | "campaign" | "online"; title: string; text: string }) {
  return <div className="empty-state"><span><Icon name={icon} size={22} /></span><strong>{title}</strong><p>{text}</p></div>;
}

export function StatusPill({ value }: { value: string }) {
  const normalized = value.toLowerCase();
  const label: Record<string, string> = {
    active: "Aktif", free: "Ücretsiz", queued: "Kuyrukta", running: "Gönderiliyor",
    completed: "Tamamlandı", processed: "İşlendi", failed: "Başarısız",
    cancelled: "İptal", draft: "Taslak", ios: "iOS", android: "Android", both: "Çoklu",
    plus: "Plus", pro: "Pro", expert: "Expert", elite: "Elite", web: "Web",
  };
  return <span className={`status-pill ${normalized}`}>{label[normalized] || value}</span>;
}

export function Modal({ children, onClose }: { children: ReactNode; onClose: () => void }) {
  return <div className="modal-backdrop" role="presentation" onMouseDown={onClose}><div className="modal-card" role="dialog" aria-modal="true" onMouseDown={(e) => e.stopPropagation()}>{children}</div></div>;
}
