package main

import (
	_ "embed"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
)

//go:embed assets/outrigapp-padded.png
var iconData []byte

func main() {
	// Start the systray
	systray.Run(onReady, onExit)
}

func onReady() {
	// Set up the systray icon and tooltip
	systray.SetIcon(iconData)
	systray.SetTooltip("Outrig")

	// Create menu items
	mOpen := systray.AddMenuItem("Open Outrig", "Open the Outrig web interface")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit Completely", "Quit the Application and Stop the Outrig Server")

	// Handle menu item clicks
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser("http://localhost:5005")
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	fmt.Printf("Exiting OutrigApp...\n")
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("Error opening browser: %v\n", err)
	}
}
