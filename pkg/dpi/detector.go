package dpi

import (
	"bytes"
	"encoding/binary"
	"sync"
)

// Detector performs deep packet inspection for protocol detection
type Detector struct {
	config     *DPIConfig
	signatures []*Signature
	mu         sync.RWMutex
}

// NewDetector creates a new DPI detector
func NewDetector(config *DPIConfig) *Detector {
	if config == nil {
		config = DefaultDPIConfig()
	}

	d := &Detector{
		config:     config,
		signatures: make([]*Signature, 0),
	}

	// Load default signatures
	d.loadDefaultSignatures()

	return d
}

// Classify inspects a packet and classifies its protocol
func (d *Detector) Classify(payload []byte, srcPort, dstPort uint16) *Classification {
	if !d.config.EnableProtocolDetection {
		return &Classification{
			Protocol:   ProtocolUnknown,
			Category:   CategoryUnknown,
			Confidence: 0.0,
		}
	}

	// Limit inspection depth
	inspectLen := len(payload)
	if inspectLen > d.config.InspectionDepth {
		inspectLen = d.config.InspectionDepth
	}
	data := payload[:inspectLen]

	// Try port-based detection first
	if protocol := d.detectByPort(srcPort, dstPort); protocol != ProtocolUnknown {
		return &Classification{
			Protocol:   protocol,
			Category:   protocol.GetCategory(),
			Confidence: 0.6,
			Matched:    []string{"port-based"},
		}
	}

	// Try signature-based detection
	d.mu.RLock()
	defer d.mu.RUnlock()

	bestMatch := &Classification{
		Protocol:   ProtocolUnknown,
		Category:   CategoryUnknown,
		Confidence: 0.0,
	}

	for _, sig := range d.signatures {
		if d.matchSignature(sig, data) {
			confidence := sig.Weight
			if confidence > bestMatch.Confidence {
				bestMatch = &Classification{
					Protocol:   sig.Protocol,
					Category:   sig.Protocol.GetCategory(),
					Confidence: confidence,
					Matched:    []string{sig.Name},
				}
			}
		}
	}

	return bestMatch
}

// detectByPort performs port-based protocol detection
func (d *Detector) detectByPort(srcPort, dstPort uint16) Protocol {
	portMap := map[uint16]Protocol{
		80:    ProtocolHTTP,
		443:   ProtocolHTTPS,
		8080:  ProtocolHTTP,
		8443:  ProtocolHTTPS,
		21:    ProtocolFTP,
		22:    ProtocolSSH,
		23:    ProtocolTelnet,
		25:    ProtocolSMTP,
		53:    ProtocolDNS,
		143:   ProtocolIMAP,
		110:   ProtocolPOP3,
		3389:  ProtocolRDP,
		5900:  ProtocolVNC,
		123:   ProtocolNTP,
		67:    ProtocolDHCP,
		68:    ProtocolDHCP,
		161:   ProtocolSNMP,
		1194:  ProtocolOpenVPN,
		51820: ProtocolWireGuard,
	}

	// Check destination port
	if proto, ok := portMap[dstPort]; ok {
		return proto
	}

	// Check source port
	if proto, ok := portMap[srcPort]; ok {
		return proto
	}

	return ProtocolUnknown
}

// matchSignature checks if a signature matches the payload
func (d *Detector) matchSignature(sig *Signature, payload []byte) bool {
	// Check length
	if len(payload) < sig.Offset+len(sig.Pattern) {
		return false
	}

	// Check depth
	if sig.Depth > 0 && len(payload) > sig.Depth {
		payload = payload[:sig.Depth]
	}

	// Search for pattern starting from offset
	searchStart := sig.Offset
	searchData := payload[searchStart:]

	return bytes.Contains(searchData, sig.Pattern)
}

