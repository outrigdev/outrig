import Foundation
import Sparkle
import Darwin     // for kill()
import AppKit     // for NSApplication

// ── parse CLI flags ─────────────────────────────────────────────
let args = CommandLine.arguments
let isBackground = args.contains("--background")
let isFirst = args.contains("--first")

guard
    let i = args.firstIndex(of: "--pid"),
    i + 1 < args.count,
    let trayPID = Int32(args[i + 1])
else {
    fputs("OutrigUpdater: missing --pid <tray-pid>\n", stderr)
    exit(1)
}

// ── Global state ────────────────────────────────────────────────
var validUpdateFound = false

// ── Initialize NSApplication ────────────────────────────────────
// This is critical for Sparkle UI to work properly
let app = NSApplication.shared

// Set activation policy based on mode
if isBackground {
    app.setActivationPolicy(.accessory) // Don't show in dock for background mode
} else {
    app.setActivationPolicy(.regular) // Show in dock for interactive and first modes
    app.activate(ignoringOtherApps: true) // Bring to foreground
}

// ── delegate ────────────────────────────────────────────────────
final class OutrigUpdaterDelegate: NSObject, SPUUpdaterDelegate {

    private let background: Bool
    private let trayPID: pid_t
    private var done = false

    init(background: Bool, trayPID: pid_t) {
        self.background = background
        self.trayPID    = trayPID
    }

    // ── Success path ────────────────────────────────────────────
    
    func updater(_ u: SPUUpdater, didFindValidUpdate item: SUAppcastItem) {
        print("Update \(item.displayVersionString) found → downloading…")
        validUpdateFound = true
        if !background {
            DispatchQueue.main.async {
                NSApplication.shared.activate(ignoringOtherApps: true)
            }
        }
    }
    
    func updater(_ updater: SPUUpdater, didDownloadUpdate item: SUAppcastItem) {
        print("Update downloaded successfully")
    }
    
    func updater(_ updater: SPUUpdater, willExtractUpdate item: SUAppcastItem) {
        print("Beginning update extraction")
    }
    
    func updater(_ updater: SPUUpdater, didExtractUpdate item: SUAppcastItem) {
        print("Update extracted successfully")
    }

    func updater(_ u: SPUUpdater, willInstallUpdate item: SUAppcastItem) {
        print("Download complete, staging install")
        if background { 
            // In background mode, the update is staged for next launch
            print("Background mode: Update staged for next launch")
            // Give Sparkle time to finish staging before quitting
            DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                self.quitHelper()
            }
        }
        // In interactive mode, continue with the update process
    }
    
    func updaterShouldRelaunchApplication(_ updater: SPUUpdater) -> Bool {
        print("Sparkle asking if should relaunch")
        if background {
            print("Background mode: Not relaunching application")
            return false
        }
        return true  // Yes, we want to relaunch
    }

    func updaterWillRelaunchApplication(_ u: SPUUpdater) {
        print("Sparkle will relaunch – SIGTERM tray (pid \(trayPID))")
        
        // Kill the parent tray app
        kill(trayPID, SIGTERM)
        
        // Give parent a moment to clean up before we exit
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) { 
            self.quitHelper() 
        }
    }

    // ── Failure paths ───────────────────────────────────────────
    
    func updaterDidNotFindUpdate(_ updater: SPUUpdater, error: Error) {
        print("No updates found: \(error.localizedDescription)")
        if !background {
            NSApplication.shared.activate(ignoringOtherApps: true)
        }
        // Don't quit here in interactive mode - let user see the dialog
        if background {
            quitHelper()
        }
    }
    
    func updater(_ u: SPUUpdater, didAbortWithError error: Error) {
        print("Update aborted with error: \(error)")
        if !background {
            NSApplication.shared.activate(ignoringOtherApps: true)
        }
        // Don't quit here in interactive mode - let user see the error
        if background {
            quitHelper()
        }
    }
    
    func updater(_ updater: SPUUpdater, failedToDownloadUpdate item: SUAppcastItem, error: Error) {
        print("Failed to download update: \(error)")
        // Don't quit here in interactive mode - let user see the error
        if background {
            quitHelper()
        }
    }
    
    func userDidCancelDownload(_ updater: SPUUpdater) {
        print("User cancelled download")
        quitHelper()  // User action, so we can quit
    }
    
    func updater(_ u: SPUUpdater, userDidSkipThisVersion item: SUAppcastItem) {
        print("User skipped version \(item.displayVersionString)")
        quitHelper()  // User action, so we can quit
    }
    
    // ── User choice handling ────────────────────────────────────
    
    func updater(_ updater: SPUUpdater,
                 userDidMake choice: SPUUserUpdateChoice,
                 forUpdate update: SUAppcastItem,
                 state: SPUUserUpdateState) {
        switch choice {
        case .install:
            print("User chose to install update")
            // Don't quit - let the update process continue
        case .skip:
            print("User skipped this version")
            quitHelper()
        case .dismiss:
            print("User dismissed update")
            quitHelper()
        @unknown default:
            print("Unknown user choice")
            quitHelper()
        }
    }
    
    // ── Optional logging methods ────────────────────────────────
    
    func updater(_ updater: SPUUpdater, didFinishLoading appcast: SUAppcast) {
        print("Appcast loaded successfully")
    }
    
    func updater(_ updater: SPUUpdater, didFinishUpdateCycleFor updateCheck: SPUUpdateCheck, error: Error?) {
        if let error = error {
            print("Update cycle finished with error: \(error)")
        } else {
            print("Update cycle finished successfully")
        }
        
        // Check if we should relaunch in interactive mode
        if isFirst && validUpdateFound {
            print("First mode found update - relaunching in interactive mode after 1s delay")
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
                updater.checkForUpdates()
            }
        } else {
            // This is the right place to quit after the update cycle completes
            // In interactive mode, this is called after the user dismisses any dialogs
            quitHelper()
        }
    }
    
    // ── UI activation methods ───────────────────────────────────
    
    func updater(_ updater: SPUUpdater, willShowModalAlert alert: NSAlert) {
        if !background {
            NSApplication.shared.activate(ignoringOtherApps: true)
        }
    }

    // ── Helper exit ─────────────────────────────────────────────
    
    func quitHelper() {
        guard !done else { return }
        done = true
        
        print("OutrigUpdater exiting")
        NSApplication.shared.terminate(nil)
    }
}

// ── start Sparkle ───────────────────────────────────────────────
let delegate = OutrigUpdaterDelegate(background: isBackground, trayPID: trayPID)

// Create updater controller
let updaterCtl = SPUStandardUpdaterController(
    startingUpdater: true,
    updaterDelegate: delegate,
    userDriverDelegate: nil
)

let updater = updaterCtl.updater

// Start update check
if isFirst {
    print("First mode – check for update information")
    updater.checkForUpdateInformation()
} else if isBackground {
    print("Background mode – silent check")
    updater.checkForUpdatesInBackground()
} else {
    print("Interactive mode – show Sparkle UI")
    updater.checkForUpdates()
}

// Run the app
app.run()