// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { resolve } from "path";
import { defineConfig } from "vite";
import pkg from "./package.json";

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
            "/ws": {
                target: "http://localhost:6005",
                ws: true,
            },
        },
    },
    build: {
        outDir: "dist-fe", // Changed from default "dist" to "dist-fe"
    },
    define: {
        "import.meta.env.PACKAGE_VERSION": JSON.stringify(pkg.version),
    },
});
