# Temporary Tunnel Notes

This folder documents the shape of a temporary public demo. It deliberately does not include account IDs, tunnel IDs, tokens, hostnames, or secrets.

## Canonical Linux-Hosted Pattern

1. Build the UI into `web/dist`.
2. Run `gossamer-server` on the Linux host with the current Go toolchain.
3. If `cloudflared` runs on the same Linux host, bind the origin to `127.0.0.1:8095` and point the public hostname at that loopback service.
4. If an existing router-hosted tunnel must stay in place temporarily, keep the router listener loopback-only and proxy `127.0.0.1:8095` / `[::1]:8095` to the Linux origin on the trusted LAN.
5. Keep the router as a router/proxy only; do not make OpenWrt the canonical application host.
6. Disable or delete temporary tunnels after the demo.

Keep all provider-specific credentials in the operator environment, not in this repository.

## Local Origin

```bash
cd web
npm run build
cd ..
go run ./cmd/gossamer-server -addr 127.0.0.1:8095 -root . -web-dir web/dist
```

For the router-proxy variant, run the Linux origin with an explicit LAN bind and remote-listen acknowledgement:

```bash
go run ./cmd/gossamer-server -addr 0.0.0.0:8095 -allow-remote -root . -web-dir web/dist
```

## Locally Managed Cloudflare Tunnel

After `jmeyer.space` is active in the Cloudflare account and this host has been authenticated with `cloudflared tunnel login`, create and route a named tunnel:

```bash
cloudflared tunnel create gossamer
cloudflared tunnel route dns gossamer gossamer.jmeyer.space
cloudflared tunnel --config ~/.cloudflared/gossamer.yml run gossamer
```

Use `cloudflared.gossamer.example.yml` as the local config shape. Do not commit the generated tunnel UUID, credentials file, token, or account details.

## Legacy Router-Hosted Fallback

For an intentional Brume2/OpenWrt fallback test, keep the dashboard-issued tunnel token on the router only, for example under `/etc/cloudflared/token`, and either run the legacy router service as the local origin or run a loopback-only router proxy to the Linux origin:

```text
gossamer.jmeyer.space -> http://localhost:8095
```

Protect that hostname with a Cloudflare Zero Trust Access application. The demo application and the router administration UI should use separate hostnames and separate Access policies if router administration is ever exposed.

If the hostname resolves through Cloudflare but returns `525`, replace any old parking or origin DNS route for that hostname with the tunnel-managed public hostname and keep the origin service HTTP-only inside the tunnel.
