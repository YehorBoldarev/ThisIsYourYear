/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The Go API listens on :8080. In development the Vite dev server proxies the
// API and the OpenAPI document to it, so the browser sees a single origin and
// the HttpOnly refresh cookie flows without CORS. (Used from Step B2 onward;
// declared here so the wiring is in place.)
const API_TARGET = process.env.VITE_API_TARGET ?? 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': { target: API_TARGET, changeOrigin: true },
      '/openapi.yaml': { target: API_TARGET, changeOrigin: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: true,
  },
})
