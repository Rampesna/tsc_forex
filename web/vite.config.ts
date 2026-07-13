import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Dev proxy forwards /api and /ws to the production backend so `npm run dev`
// works without CORS friction. Production builds hit the same-origin reverse
// proxy (nginx) which routes /api + /ws to the API service.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": { target: "https://tsc-api.talhacan.com", changeOrigin: true, secure: true },
      "/ws": { target: "https://tsc-api.talhacan.com", changeOrigin: true, secure: true, ws: true },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: false,
    chunkSizeWarningLimit: 1200,
  },
});
