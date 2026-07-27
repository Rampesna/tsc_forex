# Orchestrator-aware graceful shutdown

The API process treats Kubernetes pod termination as a coordinated state
transition rather than an immediate process exit.

## Shutdown sequence

1. Kubernetes marks the pod endpoint as terminating and not ready.
2. The `preStop` hook waits 5 seconds for proxy and load-balancer state to
   converge.
3. Kubernetes sends `SIGTERM` to the Go process.
4. The process immediately:
   - makes `/readyz` return `503`;
   - rejects new WebSocket upgrades with `503` and `Retry-After: 5`;
   - disables HTTP keep-alives.
5. Existing traffic receives a 5-second application drain window.
6. Background producers are canceled, then HTTP requests and WebSockets drain
   concurrently for up to 75 seconds.
   - HTTP uses `http.Server.Shutdown`.
   - WebSockets receive RFC 6455 close code `1012` (`service restart`) so mobile
     clients reconnect to a healthy pod.
7. Persistence workers finish, the candle store performs its final flush, and
   only then is the database pool closed.

`terminationGracePeriodSeconds` is 110 seconds:

| Phase | Budget |
| --- | ---: |
| Kubernetes `preStop` | 5s |
| Application traffic drain | 5s |
| HTTP/WebSocket graceful shutdown | 75s |
| Persistence cleanup | 10s |
| Safety reserve before `SIGKILL` | 15s |

The Deployment uses `maxUnavailable: 0`, `maxSurge: 1`, and
`minReadySeconds: 10`. A replacement therefore becomes stably ready before an
old pod is selected for termination. The PodDisruptionBudget permits at most
one unavailable API pod during voluntary disruption.

## Probe contract

- `/livez` stays `200` while the process is draining. A correct shutdown must
  not be mistaken for a deadlock and killed early by the liveness probe.
- `/readyz` becomes `503` as soon as application drain starts.
- Readiness checks run every 2 seconds with one failed check required.

## Verification

For a controlled rollout:

```sh
kubectl -n tsc-forex rollout status deployment/tsc-forex-app --timeout=10m
kubectl -n tsc-forex get pods -l app=tsc-forex-app
kubectl -n tsc-forex logs -l app=tsc-forex-app --since=15m \
  | grep -E 'shutdown drain started|shutdown complete'
```

A healthy rollout has no `OOMKilled` or exit code `137`, keeps the Deployment
available throughout, and records `shutdown complete` for every replaced pod.
Clients may reconnect once after receiving WebSocket close code `1012`; they
must not receive an abnormal close caused by `SIGKILL`.

## Configuration invariants

- `server.shutdown_timeout` must remain greater than the longest accepted API
  operation.
- Pod termination grace must exceed `preStop + drain_delay +
  shutdown_timeout + persistence cleanup`.
- Liveness must never depend on readiness/draining state.
- Database and Redis secrets are not modified by this mechanism.