// loadDefaultSignatures loads built-in protocol signatures
func (d *Detector) loadDefaultSignatures() {
	signatures := []*Signature{
		// HTTP
		{
			Name:     "HTTP GET",
			Protocol: ProtocolHTTP,
			Pattern:  []byte("GET /"),
			Offset:   0,
			Depth:    20,
			Weight:   0.9,
		},
		{
			Name:     "HTTP POST",
			Protocol: ProtocolHTTP,
			Pattern:  []byte("POST /"),
			Offset:   0,
			Depth:    20,
			Weight:   0.9,
		},
		{
			Name:     "HTTP Response",
			Protocol: ProtocolHTTP,
			Pattern:  []byte("HTTP/1."),
			Offset:   0,
			Depth:    20,
			Weight:   0.9,
		},

		// HTTPS/TLS
		{
			Name:     "TLS ClientHello",
			Protocol: ProtocolHTTPS,
			Pattern:  []byte{0x16, 0x03}, // TLS Handshake
			Offset:   0,
			Depth:    10,
			Weight:   0.8,
		},

		// DNS
		{
			Name:     "DNS Query",
			Protocol: ProtocolDNS,
			Pattern:  []byte{0x00, 0x00, 0x01, 0x00}, // Standard query
			Offset:   2,
			Depth:    12,
			Weight:   0.7,
		},

		// SSH
		{
			Name:     "SSH Protocol",
			Protocol: ProtocolSSH,
			Pattern:  []byte("SSH-2.0"),
			Offset:   0,
			Depth:    20,
			Weight:   0.95,
		},

		// BitTorrent
		{
			Name:     "BitTorrent Handshake",
			Protocol: ProtocolTorrent,
			Pattern:  []byte{0x13, 0x42, 0x69, 0x74, 0x54, 0x6f, 0x72, 0x72, 0x65, 0x6e, 0x74}, // "\x13BitTorrent"
			Offset:   0,
			Depth:    20,
			Weight:   0.95,
		},

		// YouTube (via SNI in TLS)
		{
			Name:     "YouTube SNI",
			Protocol: ProtocolYouTube,
			Pattern:  []byte("youtube.com"),
			Offset:   0,
			Depth:    1024,
			Weight:   0.85,
		},
		{
			Name:     "YouTubeVideo SNI",
			Protocol: ProtocolYouTube,
			Pattern:  []byte("googlevideo.com"),
			Offset:   0,
			Depth:    1024,
			Weight:   0.85,
		},

		// Netflix
		{
			Name:     "Netflix SNI",
			Protocol: ProtocolNetflix,
			Pattern:  []byte("netflix.com"),
			Offset:   0,
			Depth:    1024,
			Weight:   0.85,
		},

		// Facebook
		{
			Name:     "Facebook SNI",
			Protocol: ProtocolFacebook,
			Pattern:  []byte("facebook.com"),
			Offset:   0,
			Depth:    1024,
			Weight:   0.85,
		},

		// WhatsApp
		{
			Name:     "WhatsApp SNI",
			Protocol: ProtocolWhatsApp,
			Pattern:  []byte("whatsapp."),
			Offset:   0,
			Depth:    1024,
			Weight:   0.85,
		},

		// Zoom
		{
			Name:     "Zoom SNI",
			Protocol: ProtocolZoom,
			Pattern:  []byte("zoom.us"),
			Offset:   0,
			Depth:    1024,
			Weight:   0.85,
		},

		// Discord
		{
			Name:     "Discord SNI",
			Protocol: ProtocolDiscord,
			Pattern:  []byte("discord."),
			Offset:   0,
			Depth:    1024,
			Weight:   0.85,
		},

		// Spotify
		{
			Name:     "Spotify SNI",
			Protocol: ProtocolSpotify,
			Pattern:  []byte("spotify.com"),
			Offset:   0,
			Depth:    1024,
			Weight:   0.85,
		},

		// Steam
		{
			Name:     "Steam",
			Protocol: ProtocolSteam,
			Pattern:  []byte("steam"),
			Offset:   0,
			Depth:    1024,
			Weight:   0.8,
		},

		// Minecraft
		{
			Name:     "Minecraft Handshake",
			Protocol: ProtocolMinecraft,
			Pattern:  []byte{0x00}, // Handshake packet
			Offset:   2,
			Depth:    10,
			Weight:   0.6,
		},

		// RDP
		{
			Name:     "RDP Connection",
			Protocol: ProtocolRDP,
			Pattern:  []byte{0x03, 0x00}, // TPKT header
			Offset:   0,
			Depth:    10,
			Weight:   0.8,
		},

		// SMTP
		{
			Name:     "SMTP Hello",
			Protocol: ProtocolSMTP,
			Pattern:  []byte("EHLO"),
			Offset:   0,
			Depth:    20,
			Weight:   0.9,
		},
	}

	d.mu.Lock()
	d.signatures = append(d.signatures, signatures...)
	d.mu.Unlock()
}

// AddSignature adds a custom signature
func (d *Detector) AddSignature(sig *Signature) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.signatures = append(d.signatures, sig)
}

