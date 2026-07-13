import { APPLE_REDIRECT_URI, APPLE_SERVICE_ID } from "../lib/config";

// Sign in with Apple JS (popup). Requires a Services ID configured in the
// Apple Developer portal with tsc.talhacan.com as a web domain + return URL,
// and the Services ID added to APP_OAUTH_APPLE_BUNDLE_IDS on the backend.

interface AppleID {
  auth: {
    init: (cfg: {
      clientId: string;
      scope: string;
      redirectURI: string;
      usePopup: boolean;
    }) => void;
    signIn: () => Promise<{
      authorization: { id_token: string; code: string };
      user?: { name?: { firstName?: string; lastName?: string } };
    }>;
  };
}

declare global {
  interface Window {
    AppleID?: AppleID;
  }
}

let scriptPromise: Promise<void> | null = null;

function loadScript(): Promise<void> {
  if (scriptPromise) return scriptPromise;
  scriptPromise = new Promise((resolve, reject) => {
    if (window.AppleID) return resolve();
    const s = document.createElement("script");
    s.src =
      "https://appleid.cdn-apple.com/appleauth/static/jsapi/appleid/1/en_US/appleid.auth.js";
    s.async = true;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error("Apple betiği yüklenemedi"));
    document.head.appendChild(s);
  });
  return scriptPromise;
}

export async function appleSignIn(): Promise<{ idToken: string; fullName: string }> {
  await loadScript();
  if (!window.AppleID) throw new Error("Apple kullanılamıyor");
  window.AppleID.auth.init({
    clientId: APPLE_SERVICE_ID,
    scope: "name email",
    redirectURI: APPLE_REDIRECT_URI,
    usePopup: true,
  });
  const res = await window.AppleID.auth.signIn();
  const name = res.user?.name
    ? [res.user.name.firstName, res.user.name.lastName].filter(Boolean).join(" ")
    : "";
  return { idToken: res.authorization.id_token, fullName: name };
}
