package statistic

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Context struct {
	Src           string       `json:"src"`     // client remote ip address
	Dst           string       `json:"dst"`     // target domain name or ip address
	Network       string       `json:"network"` // connection network ['tcp', 'udp']
	Type          string       `json:"type"`    // connection type ['socks', 'http', 'tcp-tun', 'udp-tun', 'simple-tcp-tun']
	Rule          string       `json:"rule"`    // matched rule type
	Proxy         string       `json:"proxy"`   // matched proxy
	downloadTotal atomic.Int64 // download
	uploadTotal   atomic.Int64 // upload
}

type TrackerInfo struct {
	id    uuid.UUID
	start time.Time
	Context
}

type Tracker interface {
	ID() uuid.UUID
	TrackerInfo() *TrackerInfo
	Close() error
}

type tcpTracker struct {
	net.Conn
	Info *TrackerInfo
}

func NewTCPTracker(conn net.Conn, ctx Context) *tcpTracker {
	id := uuid.New()
	tracker := &tcpTracker{
		Conn: conn,
		Info: &TrackerInfo{
			id:      id,
			start:   time.Now(),
			Context: ctx,
		},
	}
	DefaultManager.Add(tracker)
	return tracker
}

func (tracker *tcpTracker) Read(b []byte) (n int, err error) {
	n, err = tracker.Conn.Read(b)
	// current connection tracker total upload bytes
	tracker.Info.Context.uploadTotal.Add(int64(n))
	// global manager all connections trackers total upload bytes
	DefaultManager.uploadTotal.Add(int64(n))
	// global manager upload bytes per second
	DefaultManager.uploadPerSecDelta.Add(int64(n))
	return
}

func (tracker *tcpTracker) Write(b []byte) (n int, err error) {
	n, err = tracker.Conn.Write(b)
	tracker.Info.Context.downloadTotal.Add(int64(n))
	DefaultManager.downloadTotal.Add(int64(n))
	DefaultManager.downloadPerSecDelta.Add(int64(n))
	return
}

func (tracker *tcpTracker) Close() error {
	DefaultManager.Remove(tracker)
	return tracker.Conn.Close()
}

func (tracker *tcpTracker) TrackerInfo() *TrackerInfo {
	return tracker.Info
}

func (tracker *tcpTracker) ID() uuid.UUID {
	return tracker.Info.id
}

type udpTracker struct {
	net.PacketConn
	Info *TrackerInfo
}

func NewUDPTracker(conn net.PacketConn, ctx Context) *udpTracker {
	id := uuid.New()
	tracker := &udpTracker{
		PacketConn: conn,
		Info: &TrackerInfo{
			id:      id,
			start:   time.Now(),
			Context: ctx,
		},
	}
	DefaultManager.Add(tracker)
	return tracker
}

func (tracker *udpTracker) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, addr, err = tracker.PacketConn.ReadFrom(b)
	tracker.Info.Context.uploadTotal.Add(int64(n))
	DefaultManager.uploadTotal.Add(int64(n))
	DefaultManager.uploadPerSecDelta.Add(int64(n))
	return
}

func (tracker *udpTracker) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	n, err = tracker.PacketConn.WriteTo(b, addr)
	tracker.Info.Context.downloadTotal.Add(int64(n))
	DefaultManager.uploadPerSecDelta.Add(int64(n))
	DefaultManager.uploadTotal.Add(int64(n))
	return
}

func (tracker *udpTracker) Close() error {
	DefaultManager.Remove(tracker)
	return tracker.PacketConn.Close()
}

func (tracker *udpTracker) TrackerInfo() *TrackerInfo {
	return tracker.Info
}

func (tracker *udpTracker) ID() uuid.UUID {
	return tracker.Info.id
}
