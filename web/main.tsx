import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
// CSS is now loaded via link tag in index.html
import { AppLoader } from "./apploader";
import { initRpcSystem, isDev } from "./init.ts";

initRpcSystem();

// Set document title based on development mode
if (isDev) {
    document.title = "Outrig (Dev)";
}

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

// Function to wait for fonts to load
const waitForFonts = async () => {
    // Create a set of font faces to load
    const interFont = new FontFace(
        "Inter", 
        "url(/fonts/inter-variable.woff2) format('woff2')",
        { 
            display: "block", // Use block instead of swap to prevent FOUT
            weight: "100 900"
        }
    );
    
    try {
        // Load the font
        const loadedFont = await interFont.load();
        
        // Add the font to the document
        document.fonts.add(loadedFont);
        
        // Wait for all fonts to be loaded
        await document.fonts.ready;
        
        console.log("All fonts loaded successfully");
    } catch (error) {
        console.error("Error loading fonts:", error);
        // Continue with rendering even if font loading fails
    }
};

// Main initialization function
const initialize = async () => {
    // Wait for both CSS and fonts to load
    const cssLoaded = new Promise<void>((resolve) => {
        if (window.outrigCssLoaded) {
            resolve();
        } else {
            document.addEventListener("outrig-css-loaded", () => resolve());
        }
    });
    
    try {
        // Wait for both CSS and fonts to load
        await Promise.all([cssLoaded, waitForFonts()]);
    } catch (error) {
        console.error("Error during initialization:", error);
    }
    
    // Render the app once everything is loaded
    renderApp();
};

// Start the initialization process
initialize();
