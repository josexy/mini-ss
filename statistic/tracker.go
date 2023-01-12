package statistic

import (
	"net"
	"time"

	"github.com/google/uuid"
	"go.uber.org/atomic"
)

type trackerInfo struct {
	id            uuid.UUID
	start         time.Time // the connection created time
	downloadTotal *atomic.Int64
	uploadTotal   *atomic.Int64
	network       string
	src           string
	dst           string
	LazyContext
}

type tracker interface {
	ID() uuid.UUID
	Close() error
	TrackerInfo() *trackerInfo
}

type tcpTracker struct {
	net.Conn
	*trackerInfo
	manager *Manager
}

func NewTcpTracker(c net.Conn, src, dst string, ctx LazyContext, manager *Manager) *tcpTracker {
	tracker := &tcpTracker{
		Conn:    c,
		manager: manager,
		trackerInfo: &trackerInfo{
			network:       "tcp",
			src:           src,
			dst:           dst,
			start:         time.Now(),
			id:            uuid.New(),
			downloadTotal: atomic.NewInt64(0),
			uploadTotal:   atomic.NewInt64(0),
			LazyContext:   ctx,
		},
	}
	manager.Add(tracker)
	return tracker
}

func (t *tcpTracker) ID() uuid.UUID {
	return t.id
}

func (t *tcpTracker) TrackerInfo() *trackerInfo {
	return t.trackerInfo
}

func (t *tcpTracker) Read(b []byte) (int, error) {
	n, err := t.Conn.Read(b)
	upload := int64(n)
	t.manager.UpdateUpload(upload)
	t.uploadTotal.Add(upload)
	return n, err
}

func (t *tcpTracker) Write(b []byte) (int, error) {
	n, err := t.Conn.Write(b)
	download := int64(n)
	t.manager.UpdateDownload(download)
	t.downloadTotal.Add(download)
	return n, err
}

func (t *tcpTracker) Close() error {
	t.manager.Remove(t)
	return t.Conn.Close()
}

type udpTracker struct {
	net.PacketConn
	*trackerInfo
	manager *Manager
}

func NewUdpTracker(c net.PacketConn, src, dst string, ctx LazyContext, manager *Manager) *udpTracker {
	tracker := &udpTracker{
		PacketConn: c,
		manager:    manager,
		trackerInfo: &trackerInfo{
			network:       "udp",
			src:           src,
			dst:           dst,
			start:         time.Now(),
			id:            uuid.New(),
			downloadTotal: atomic.NewInt64(0),
			uploadTotal:   atomic.NewInt64(0),
			LazyContext:   ctx,
		},
	}
	manager.Add(tracker)
	return tracker
}

func (t *udpTracker) ID() uuid.UUID {
	return t.id
}

func (t *udpTracker) TrackerInfo() *trackerInfo {
	return t.trackerInfo
}

func (t *udpTracker) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := t.PacketConn.ReadFrom(b)
	upload := int64(n)
	t.manager.UpdateUpload(upload)
	t.uploadTotal.Add(upload)
	return n, addr, err
}

func (t *udpTracker) WriteTo(b []byte, addr net.Addr) (int, error) {
	n, err := t.PacketConn.WriteTo(b, addr)
	download := int64(n)
	t.manager.UpdateDownload(download)
	t.downloadTotal.Add(download)
	return n, err
}

func (t *udpTracker) Close() error {
	t.manager.Remove(t)
	return t.PacketConn.Close()
}
