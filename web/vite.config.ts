import path from "path"
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, "index.html"),
        ...(process.env.VITE_ENABLE_DEMOS === "true" && {
          demo: path.resolve(__dirname, "demo.html"),
          "dice-demo": path.resolve(__dirname, "dice-demo.html"),
        }),
      },
    },
  },
})
