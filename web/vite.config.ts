import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const apiProxyTarget = process.env.GOSSAMER_API_PROXY_TARGET || "http://127.0.0.1:8095";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": apiProxyTarget,
      "/healthz": apiProxyTarget,
      "/data": apiProxyTarget
    }
  }
});
