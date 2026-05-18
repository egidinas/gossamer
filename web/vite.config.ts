import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath, URL } from "node:url";

const apiProxyTarget = process.env.GOSSAMER_API_PROXY_TARGET || "http://127.0.0.1:8095";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "signalforge-web": fileURLToPath(new URL("./vendor/signalforge-web/dist/signalforge-web.es.js", import.meta.url)),
    }
  },
  server: {
    proxy: {
      "/api": apiProxyTarget,
      "/healthz": apiProxyTarget,
      "/data": apiProxyTarget
    }
  }
});
