import laravel from "laravel-vite-plugin";
import { defineConfig } from "vite";

import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [
    laravel({
      input: "resources/js/app.tsx",
      publicDirectory: "public",
      buildDirectory: "build",
      refresh: true,
    }),
    react({ include: /\.(tsx|ts|jsx|js)$/ }),
  ],
  // build: {
  //   manifest: "manifest.json",
  //   outDir: "public/build",
  //   rollupOptions: {
  //     input: "resources/js/app.tsx",
  //     output: {
  //       entryFileNames: "assets/[name].js",
  //       chunkFileNames: "assets/[name].js",
  //       assetFileNames: "assets/[name].[ext]",
  //       manualChunks: undefined,
  //     },
  //   },
  // },
  server: {
    hmr: { host: "localhost" },
  },
});
