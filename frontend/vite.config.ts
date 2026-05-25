import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // Use '/api/' so '/apis/...' (source folder) is not mistaken for API routes.
      // http://localhost:5173/api/... is the subject of the proxy.
      '/api/': 'http://backend:8080',
    },
  }
})
