export const AI_CONSENT_VERSION = "2026-07-17.v1";
const AI_CONSENT_KEY = "tsc_ai_openai_consent";

// This browser-scoped identifier is only a stable abuse-correlation value. It
// is not a hardware identifier, App Attest proof, or a physical-device claim.
// The one-time free research trial is unlocked only by the attested iOS app.
export const WEB_ABUSE_ID_KEY = "tsc_web_device_id_v1";

interface StoredAIConsent {
  version: string;
  granted_at: string;
}

export function hasCurrentAIConsent(): boolean {
  try {
    const value = JSON.parse(localStorage.getItem(AI_CONSENT_KEY) || "null") as StoredAIConsent | null;
    return value?.version === AI_CONSENT_VERSION && Boolean(value.granted_at);
  } catch {
    return false;
  }
}

export function grantAIConsent(): void {
  const value: StoredAIConsent = {
    version: AI_CONSENT_VERSION,
    granted_at: new Date().toISOString(),
  };
  localStorage.setItem(AI_CONSENT_KEY, JSON.stringify(value));
  window.dispatchEvent(new Event("tsc-ai-consent-change"));
}

export function withdrawAIConsent(): void {
  localStorage.removeItem(AI_CONSENT_KEY);
  window.dispatchEvent(new Event("tsc-ai-consent-change"));
}

// Called only after the server confirms account deletion. All browser-held
// account data, consent, tokens, portfolio caches and offline outbox entries
// are removed. The de-linked abuse identifier is deliberately retained so a
// deleted/recreated account cannot reset abuse controls.
export function clearPersonalBrowserDataAfterDeletion(): void {
  const abuseID = localStorage.getItem(WEB_ABUSE_ID_KEY);
  localStorage.clear();
  sessionStorage.clear();
  if (abuseID) localStorage.setItem(WEB_ABUSE_ID_KEY, abuseID);
}
