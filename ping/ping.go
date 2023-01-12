package ping

import (
	"errors"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/josexy/mini-ss/util"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var errTimeout = errors.New("timeout")

const (
	timeSliceLength     = 8
	readDeadlineTimeout = 200 * time.Millisecond
)

var pingData = []byte("0123456789!@#$%^&*()_+")

// ping need root permission
type ping struct {
	conn net.PacketConn
	seq  int
	buf  []byte
	rtts []time.Duration
}

func newPing() (*ping, error) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, err
	}
	return &ping{
		conn: conn,
		buf:  make([]byte, 1500),
	}, nil
}

// timestamp bytes convert time.Time to slice bytes in big-endian
func timestamp2bytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

// calcAvgRtt calc average rtt from reply
func calcAvgRtt(rtts []time.Duration) (rtt time.Duration, err error) {
	var avgRtt float64
	for _, rtt := range rtts {
		avgRtt += float64(rtt.Microseconds())
	}
	avgRtt /= float64(len(rtts))
	avgRtt /= 1e3
	if rtt, err = time.ParseDuration(strconv.FormatFloat(avgRtt, 'f', 2, 64) + "ms"); err != nil {
		return 0, errors.New("ping failed")
	}
	return rtt, nil
}

// bytes timestamp convert slice bytes to time.Time in big-endian
func bytes2timestamp(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}

func (p *ping) ping(dst string, count int) (rtt time.Duration, err error) {
	if count <= 0 {
		return 0, nil
	}
	ip, err := net.ResolveIPAddr("ip", dst)
	if err != nil {
		return 0, err
	}
	p.rtts = p.rtts[:0]
	for i := 0; i < count; i++ {
		if err := p.sendIcmp(ip); err != nil {
			continue
		}
		if rtt, err := p.recvIcmp(); err != nil {
			continue
		} else {
			p.rtts = append(p.rtts, rtt)
		}
	}
	if len(p.rtts) == 0 {
		return 0, errTimeout
	}
	return calcAvgRtt(p.rtts)
}

func (p *ping) sendIcmp(addr net.Addr) error {
	req := &icmp.Message{
		Type:     ipv4.ICMPTypeEcho,
		Code:     0,
		Checksum: 0,
		Body: &icmp.Echo{
			ID:   os.Getgid(),
			Seq:  p.seq,
			Data: append(timestamp2bytes(time.Now()), pingData...),
		},
	}
	data, err := req.Marshal(nil)
	if err != nil {
		return err
	}
	_, err = p.conn.WriteTo(data, addr)
	if err != nil {
		return err
	}
	p.seq++
	return nil
}

// recvIcmp read icmp packet
func (p *ping) recvIcmp() (rtt time.Duration, err error) {
	var n int

	p.conn.SetReadDeadline(time.Now().Add(readDeadlineTimeout))
	n, _, err = p.conn.ReadFrom(p.buf)
	if err != nil {
		return
	}
	msg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), p.buf[:n])
	if err != nil {
		return
	}

	// bad reply icmp packet
	if msg.Type != ipv4.ICMPTypeEchoReply {
		err = errors.New("icmp reply echo bad")
		return
	}
	if reply, ok := msg.Body.(*icmp.Echo); ok {
		// read sender sent time
		t := bytes2timestamp(reply.Data[:timeSliceLength])
		rtt = time.Since(t)
	}
	return
}

func Ping(dst string, count int) (time.Duration, error) {
	host, _ := util.SplitHostPort(dst)
	// p, err := newPing()
	// if err != nil {
	// 	return 0, err
	// }
	// return p.ping(host, count)
	return PingWithoutRoot(host, count)
}

func PingList(dstList []string, count int) ([]time.Duration, []error) {
	var res []time.Duration
	var errs []error
	for _, dst := range dstList {
		rtt, err := Ping(dst, count)
		res = append(res, rtt)
		errs = append(errs, err)
	}
	return res, errs
}
