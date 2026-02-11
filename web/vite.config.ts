import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const proxyTarget = process.env.VITE_PROXY_TARGET || 'http://localhost:8080'
const wsTarget = proxyTarget.replace(/^http/, 'ws')

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist'
  },
  server: {
    port: 5173,
    proxy: {
      '/ws': {
        target: wsTarget,
        ws: true
      },
      '/health': {
        target: proxyTarget
      }
    }
  }
})
