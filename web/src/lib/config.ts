// Runtime configuration. The web app is served same-origin behind nginx which
// reverse-proxies /api and /ws to the API service, so no CORS/host juggling.
export const API_BASE = "/api/v1";

export function wsURL(token?: string | null): string {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const base = `${proto}://${location.host}/ws`;
  return token ? `${base}?token=${encodeURIComponent(token)}` : base;
}

// Public OAuth identifiers.
// Google "Web" client ID (also the accepted aud on the backend).
export const GOOGLE_WEB_CLIENT_ID =
  "83380535128-o4h4v9rc9u1tuis9hggcckqei0idahg3.apps.googleusercontent.com";

// Apple "Services ID" for Sign in with Apple on the web (NOT the app bundle id).
// Create one in the Apple Developer portal bound to tsc.talhacan.com, then add
// it to APP_OAUTH_APPLE_BUNDLE_IDS on the backend so the aud is accepted.
export const APPLE_SERVICE_ID = "talhacan.tscforex.web";
export const APPLE_REDIRECT_URI = "https://tsc.talhacan.com/auth/apple/callback";

export const BRAND = {
  name: "TSC Forex",
  tagline: "Canlı Altın & Döviz Piyasası",
};
