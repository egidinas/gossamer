# Public Demo Access

Gossamer is safest as a local demonstrator. Temporary public access is possible, but the deployment should stay narrow and revocable.

## Recommended Shape

- Run the Go API on loopback, for example `127.0.0.1:8095`.
- Build the web UI with `npm run build`.
- Serve `web/dist` as static files.
- Reverse-proxy only the static UI and `/api/*` routes.
- Use HTTPS at the public domain.
- Do not expose shell access, repository directories, fixture-generation commands, or logs.

## Example Local Commands

```bash
go run ./cmd/gossamer-server -addr 127.0.0.1:8095
cd web
npm run build
```

## Domain Notes

Either `egidinas.de` or `jmeyer.space` can host a temporary demo subdomain. Suggested names:

- `gossamer.egidinas.de`
- `gossamer.jmeyer.space`

Keep DNS and tunnel credentials outside this repository.

## Operational Guardrails

- Treat the deployment as temporary and easy to revoke.
- Rebuild from a clean git checkout.
- Run the forbidden-term scan before publishing.
- Keep API state mocked or read-only for public access.
- Add basic access control if the demo should only be visible during a scheduled review.
