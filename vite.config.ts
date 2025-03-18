import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { resolve } from "path";
import { defineConfig } from "vite";

export default defineConfig({
    plugins: [react(), tailwindcss()],
    resolve: {
        alias: {
            "@": resolve(__dirname, "./frontend"),
        },
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
