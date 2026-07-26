import react from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vite";
import { fileURLToPath } from "node:url";

import { resolveBuildProfile, type BuildProfile } from "./src/app/build-profile";
import { contentSecurityPolicy } from "./src/app/csp-policy";

function buildProfilePlugin(profile: BuildProfile, entryName: string, localDevelopment: boolean): Plugin {
  return {
    name: "aviasurveil360-build-profile",
    transformIndexHtml: {
      order: "pre",
      handler(html) {
        return html
          .replace("__AVIA_ENTRY__", entryName)
          .replace("__AVIA_CSP__", contentSecurityPolicy(profile, localDevelopment));
      },
    },
    generateBundle(_options, bundle) {
      const inputs = new Set<string>();
      for (const output of Object.values(bundle)) {
        if (output.type !== "chunk") continue;
        for (const moduleId of Object.keys(output.modules)) inputs.add(moduleId);
      }
      this.emitFile({
        type: "asset",
        fileName: "build-inputs.json",
        source: `${JSON.stringify({ profile, inputs: [...inputs].sort() }, null, 2)}\n`,
      });
      const assets = Object.keys(bundle)
        .filter(
          (fileName) =>
            fileName.startsWith("assets/") &&
            /\.(?:css|js|svg|png|jpg|jpeg|webp|ttf|woff|woff2)$/.test(fileName),
        )
        .map((fileName) => `/${fileName}`)
        .sort();
      assets.push(profile === "http" ? "/http-config.json" : "/demo-build.json");
      this.emitFile({
        type: "asset",
        fileName: "app-shell-assets.json",
        source: `${JSON.stringify({ appShellVersion: 1, profile, assets }, null, 2)}\n`,
      });
    },
  };
}

export default defineConfig(({ command }) => {
  const profile = resolveBuildProfile(process.env.AVIA_BUILD_PROFILE, Boolean(process.env.VITEST));
  const httpTestProfile = profile === "http" && process.env.AVIA_HTTP_TEST_PROFILE === "canonical";
  const apiTarget = process.env.AVIA_HTTP_API_TARGET;
  const webRoot = fileURLToPath(new URL(".", import.meta.url));
  const assetsRoot = fileURLToPath(new URL("../../assets", import.meta.url));

  return {
    plugins: [
      react(),
      buildProfilePlugin(profile, httpTestProfile ? "http-test" : profile, command === "serve"),
    ],
    publicDir: `public/${profile}`,
    resolve: {
      alias: {
        "react-router-dom": fileURLToPath(
          new URL("./src/routing/client-router.tsx", import.meta.url),
        ),
      },
    },
    define: {
      __AVIA_BUILD_PROFILE__: JSON.stringify(profile),
      __AVIA_CANONICAL_TEST_TOKEN__: JSON.stringify(
        httpTestProfile ? (process.env.AVIA_CANONICAL_TEST_TOKEN ?? "") : "",
      ),
    },
    server: {
      fs: {
        allow: [webRoot, assetsRoot],
      },
      proxy:
        profile === "http" && apiTarget
          ? {
              "/api": {
                target: apiTarget,
                rewrite: (path) => path.replace(/^\/api/, ""),
              },
              "/v1": { target: apiTarget },
              "/auth": { target: apiTarget },
              "/health": { target: apiTarget },
            }
          : undefined,
    },
    build: {
      outDir: `dist/${profile}`,
      assetsInlineLimit: 0,
      emptyOutDir: true,
      manifest: true,
      sourcemap: true,
      rollupOptions: {
        input: {
          app: `${webRoot}index.html`,
          sw: `${webRoot}src/sw.ts`,
        },
        output: {
          entryFileNames: (chunk) =>
            chunk.name === "sw" ? "sw.js" : "assets/[name]-[hash].js",
        },
      },
    },
    test: {
      environment: "node",
      fileParallelism: false,
      hookTimeout: 15_000,
      testTimeout: 15_000,
      exclude: [
        "tests/e2e/**",
        "tests/offline/restart-recovery.spec.ts",
        "tests/contract/http-backend-live.test.ts",
        "node_modules/**",
        "dist/**",
      ],
      restoreMocks: true,
    },
  };
});
