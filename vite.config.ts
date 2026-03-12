import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    host: true,
    proxy: {
      // Python data fetcher (OSINT live data)
      '/osint': {
        target: 'http://127.0.0.1:8000',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/osint/, '/api'),
      },
      // Go SENTINEL backend (events, alerts, health, etc.)
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/metrics': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
})
