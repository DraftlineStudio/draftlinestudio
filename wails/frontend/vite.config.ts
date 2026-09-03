import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [
    react(),
    // onnxruntime-web references its .wasm via import.meta.url, which makes
    // Vite copy 21 MB into dist — dead weight in the embedded binary, since
    // Read Aloud always loads the runtime from the downloaded bundle via
    // env.backends.onnx.wasm.wasmPaths. Drop it from the build output.
    {
      name: 'drop-bundled-ort-wasm',
      generateBundle(_options, bundle) {
        for (const key of Object.keys(bundle)) {
          if (key.includes('ort-wasm') && key.endsWith('.wasm')) delete bundle[key]
        }
      },
    },
  ],
  // kokoro-js/transformers.js resolve their onnxruntime .wasm files relative
  // to import.meta.url inside the Read Aloud worker; esbuild pre-bundling
  // breaks that resolution, so they load as-is.
  optimizeDeps: {
    exclude: ['kokoro-js', '@huggingface/transformers'],
  },
  // The Read Aloud worker lazy-imports kokoro-js, which requires
  // code-splitting; only the ES worker format supports that.
  worker: {
    format: 'es',
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
  server: {
    port: 5173,
    strictPort: true,
    hmr: {
      host: 'localhost',
      protocol: 'ws',
      port: 5173,
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: false,
  },
})
