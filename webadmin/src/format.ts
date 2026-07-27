export const trDate = (value?: string) =>
  value
    ? new Intl.DateTimeFormat("tr-TR", {
        day: "2-digit",
        month: "short",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }).format(new Date(value))
    : "—";

export const shortDate = (value: string) =>
  new Intl.DateTimeFormat("tr-TR", { day: "2-digit", month: "short" }).format(new Date(value));

export const tryMoney = (value: number) =>
  new Intl.NumberFormat("tr-TR", {
    style: "currency",
    currency: "TRY",
    maximumFractionDigits: 0,
  }).format(value);

export const compact = (value: number) =>
  new Intl.NumberFormat("tr-TR", { notation: "compact", maximumFractionDigits: 1 }).format(value);

export const planName = (code: string) =>
  ({ free: "Ücretsiz", plus: "Başlangıç", pro: "Pro", expert: "Uzman", elite: "Elite" })[code] || code;

export const platformName = (value: string) =>
  ({ ios: "iOS", android: "Android", both: "iOS + Android", apple: "App Store", google: "Google Play", unknown: "Bilinmiyor" })[value] || value;
