import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import { resolve } from 'node:path';

export default defineConfig({
  plugins: [
    tailwindcss()
  ],
  build: {
    rollupOptions: {
      input: {
        iss: resolve(__dirname, 'iss/index.html'),
        presence: resolve(__dirname, 'presence/index.html'),
        todos: resolve(__dirname, 'todos/index.html'),
      },
    },
  },
});