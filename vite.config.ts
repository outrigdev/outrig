import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { resolve } from "path";

export default defineConfig({
    plugins: [react(), tailwindcss()],
    resolve: {
        alias: {
            "@": resolve(__dirname, "./web")
        }
    },
    server: {
        watch: {
            ignored: ["**/*.go", "go.mod", "go.sum", "**/*.md"],
        },
        proxy: {
            "/api": "http://localhost:6005",
        },
    },
});
