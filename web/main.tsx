import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
// CSS is now loaded via link tag in index.html
import { AppLoader } from "./apploader";
import { initRpcSystem } from "./init.ts";

initRpcSystem();

const params = new URLSearchParams(window.location.search);
const isStrict = params.get("strict") != "0";

// Function to render the app
const renderApp = () => {
    createRoot(document.getElementById("root")!).render(
        isStrict ? (
            <StrictMode>
                <AppLoader />
            </StrictMode>
        ) : (
            <AppLoader />
        )
    );
};

// Check if CSS is already loaded or wait for it
if (window.outrigCssLoaded) {
    // CSS already loaded, render immediately
    renderApp();
} else {
    // Wait for CSS to load before rendering
    document.addEventListener("outrig-css-loaded", renderApp);
}
