//go:build !go1.23

package controller

// initCrashOutput is a no-op for Go versions before 1.23
func (c *ControllerImpl) initCrashOutput() {
	// No-op for Go versions before 1.23
}
