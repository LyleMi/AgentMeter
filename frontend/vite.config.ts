import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:34115'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 550,
    rollupOptions: {
      output: {
        manualChunks(id) {
          const normalizedId = id.replace(/\\/g, '/')
          if (!normalizedId.includes('/node_modules/')) return
          if (normalizedId.includes('/node_modules/echarts/')) return 'vendor-echarts'
          if (normalizedId.includes('/node_modules/@ant-design/icons-vue/')) return 'vendor-icons'
          if (normalizedId.match(/\/node_modules\/(@vue|vue|vue-router)\//)) return 'vendor-vue'
        }
      }
    }
  }
})
