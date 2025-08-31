import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const API_BASE = process.env.VITE_API_BASE || ''

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': { target: 'http://localhost:8080', changeOrigin: true },
      '/health': { target: 'http://localhost:8080', changeOrigin: true }
    }
  },
  define: {
    __API_BASE__: JSON.stringify(API_BASE)
  }
})
