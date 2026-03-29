import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    // Plugin to handle OAuth callback route - serve SPA for /api/v1/auth/github/callback
    {
      name: 'oauth-callback-spa',
      configureServer(server) {
        server.middlewares.use((req, _res, next) => {
          // Check if this is the OAuth callback route (browser navigation, not XHR)
          if (req.url?.startsWith('/api/v1/auth/github/callback?') &&
              !req.headers.accept?.includes('application/json')) {
            // Rewrite URL to root so SPA can handle it
            req.url = '/'
          }
          next()
        })
      }
    }
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    host: '0.0.0.0', // Allow external access
    proxy: {
      '/auth': {
        target: process.env.VITE_API_TARGET || 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/api': {
        target: process.env.VITE_API_TARGET || 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/health': {
        target: process.env.VITE_API_TARGET || 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
    },
  },
  build: {
    outDir: "dist",
    sourcemap: true,
  },
})
