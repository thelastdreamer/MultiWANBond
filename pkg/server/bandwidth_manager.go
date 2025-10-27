package server

import (
	"sync"
	"time"
)

// BandwidthManager manages bandwidth allocation and accounting for clients
type BandwidthManager struct {
	mu                sync.RWMutex
	sessionManager    *SessionManager
	totalUpload       uint64 // Total server upload capacity (bytes/sec)
	totalDownload     uint64 // Total server download capacity (bytes/sec)
	currentUpload     uint64 // Current upload usage
	currentDownload   uint64 // Current download usage
	measurementWindow time.Duration
	stopChan          chan struct{}
	wg                sync.WaitGroup
}

// NewBandwidthManager creates a new bandwidth manager
func NewBandwidthManager(sessionManager *SessionManager, totalUpload, totalDownload uint64) *BandwidthManager {
	return &BandwidthManager{
		sessionManager:    sessionManager,
		totalUpload:       totalUpload,
		totalDownload:     totalDownload,
		measurementWindow: 1 * time.Second,
		stopChan:          make(chan struct{}),
	}
}

// Start starts the bandwidth manager
func (bm *BandwidthManager) Start() {
	bm.wg.Add(2)
	go bm.measurementRoutine()
	go bm.quotaCheckRoutine()
}

// Stop stops the bandwidth manager
func (bm *BandwidthManager) Stop() {
	close(bm.stopChan)
	bm.wg.Wait()
}

// AccountTraffic accounts traffic for a session
func (bm *BandwidthManager) AccountTraffic(sessionID string, uploaded, downloaded uint64) error {
	session, err := bm.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Update session counters
	session.mu.Lock()
	session.BytesSent += uploaded
	session.BytesReceived += downloaded
	session.mu.Unlock()

	// Update bandwidth quota
	session.BandwidthQuota.UpdateBandwidth(uploaded, downloaded)

	// Check quota
	if session.BandwidthQuota.CheckQuota(
		session.Config.DailyDataQuota,
		session.Config.MonthlyDataQuota,
	) {
		if !session.BandwidthQuota.QuotaExceeded {
			session.BandwidthQuota.QuotaExceeded = true

			// Send quota exceeded event
			bm.sessionManager.sendEvent(SessionEvent{
				Type:      EventQuotaExceeded,
				SessionID: sessionID,
				ClientID:  session.ClientID,
				Timestamp: time.Now(),
				Details:   "Data quota exceeded",
			})
		}
	}

	return nil
}

// CheckBandwidthLimit checks if a client can send/receive data
func (bm *BandwidthManager) CheckBandwidthLimit(sessionID string, uploadBytes, downloadBytes uint64) bool {
	session, err := bm.sessionManager.GetSession(sessionID)
	if err != nil {
		return false
	}

	// Check if quota exceeded
	if session.BandwidthQuota.QuotaExceeded {
		return false
	}

	// Check rate limits
	session.BandwidthQuota.mu.RLock()
	currentUp := session.BandwidthQuota.CurrentUpload
	currentDown := session.BandwidthQuota.CurrentDownload
	maxUp := session.BandwidthQuota.MaxUpload
	maxDown := session.BandwidthQuota.MaxDownload
	session.BandwidthQuota.mu.RUnlock()

	// Check if adding this traffic would exceed limits
	if currentUp+uploadBytes > maxUp {
		return false // Upload limit exceeded
	}

	if currentDown+downloadBytes > maxDown {
		return false // Download limit exceeded
	}

	// Check server-wide limits
	bm.mu.RLock()
	serverUpOk := bm.currentUpload+uploadBytes <= bm.totalUpload
	serverDownOk := bm.currentDownload+downloadBytes <= bm.totalDownload
	bm.mu.RUnlock()

	return serverUpOk && serverDownOk
}

// GetClientBandwidth returns current bandwidth usage for a client
func (bm *BandwidthManager) GetClientBandwidth(sessionID string) (upload, download uint64, err error) {
	session, err := bm.sessionManager.GetSession(sessionID)
	if err != nil {
		return 0, 0, err
	}

	session.BandwidthQuota.mu.RLock()
	upload = session.BandwidthQuota.CurrentUpload
	download = session.BandwidthQuota.CurrentDownload
	session.BandwidthQuota.mu.RUnlock()

	return upload, download, nil
}

// GetServerBandwidth returns current server-wide bandwidth usage
func (bm *BandwidthManager) GetServerBandwidth() (upload, download uint64) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.currentUpload, bm.currentDownload
}

