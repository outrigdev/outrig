import Foundation
import Sparkle

class OutrigUpdaterDelegate: NSObject, SPUUpdaterDelegate {
    func updater(_ updater: SPUUpdater, didFindValidUpdate item: SUAppcastItem) {
        print("Found valid update: \(item.displayVersionString)")
    }
    
    func updaterDidNotFindUpdate(_ updater: SPUUpdater) {
        print("No updates available")
    }
}

// Parse command line arguments
let arguments = CommandLine.arguments
let isBackgroundMode = arguments.contains("--background")

// Set up the delegate
let delegate = OutrigUpdaterDelegate()

// Create the updater with the delegate
let updaterController = SPUStandardUpdaterController(startingUpdater: true, updaterDelegate: delegate, userDriverDelegate: nil)
let updater = updaterController.updater

if isBackgroundMode {
    print("Running in background mode - checking for updates silently")
    // Check for updates silently in background mode
    updater.checkForUpdatesInBackground()
} else {
    print("Running in interactive mode - showing update UI")
    // Show the update UI in interactive mode
    updater.checkForUpdates()
}

// Keep the app running until update check completes
RunLoop.main.run()
