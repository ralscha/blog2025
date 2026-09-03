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
        iss: resolve(import.meta.dirname, 'iss/index.html'),
        presence: resolve(import.meta.dirname, 'presence/index.html'),
        todos: resolve(import.meta.dirname, 'todos/index.html'),
      },
    },
  },
});
