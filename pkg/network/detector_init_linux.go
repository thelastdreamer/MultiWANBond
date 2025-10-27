//go:build linux
// +build linux

package network

// newPlatformDetector creates the Linux-specific network detector
func newPlatformDetector() (Detector, error) {
	return newLinuxDetector()
}
