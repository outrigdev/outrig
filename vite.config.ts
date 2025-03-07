import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
    plugins: [react(), tailwindcss()],
    server: {
        watch: {
            ignored: ["**/*.go", "go.mod", "go.sum"],
        },
        proxy: {
            "/api": "http://localhost:5005",
        },
    },
});
