import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    exclude: ["a11y/**", "node_modules/**", "dist/**", ".astro/**"],
  },
});
