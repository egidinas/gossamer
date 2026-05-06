# Temporary Tunnel Notes

This folder documents the shape of a temporary public demo. It deliberately does not include account IDs, tunnel IDs, tokens, hostnames, or secrets.

## Pattern

1. Build the UI into `web/dist`.
2. Run the API on `127.0.0.1:8095`.
3. Serve the static UI and proxy `/api/*` through a local reverse proxy.
4. Point a temporary HTTPS tunnel at that reverse proxy.
5. Disable or delete the tunnel after the demo.

Keep all provider-specific credentials in the operator environment, not in this repository.
