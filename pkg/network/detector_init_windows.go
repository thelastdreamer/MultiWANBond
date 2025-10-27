//go:build windows
// +build windows

package network

// newPlatformDetector creates the Windows-specific network detector
func newPlatformDetector() (Detector, error) {
	return newWindowsDetector()
}