// GetSignatures returns all signatures
func (d *Detector) GetSignatures() []*Signature {
	d.mu.RLock()
	defer d.mu.RUnlock()

	sigs := make([]*Signature, len(d.signatures))
	copy(sigs, d.signatures)
	return sigs
}

// ClassifyTLS attempts to extract SNI from TLS ClientHello
func (d *Detector) ClassifyTLS(payload []byte) *Classification {
	if len(payload) < 43 {
		return &Classification{Protocol: ProtocolHTTPS, Confidence: 0.5}
	}

	// Check for TLS handshake (0x16) and ClientHello (0x01)
	if payload[0] != 0x16 || payload[5] != 0x01 {
		return &Classification{Protocol: ProtocolHTTPS, Confidence: 0.5}
	}

	// Try to extract SNI
	sni := d.extractSNI(payload)
	if sni == "" {
		return &Classification{Protocol: ProtocolHTTPS, Confidence: 0.6}
	}

	// Match SNI against known services
	protocol := d.matchSNI(sni)
	if protocol != ProtocolUnknown {
		return &Classification{
			Protocol:   protocol,
			Category:   protocol.GetCategory(),
			Confidence: 0.9,
			Matched:    []string{"SNI: " + sni},
		}
	}

	return &Classification{
		Protocol:   ProtocolHTTPS,
		Category:   CategoryWeb,
		Confidence: 0.7,
	}
}

// extractSNI extracts Server Name Indication from TLS ClientHello
func (d *Detector) extractSNI(payload []byte) string {
	if len(payload) < 43 {
		return ""
	}

	// Skip to extensions
	pos := 43
	if pos >= len(payload) {
		return ""
	}

	// Session ID length
	sessionIDLen := int(payload[pos])
	pos += 1 + sessionIDLen

	if pos+2 >= len(payload) {
		return ""
	}

	// Cipher suites length
	cipherSuitesLen := int(binary.BigEndian.Uint16(payload[pos : pos+2]))
	pos += 2 + cipherSuitesLen

	if pos+1 >= len(payload) {
		return ""
	}

	// Compression methods length
	compressionLen := int(payload[pos])
	pos += 1 + compressionLen

	if pos+2 >= len(payload) {
		return ""
	}

	// Extensions length
	extensionsLen := int(binary.BigEndian.Uint16(payload[pos : pos+2]))
	pos += 2

	endPos := pos + extensionsLen
	if endPos > len(payload) {
		return ""
	}

	// Parse extensions
	for pos+4 < endPos {
		extType := binary.BigEndian.Uint16(payload[pos : pos+2])
		extLen := int(binary.BigEndian.Uint16(payload[pos+2 : pos+4]))
		pos += 4

		if pos+extLen > len(payload) {
			return ""
		}

		// SNI extension (type 0)
		if extType == 0 && extLen > 5 {
			sniLen := int(binary.BigEndian.Uint16(payload[pos+3 : pos+5]))
			if pos+5+sniLen <= len(payload) {
				return string(payload[pos+5 : pos+5+sniLen])
			}
		}

		pos += extLen
	}

	return ""
}

// matchSNI matches SNI against known services
func (d *Detector) matchSNI(sni string) Protocol {
	sniMap := map[string]Protocol{
		"youtube.com":      ProtocolYouTube,
		"googlevideo.com":  ProtocolYouTube,
		"netflix.com":      ProtocolNetflix,
		"nflxvideo.net":    ProtocolNetflix,
		"facebook.com":     ProtocolFacebook,
		"fbcdn.net":        ProtocolFacebook,
		"instagram.com":    ProtocolInstagram,
		"cdninstagram.com": ProtocolInstagram,
		"twitter.com":      ProtocolTwitter,
		"twimg.com":        ProtocolTwitter,
		"tiktok.com":       ProtocolTikTok,
		"whatsapp.com":     ProtocolWhatsApp,
		"whatsapp.net":     ProtocolWhatsApp,
		"zoom.us":          ProtocolZoom,
		"discord.com":      ProtocolDiscord,
		"discordapp.com":   ProtocolDiscord,
		"spotify.com":      ProtocolSpotify,
		"scdn.co":          ProtocolSpotify,
		"teams.microsoft.com": ProtocolTeams,
		"dropbox.com":      ProtocolDropbox,
	}

	// Check exact match
	if proto, ok := sniMap[sni]; ok {
		return proto
	}

	// Check if SNI contains any known domain
	for domain, proto := range sniMap {
		if bytes.Contains([]byte(sni), []byte(domain)) {
			return proto
		}
	}

	return ProtocolUnknown
}
