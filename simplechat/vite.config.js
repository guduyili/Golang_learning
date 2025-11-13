import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from "node:path"

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src')
    }
  },
    server: {
    host: true,          // 等价于 '0.0.0.0'，允许局域网/热点访问
    port: 3000,
    strictPort: true,
    cors: true,
    proxy: {
      // 代理 Socket.IO 到后端（NestJS 默认 3001）
      "/socket.io": {
        target: "http://localhost:3001",
        ws: true,
        changeOrigin: true,
      },
    },
    // 如手机 HMR 无法连接，可手动指定：
    // hmr: { host: "你的电脑IP", protocol: "ws", port: 3000 },
  },
})
