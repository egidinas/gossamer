# Temporary Tunnel Notes

This folder documents the shape of a temporary public demo. It deliberately does not include account IDs, tunnel IDs, tokens, hostnames, or secrets.

## Pattern

1. Build the UI into `web/dist`.
2. Run the demo server on `127.0.0.1:8095` with `-web-dir web/dist`.
3. Point a temporary HTTPS tunnel at that loopback service.
4. Disable or delete the tunnel after the demo.

Keep all provider-specific credentials in the operator environment, not in this repository.

## Local Origin

```bash
cd web
npm run build
cd ..
go run ./cmd/gossamer-server -addr 127.0.0.1:8095 -root . -web-dir web/dist
```

## Locally Managed Cloudflare Tunnel

After `jmeyer.space` is active in the Cloudflare account and this host has been authenticated with `cloudflared tunnel login`, create and route a named tunnel:

```bash
cloudflared tunnel create gossamer
cloudflared tunnel route dns gossamer gossamer.jmeyer.space
cloudflared tunnel --config ~/.cloudflared/gossamer.yml run gossamer
```

Use `cloudflared.gossamer.example.yml` as the local config shape. Do not commit the generated tunnel UUID, credentials file, token, or account details.

## Router-Hosted Tunnel

For the Brume2/OpenWrt deployment, keep the dashboard-issued tunnel token on the router only, for example under `/etc/cloudflared/token`, and run the router service as the local origin:

```text
gossamer.jmeyer.space -> http://localhost:8095
```

Protect that hostname with a Cloudflare Zero Trust Access application. The demo application and the router administration UI should use separate hostnames and separate Access policies if router administration is ever exposed.

If the hostname resolves through Cloudflare but returns `525`, replace any old parking or origin DNS route for that hostname with the tunnel-managed public hostname and keep the origin service HTTP-only inside the tunnel.
