/**
 * Simplifies a Go stack trace by removing hex arguments and offsets
 *
 * @param stacktrace The original stack trace string
 * @returns A simplified version of the stack trace
 */
export function simplifyStackTrace(stacktrace: string): string {
    if (!stacktrace) {
        return stacktrace;
    }

    // Process the stack trace line by line
    const lines = stacktrace.split("\n");
    const simplifiedLines = lines.map((line) => {
        // Remove hex arguments from function calls
        // Example: github.com/outrigdev/outrig/pkg/collector/watch.(*WatchCollector).doWatchSync(0x14000168100, {0x100bb16ee?, 0x0?}, {0x100c3bca0, 0x100d6f4c8}, {0x100bfd620?, 0x100d6f4b8?, 0x0?})
        // Becomes: github.com/outrigdev/outrig/pkg/collector/watch.(*WatchCollector).doWatchSync()
        if (line.includes("(") && line.includes(")") && !line.includes(".go:")) {
            // Find the function name and replace everything between parentheses
            const funcNameMatch = line.match(/^(\S+)\(/);
            if (funcNameMatch) {
                return `${funcNameMatch[1]}()`;
            }
        }

        // Remove offsets from file references
        // Example: /opt/homebrew/Cellar/go/1.23.4/libexec/src/fmt/print.go:702 +0x4b8
        // Becomes: /opt/homebrew/Cellar/go/1.23.4/libexec/src/fmt/print.go:702
        if (line.includes(".go:") && line.includes("+0x")) {
            return line.replace(/\s+\+0x[0-9a-f]+/, "");
        }

        return line;
    });

    return simplifiedLines.join("\n");
}
