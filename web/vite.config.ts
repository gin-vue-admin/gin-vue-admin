import { defineConfig } from 'vite'
import path from 'path'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
// https://vitejs.dev/config/
// base 可通过 VITE_BASE 环境变量覆盖（云主机部署到根路径用 '/'，子路径用 '/xxx/'）；
// 默认开发为 ''，生产为 '/vue-admin/'（GitHub Pages 子路径）
const base = process.env.VITE_BASE ?? (process.env.NODE_ENV === 'production' ? '/vue-admin/' : '')
// dev server 端口：默认 5173（vite 标准）；端口冲突用 VITE_PORT 覆盖
const port = Number(process.env.VITE_PORT) || 5173
// 后端 API 地址：默认 http://localhost:8080（与 server/configs/config.yaml 对齐）；
// 本机后端跑在非默认端口时用 GVA_API_TARGET 覆盖
const apiTarget = process.env.GVA_API_TARGET ?? 'http://localhost:8080'

export default defineConfig({
  base,
  server: {
    host: true,
    port,
    watch: { usePolling: true },
    hmr: true,
    // 真实后端转发：/api/* 与 /swagger → gva server（关 MSW 后前端走真实接口）
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true
      },
      '/swagger': {
        target: apiTarget,
        changeOrigin: true
      }
    }
  },
  plugins: [
    vue(),
    vueJsx(),
    AutoImport({
      //imports: ['vue'],
      //dts: 'src/auto-import.d.ts',
      resolvers: [ElementPlusResolver()]
    }),
    Components({
      //dts: 'src/commponents.d.ts',
      resolvers: [ElementPlusResolver()]
    }),

  ],
  css: {
    // css预处理器
    preprocessorOptions: {
      scss: {
        // 引入 mixin.scss 这样就可以在全局中使用 mixin.scss中预定义的变量了
        // 给导入的路径最后加上 ;
        //additionalData: '@import "@/assets/main.scss";'
      }
    }
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src')
    },
    extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue']
  },
  build: {
    // 代码分割配置
    rollupOptions: {
      output: {
        manualChunks: {
          // 将 Element Plus 相关代码单独打包
          'element-plus': ['element-plus'],
          // 将 Vue 相关代码单独打包
          vue: ['vue', 'vue-router', 'pinia'],
          // 将其他第三方库单独打包
          vendor: ['axios', '@vueuse/core'],
          // ECharts 独立分包（仅 dashboard 懒加载，独立缓存）
          echarts: ['echarts']
        }
      }
    },
    // 压缩配置
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true
      }
    },
    // 打包文件大小限制
    chunkSizeWarningLimit: 1000
  }
})
