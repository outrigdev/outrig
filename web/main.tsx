import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./app.css";
import { App } from "./app.tsx";
import { initRpcSystem } from "./init.ts";

initRpcSystem();

const params = new URLSearchParams(window.location.search);
const isStrict = params.get("strict") != "0";

createRoot(document.getElementById("root")!).render(
    isStrict ? (
        <StrictMode>
            <App />
        </StrictMode>
    ) : (
        <App />
    )
);
