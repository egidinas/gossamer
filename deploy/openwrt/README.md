# Brume2 OpenWrt Fallback Deployment

This deployment profile targets the GL-MT2500/Brume2 class router as a legacy fallback origin for one or two clients. The preferred public deployment is Linux-hosted `gossamer-server` plus `cloudflared`, with the router only routing or proxying traffic. Use this OpenWrt package only when intentionally testing router-hosted compatibility.

Do not place Cloudflare tokens, tunnel IDs, account IDs, router passwords, or client emails in this repository.

## Build Package

From the repository root:

```bash
deploy/openwrt/package_brume2.sh
```

The script regenerates fixtures, builds the web UI, cross-compiles `gossamer-server` for `linux/arm64`, and writes `/tmp/gossamer-brume2.tar.gz`. It defaults the fallback router artifact to the known-good OpenWrt runtime toolchain (`GOTOOLCHAIN=go1.22.12`) while the canonical Linux deployment uses the normal current Go toolchain. Set `GOSSAMER_BRUME2_GOTOOLCHAIN` only for an intentional router-runtime compatibility test. The OpenWrt binary is built with `-tags noduckdb` because the public demo is served from static tile bundles and does not need the CGO-only Parquet preview endpoint.

## Install On Router

The preferred current router role is a loopback-only proxy for an origin on the Linux host. In that mode, keep the Linux host on the current Go/Node toolchains and run:

```text
gossamer-server -addr 0.0.0.0:8095 -allow-remote -root /home/svc_pmg_testbed_b/gossamer -web-dir /home/svc_pmg_testbed_b/gossamer/web/dist
```

Then configure the router service to listen only on `127.0.0.1:8095` and `[::1]:8095`, forwarding to the Linux host's trusted LAN address. The Cloudflare public hostname can continue pointing at `http://localhost:8095` on the router while the application itself is served by Linux.

Use the package below only for the legacy router-origin fallback.

```bash
scp /tmp/gossamer-brume2.tar.gz root@192.168.8.1:/tmp/
ssh root@192.168.8.1 'mkdir -p /opt/gossamer && tar -C /opt/gossamer -xzf /tmp/gossamer-brume2.tar.gz && cp /opt/gossamer/gossamer.init /etc/init.d/gossamer && chmod +x /opt/gossamer/gossamer-server /etc/init.d/gossamer && /etc/init.d/gossamer enable && /etc/init.d/gossamer restart'
```

The init script starts:

```text
/opt/gossamer/gossamer-server -addr 0.0.0.0:8095 -allow-remote -root /opt/gossamer -web-dir /opt/gossamer/web/dist
```

This leaves the router firewall untouched. On the current Brume2/OpenWrt profile, verify the origin through `localhost` or IPv6 loopback for the tunnel path. Direct IPv4 LAN access can still be blocked by the router firewall policy.

## Verify

```bash
ssh root@192.168.8.1 '/etc/init.d/gossamer status; curl -g -fsSI http://[::1]:8095/'
curl -I https://gossamer.jmeyer.space/
```

If `gossamer.jmeyer.space` returns Cloudflare `525`, DNS is reaching Cloudflare but the hostname is not yet routed to the tunnel origin with a compatible SSL/TLS setting. Add or update the tunnel public hostname so `gossamer.jmeyer.space` points to `http://localhost:8095` or `http://[::1]:8095`.

## Cloudflare Access

For a dashboard-managed Cloudflare Tunnel token on the router, keep the token only under `/etc/cloudflared/token` and point a public hostname at the local origin:

```text
gossamer.jmeyer.space -> http://localhost:8095
```

Then create a Cloudflare Zero Trust Access application:

- Type: Self-hosted
- Public hostname: `gossamer.jmeyer.space`
- Session duration: short demo-appropriate duration
- Policy: allow only the owner identity and explicitly invited client identities

Do not expose the router administration UI through this Gossamer Access application. If router administration ever needs Cloudflare Access, create a separate hostname, separate policy, and separate review step.
