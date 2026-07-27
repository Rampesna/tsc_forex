# TSC Forex Operations Console

Private React operations console for TSC Forex. It is served by its own
Kubernetes deployment on NodePort `4775` and calls the production API at
`https://tsc-api.talhacan.com/api/v1`.

## Security model

- No public registration route exists.
- The initial administrator email comes from non-secret configuration.
- The initial password must be supplied through the
  `APP_ADMIN_BOOTSTRAP_PASSWORD` Kubernetes/GitHub Actions secret. It is never
  embedded in this application, Git, a container image, or a log.
- Access JWTs are short lived and kept only in browser memory.
- Refresh credentials are opaque, rotating, revocable, `HttpOnly`, `Secure`,
  `SameSite=Strict` cookies scoped to the admin auth endpoints.
- Every privileged read or mutation is recorded in the admin audit log.
- The first login requires a password change; changing it revokes all existing
  refresh sessions.

## AI notification workflow

The model cannot submit SQL. It can only produce an allow-listed targeting
document (audience, platforms, plans, asset sources and bounded performance
filters). The backend compiles those values into static parameterized queries.
Every campaign follows:

1. prompt interpretation;
2. deterministic validation and recipient cap;
3. recipient/sample preview;
4. explicit administrator confirmation;
5. durable, idempotent recipient queue;
6. rate-limited APNs/FCM and in-app delivery;
7. immutable audit events.

Campaign workers use row leases and idempotency keys, so another pod can safely
resume work after an interruption without duplicating the user's inbox entry.

## Local checks

```sh
npm ci
npm audit --audit-level=high
npm run lint
npm run build
```

## Production proxy

Configure the aaPanel proxy project for `tscfm.talhacan.com` to forward to:

```text
http://127.0.0.1:4775
```

Cloudflare/aaPanel must serve HTTPS. The API CORS allow-list accepts only
`https://tscfm.talhacan.com` for credentialed admin requests.
