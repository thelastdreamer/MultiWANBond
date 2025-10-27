//go:build darwin
// +build darwin

package network

// newPlatformDetector creates the Darwin-specific network detector
func newPlatformDetector() (Detector, error) {
	return newDarwinDetector()
}
