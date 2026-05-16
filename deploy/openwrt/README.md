# Brume2 OpenWrt Deployment

This deployment profile targets the GL-MT2500/Brume2 class router as a small, private Gossamer origin for one or two clients. It keeps the HTTP service local to the router/LAN and expects public access to be mediated through Cloudflare Tunnel plus Cloudflare Access.

Do not place Cloudflare tokens, tunnel IDs, account IDs, router passwords, or client emails in this repository.

## Build Package

From the repository root:

```bash
deploy/openwrt/package_brume2.sh
```

The script regenerates fixtures, builds the web UI, cross-compiles `gossamer-server` for `linux/arm64`, and writes `/tmp/gossamer-brume2.tar.gz`.

## Install On Router

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
