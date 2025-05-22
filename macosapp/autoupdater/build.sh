#!/bin/bash
set -e

# Build the OutrigUpdater in release mode
echo "Building OutrigUpdater..."
swift build -c release

# Get the path to the built binary
BINARY_PATH=$(swift build -c release --show-bin-path)/OutrigUpdater

echo "Built binary at: $BINARY_PATH"
echo "Copy this binary to Outrig.app/Contents/MacOS/OutrigUpdater"

# Instructions for the Sparkle framework
echo ""
echo "NEXT STEPS:"
echo "1. Download Sparkle.framework from https://github.com/sparkle-project/Sparkle/releases"
echo "2. Create directory: Outrig.app/Contents/Frameworks/"
echo "3. Copy Sparkle.framework to Outrig.app/Contents/Frameworks/"
echo "4. Copy $BINARY_PATH to Outrig.app/Contents/MacOS/OutrigUpdater"
echo "5. Update Outrig.app/Contents/Info.plist with Sparkle configuration"
echo ""
echo "Example Info.plist Sparkle configuration:"
echo "<key>SUFeedURL</key>"
echo "<string>https://updates.outrig.run/appcast.xml</string>"
echo "<key>SUPublicEDKey</key>"
echo "<string>[Your Sparkle public key]</string>"
echo "<key>SUAutomaticallyUpdate</key>"
echo "<true/>"