// measurementRoutine periodically measures bandwidth usage
func (bm *BandwidthManager) measurementRoutine() {
	defer bm.wg.Done()

	ticker := time.NewTicker(bm.measurementWindow)
	defer ticker.Stop()

	// Track previous measurements
	type sessionMeasurement struct {
		prevBytesSent uint64
		prevBytesRecv uint64
	}
	prevMeasurements := make(map[string]*sessionMeasurement)

	for {
		select {
		case <-bm.stopChan:
			return
		case <-ticker.C:
			sessions := bm.sessionManager.GetAllSessions()

			var totalUpload uint64 = 0
			var totalDownload uint64 = 0

			for _, session := range sessions {
				session.mu.RLock()
				bytesSent := session.BytesSent
				bytesRecv := session.BytesReceived
				session.mu.RUnlock()

				// Get previous measurement
				prev, exists := prevMeasurements[session.ID]
				if !exists {
					prev = &sessionMeasurement{
						prevBytesSent: bytesSent,
						prevBytesRecv: bytesRecv,
					}
					prevMeasurements[session.ID] = prev
					continue
				}

				// Calculate rate (bytes per measurement window)
				uploadRate := bytesSent - prev.prevBytesSent
				downloadRate := bytesRecv - prev.prevBytesRecv

				// Update session bandwidth
				session.BandwidthQuota.mu.Lock()
				session.BandwidthQuota.CurrentUpload = uploadRate
				session.BandwidthQuota.CurrentDownload = downloadRate
				session.BandwidthQuota.mu.Unlock()

				// Update previous
				prev.prevBytesSent = bytesSent
				prev.prevBytesRecv = bytesRecv

				// Add to totals
				totalUpload += uploadRate
				totalDownload += downloadRate
			}

			// Update server totals
			bm.mu.Lock()
			bm.currentUpload = totalUpload
			bm.currentDownload = totalDownload
			bm.mu.Unlock()

			// Clean up removed sessions
			activeSessions := make(map[string]bool)
			for _, session := range sessions {
				activeSessions[session.ID] = true
			}
			for sessionID := range prevMeasurements {
				if !activeSessions[sessionID] {
					delete(prevMeasurements, sessionID)
				}
			}
		}
	}
}

// quotaCheckRoutine periodically checks and resets quotas
func (bm *BandwidthManager) quotaCheckRoutine() {
	defer bm.wg.Done()

	// Check daily reset at midnight
	dailyTicker := time.NewTicker(1 * time.Hour)
	defer dailyTicker.Stop()

	lastDailyReset := time.Now()
	lastMonthlyReset := time.Now()

	for {
		select {
		case <-bm.stopChan:
			return
		case now := <-dailyTicker.C:
			// Check if we crossed midnight (daily reset)
			if now.Day() != lastDailyReset.Day() {
				bm.resetDailyQuotas()
				lastDailyReset = now
			}

			// Check if we crossed month boundary (monthly reset)
			if now.Month() != lastMonthlyReset.Month() {
				bm.resetMonthlyQuotas()
				lastMonthlyReset = now
			}
		}
	}
}

// resetDailyQuotas resets daily quotas for all sessions
func (bm *BandwidthManager) resetDailyQuotas() {
	sessions := bm.sessionManager.GetAllSessions()

	for _, session := range sessions {
		session.BandwidthQuota.ResetDaily()
		session.BandwidthQuota.QuotaExceeded = false
	}
}

// resetMonthlyQuotas resets monthly quotas for all sessions
func (bm *BandwidthManager) resetMonthlyQuotas() {
	sessions := bm.sessionManager.GetAllSessions()

	for _, session := range sessions {
		session.BandwidthQuota.ResetMonthly()
		session.BandwidthQuota.QuotaExceeded = false
	}
}

// GetTotalTraffic returns total traffic for a session
func (bm *BandwidthManager) GetTotalTraffic(sessionID string) (uploaded, downloaded uint64, err error) {
	session, err := bm.sessionManager.GetSession(sessionID)
	if err != nil {
		return 0, 0, err
	}

	session.BandwidthQuota.mu.RLock()
	uploaded = session.BandwidthQuota.TotalUploaded
	downloaded = session.BandwidthQuota.TotalDownloaded
	session.BandwidthQuota.mu.RUnlock()

	return uploaded, downloaded, nil
}

// SetClientBandwidthLimit updates bandwidth limits for a client
func (bm *BandwidthManager) SetClientBandwidthLimit(sessionID string, maxUpload, maxDownload uint64) error {
	session, err := bm.sessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.BandwidthQuota.mu.Lock()
	session.BandwidthQuota.MaxUpload = maxUpload
	session.BandwidthQuota.MaxDownload = maxDownload
	session.BandwidthQuota.mu.Unlock()

	return nil
}
