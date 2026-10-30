import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The browser talks to this dev server only. Requests to /api are forwarded to the Go
// backend, so the login cookie stays same-origin and no CORS setup is needed.
const target = process.env.VITE_API_TARGET || 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/testSetup.js',
  },
  server: {
    port: 5173,
    proxy: { '/api': { target, changeOrigin: false } },
  },
})
