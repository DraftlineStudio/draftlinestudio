import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
  server: {
    // The Go tree is outside this project root, and one file in it is read
    // from a test: internal/types/testdata/copyright_cases.json is the shared
    // table of worked copyright pages that the Go generator and the
    // TypeScript preview are both checked against. Allowing exactly that
    // directory keeps one copy of the table instead of two that can drift.
    fs: { allow: ['.', '../internal/types/testdata'] },
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
