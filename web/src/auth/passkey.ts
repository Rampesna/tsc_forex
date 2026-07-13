import { request } from "../lib/api";
import type { TokenResponse } from "../lib/types";

// --- base64url helpers ------------------------------------------------------
function b64urlToBuf(s: string): ArrayBuffer {
  const pad = "=".repeat((4 - (s.length % 4)) % 4);
  const b64 = (s + pad).replace(/-/g, "+").replace(/_/g, "/");
  const bin = atob(b64);
  const buf = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i);
  return buf.buffer;
}

function bufToB64url(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function passkeySupported(): boolean {
  return typeof window !== "undefined" && !!window.PublicKeyCredential;
}

// Usernameless (discoverable) passkey login.
export async function passkeyLogin(): Promise<TokenResponse> {
  const begin = await request<{ options: { publicKey: PublicKeyCredentialRequestOptionsJSON }; session: string }>(
    "/auth/passkey/login/begin",
    { method: "POST", body: {} },
  );
  const pk = begin.options.publicKey;
  const publicKey: PublicKeyCredentialRequestOptions = {
    challenge: b64urlToBuf(pk.challenge),
    rpId: pk.rpId,
    timeout: pk.timeout,
    userVerification: (pk.userVerification as UserVerificationRequirement) ?? "preferred",
    allowCredentials: (pk.allowCredentials ?? []).map((c) => ({
      id: b64urlToBuf(c.id),
      type: "public-key",
      transports: c.transports as AuthenticatorTransport[] | undefined,
    })),
  };
  const cred = (await navigator.credentials.get({ publicKey })) as PublicKeyCredential;
  const resp = cred.response as AuthenticatorAssertionResponse;
  const credential = {
    id: cred.id,
    rawId: bufToB64url(cred.rawId),
    type: "public-key",
    response: {
      authenticatorData: bufToB64url(resp.authenticatorData),
      clientDataJSON: bufToB64url(resp.clientDataJSON),
      signature: bufToB64url(resp.signature),
      userHandle: resp.userHandle ? bufToB64url(resp.userHandle) : "",
    },
  };
  return request<TokenResponse>("/auth/passkey/login/finish", {
    method: "POST",
    body: { session: begin.session, credential },
  });
}

// Register a passkey for the signed-in user.
export async function passkeyRegister(): Promise<void> {
  const begin = await request<{ options: { publicKey: PublicKeyCredentialCreationOptionsJSON }; session: string }>(
    "/me/passkey/register/begin",
    { method: "POST", auth: true, body: {} },
  );
  const pk = begin.options.publicKey;
  const publicKey: PublicKeyCredentialCreationOptions = {
    challenge: b64urlToBuf(pk.challenge),
    rp: pk.rp,
    user: {
      id: b64urlToBuf(pk.user.id),
      name: pk.user.name,
      displayName: pk.user.displayName,
    },
    pubKeyCredParams: pk.pubKeyCredParams as PublicKeyCredentialParameters[],
    timeout: pk.timeout,
    attestation: (pk.attestation as AttestationConveyancePreference) ?? "none",
    authenticatorSelection: pk.authenticatorSelection as AuthenticatorSelectionCriteria | undefined,
    excludeCredentials: (pk.excludeCredentials ?? []).map((c) => ({
      id: b64urlToBuf(c.id),
      type: "public-key",
    })),
  };
  const cred = (await navigator.credentials.create({ publicKey })) as PublicKeyCredential;
  const resp = cred.response as AuthenticatorAttestationResponse;
  const credential = {
    id: cred.id,
    rawId: bufToB64url(cred.rawId),
    type: "public-key",
    response: {
      attestationObject: bufToB64url(resp.attestationObject),
      clientDataJSON: bufToB64url(resp.clientDataJSON),
    },
  };
  await request("/me/passkey/register/finish", {
    method: "POST",
    auth: true,
    body: { session: begin.session, credential },
  });
}

// Minimal JSON option shapes returned by the backend (base64url strings).
interface PublicKeyCredentialRequestOptionsJSON {
  challenge: string;
  rpId: string;
  timeout?: number;
  userVerification?: string;
  allowCredentials?: { id: string; type: string; transports?: string[] }[];
}
interface PublicKeyCredentialCreationOptionsJSON {
  challenge: string;
  rp: { id: string; name: string };
  user: { id: string; name: string; displayName: string };
  pubKeyCredParams: { type: string; alg: number }[];
  timeout?: number;
  attestation?: string;
  authenticatorSelection?: unknown;
  excludeCredentials?: { id: string; type: string }[];
}
