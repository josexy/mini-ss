package statistic

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/atomic"
)

var DefaultManager *Manager

type ConnectionSnapshot struct {
	ID            string    `json:"id"`
	StartTime     time.Time `json:"start_time"`
	Network       string    `json:"network"`
	Src           string    `json:"src"`
	Dst           string    `json:"dst"`
	DownloadTotal int64     `json:"download_total"`
	UploadTotal   int64     `json:"upload_total"`
	Host          string    `json:"host"`
	RuleMode      string    `json:"rule_mode"`
	RuleType      string    `json:"rule_type"`
	Proxy         string    `json:"proxy"`
}

type LazyContext struct {
	Host     string `json:"host"`
	RuleMode string `json:"rule_mode"`
	RuleType string `json:"rule_type"`
	Proxy    string `json:"proxy"`
}

type AllDumpSnapshot struct {
	DownloadTotal int64                 `json:"download_total"`
	UploadTotal   int64                 `json:"upload_total"`
	Connections   []*ConnectionSnapshot `json:"connections"`
}

type Manager struct {
	connections    sync.Map
	downloadCur    *atomic.Int64
	downloadPerSec *atomic.Int64
	uploadCur      *atomic.Int64
	uploadPerSec   *atomic.Int64
	downloadTotal  *atomic.Int64
	uploadTotal    *atomic.Int64
}

func InitGlobalStatisticManager() {
	if DefaultManager != nil {
		// reset statistic status information
		DefaultManager.ResetStatistic()
		DefaultManager.clearMap()
		return
	}
	DefaultManager = &Manager{
		downloadCur:    atomic.NewInt64(0),
		downloadPerSec: atomic.NewInt64(0),
		downloadTotal:  atomic.NewInt64(0),
		uploadCur:      atomic.NewInt64(0),
		uploadPerSec:   atomic.NewInt64(0),
		uploadTotal:    atomic.NewInt64(0),
	}

	go DefaultManager.handle()
}

func (m *Manager) clearMap() {
	m.connections.Range(func(key, value any) bool {
		value.(tracker).Close()
		return true
	})
}

func (m *Manager) LazySet(addr string, ctx LazyContext) {
	var tr tracker
	m.connections.Range(func(_, value any) bool {
		t := value.(tracker)
		if t.TrackerInfo().src == addr {
			tr = t
			return false
		}
		return true
	})
	if tr != nil {
		tr.TrackerInfo().LazyContext = ctx
	}
}

func (m *Manager) Add(tracker tracker) {
	if tracker == nil {
		return
	}
	m.connections.Store(tracker.ID(), tracker)
}

func (m *Manager) Remove(tracker tracker) {
	if tracker == nil {
		return
	}
	m.connections.Delete(tracker.ID())
}

// UpdateDownload when tcp or udp calls Read()/ReadFrom(), update the downloaded delta and total data size
func (m *Manager) UpdateDownload(n int64) {
	m.downloadCur.Add(n)
	m.downloadTotal.Add(n)
}

// UpdateUpload when tcp or udp calls Write()/WriteTo(), update the uploaded delta and total data size
func (m *Manager) UpdateUpload(n int64) {
	m.uploadCur.Add(n)
	m.uploadTotal.Add(n)
}

// TrafficSpeedTick return current download and upload rate per second
func (m *Manager) TrafficSpeedTick() (download, upload int64) {
	return m.downloadPerSec.Load(), m.uploadPerSec.Load()
}

func (m *Manager) handle() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// reset
		m.uploadPerSec.Store(m.uploadCur.Load())
		m.uploadCur.Store(0)

		m.downloadPerSec.Store(m.downloadCur.Load())
		m.downloadCur.Store(0)
	}
}

func (m *Manager) ResetStatistic() {
	m.downloadCur.Store(0)
	m.downloadPerSec.Store(0)
	m.uploadCur.Store(0)
	m.uploadPerSec.Store(0)
}

func (m *Manager) Snapshot() *AllDumpSnapshot {
	var connections []*ConnectionSnapshot

	m.connections.Range(func(id, value any) bool {
		trackerInfo := value.(tracker).TrackerInfo()
		connections = append(connections, &ConnectionSnapshot{
			ID:            id.(uuid.UUID).String(),
			Network:       trackerInfo.network,
			StartTime:     trackerInfo.start,
			Src:           trackerInfo.src,
			Dst:           trackerInfo.dst,
			DownloadTotal: trackerInfo.downloadTotal.Load(),
			UploadTotal:   trackerInfo.uploadTotal.Load(),
			Host:          trackerInfo.Host,
			RuleMode:      trackerInfo.RuleMode,
			RuleType:      trackerInfo.RuleType,
			Proxy:         trackerInfo.Proxy,
		})
		return true
	})
	return &AllDumpSnapshot{
		DownloadTotal: m.downloadTotal.Load(),
		UploadTotal:   m.uploadTotal.Load(),
		Connections:   connections,
	}
}
