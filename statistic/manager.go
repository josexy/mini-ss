package statistic

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	TrafficSpeedTime    = 2 * time.Second
	TrafficSnapshotTime = 5 * time.Second
)

var DefaultManager *TrackerManager = NewTrackerManager()

var EnableStatistic = false

type ConnectionSnapshot struct {
	Context
	DownloadTotal int64 `json:"download"`
	UploadTotal   int64 `json:"upload"`
}

type Snapshot struct {
	DownloadTotal int64                 `json:"download_total"`
	UploadTotal   int64                 `json:"upload_total"`
	Connections   []*ConnectionSnapshot `json:"connections"`
}

type TrackerManager struct {
	trackers            sync.Map
	downloadPerSec      atomic.Int64
	uploadPerSec        atomic.Int64
	downloadPerSecDelta atomic.Int64
	uploadPerSecDelta   atomic.Int64
	downloadTotal       atomic.Int64
	uploadTotal         atomic.Int64
	stopped             chan struct{}
}

func NewTrackerManager() *TrackerManager {
	manager := &TrackerManager{
		stopped: make(chan struct{}),
	}
	go manager.listen()
	return manager
}

func (manager *TrackerManager) Add(tracker Tracker) {
	if tracker == nil {
		return
	}
	manager.trackers.Store(tracker.ID(), tracker)
}

func (manager *TrackerManager) Remove(tracker Tracker) {
	if tracker == nil {
		return
	}
	manager.trackers.Delete(tracker.ID())
}

func (manager *TrackerManager) Reset() {
	manager.downloadPerSec.Store(0)
	manager.uploadPerSec.Store(0)
	manager.downloadPerSecDelta.Store(0)
	manager.uploadPerSecDelta.Store(0)
	manager.downloadTotal.Store(0)
	manager.uploadTotal.Store(0)
}

func (manager *TrackerManager) CloseAll() {
	manager.stopped <- struct{}{}
	manager.trackers.Range(func(key, value any) bool {
		value.(Tracker).Close()
		return true
	})
}

func (manager *TrackerManager) TrafficSpeed() (download, upload int64) {
	return manager.downloadPerSec.Load(), manager.uploadPerSec.Load()
}

func (manager *TrackerManager) DumpSnapshot() Snapshot {
	snapshot := Snapshot{
		DownloadTotal: manager.downloadTotal.Load(),
		UploadTotal:   manager.uploadTotal.Load(),
	}
	manager.trackers.Range(func(key, value any) bool {
		info := value.(Tracker).TrackerInfo()
		snapshot.Connections = append(snapshot.Connections, &ConnectionSnapshot{
			Context:       info.Context,
			DownloadTotal: info.Context.downloadTotal.Load(),
			UploadTotal:   info.Context.uploadTotal.Load(),
		})
		return true
	})
	return snapshot
}

func (manager *TrackerManager) listen() {
	ticker := time.NewTicker(TrafficSpeedTime)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			manager.downloadPerSec.Store(manager.downloadPerSecDelta.Load())
			manager.downloadPerSecDelta.Store(0)
			manager.uploadPerSec.Store(manager.uploadPerSecDelta.Load())
			manager.uploadPerSecDelta.Store(0)
		case <-manager.stopped:
			return
		}
	}
}
