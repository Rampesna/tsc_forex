import { GOOGLE_WEB_CLIENT_ID } from "../lib/config";

// Loads Google Identity Services and resolves a Google ID token via the
// One Tap / button flow. The token's aud is the web client id, which the
// backend accepts (APP_OAUTH_GOOGLE_CLIENT_IDS).
//
// Requires https://tsc.talhacan.com to be an "Authorized JavaScript origin"
// on the web OAuth client in the Google Cloud console.

interface GoogleAccounts {
  accounts: {
    id: {
      initialize: (cfg: {
        client_id: string;
        callback: (resp: { credential: string }) => void;
        auto_select?: boolean;
        use_fedcm_for_prompt?: boolean;
      }) => void;
      prompt: () => void;
      renderButton: (el: HTMLElement, opts: Record<string, unknown>) => void;
    };
  };
}

declare global {
  interface Window {
    google?: GoogleAccounts;
  }
}

let scriptPromise: Promise<void> | null = null;

function loadScript(): Promise<void> {
  if (scriptPromise) return scriptPromise;
  scriptPromise = new Promise((resolve, reject) => {
    if (window.google?.accounts?.id) return resolve();
    const s = document.createElement("script");
    s.src = "https://accounts.google.com/gsi/client";
    s.async = true;
    s.defer = true;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error("Google betiği yüklenemedi"));
    document.head.appendChild(s);
  });
  return scriptPromise;
}

// Renders the official Google button into the given container and resolves
// with an ID token when the user completes sign-in.
export async function googleSignIn(container: HTMLElement): Promise<string> {
  await loadScript();
  return new Promise((resolve, reject) => {
    if (!window.google?.accounts?.id) return reject(new Error("Google kullanılamıyor"));
    window.google.accounts.id.initialize({
      client_id: GOOGLE_WEB_CLIENT_ID,
      callback: (resp) => (resp?.credential ? resolve(resp.credential) : reject(new Error("Google iptal"))),
      use_fedcm_for_prompt: true,
    });
    window.google.accounts.id.renderButton(container, {
      theme: "filled_black",
      size: "large",
      shape: "pill",
      text: "continue_with",
      width: 320,
      locale: "tr",
    });
  });
}
