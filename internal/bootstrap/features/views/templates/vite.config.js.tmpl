import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    // Output to ../static/dist/ so Go's embed can pick it up.
    // The dist/ subdirectory is git-ignored while static/embed.go is tracked.
    outDir: '../static/dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      // Proxy API and Swagger requests to the Go backend in dev mode.
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/auth': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/swagger': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
});
