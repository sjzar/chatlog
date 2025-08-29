import { fileURLToPath } from 'url'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: [
      {
        find: '@',
        replacement: _resolve('src'),
      }
    ],
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:5030',
        changeOrigin: true,
        secure: false,
      },
    },
  },
})

function _resolve(dir: string): string {
  // return path.resolve(__dirname, dir)
  return fileURLToPath(new URL(dir, import.meta.url));
}
