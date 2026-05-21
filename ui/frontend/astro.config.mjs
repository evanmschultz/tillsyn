import { defineConfig } from 'astro/config';
import solidJs from '@astrojs/solid-js';

// https://astro.build/config
export default defineConfig({
  integrations: [solidJs()],
  output: 'static',
  server: {
    // Pinned to match ui/wails.json frontend:dev:serverUrl. Wails proxies its
    // webview to this port during `mage uiDev` — drift between the two configs
    // surfaces as "Timeout waiting for frontend DevServer" + blank-white-screen
    // because the Wails proxy can't reach the Astro server.
    port: 51428,
  },
});
