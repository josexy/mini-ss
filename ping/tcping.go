package ping

import (
	"net"
	"time"
)

func TCPing(dst string, count int) (time.Duration, error) {
	if count <= 0 {
		return 0, nil
	}
	_, _, err := net.SplitHostPort(dst)
	if err != nil {
		return 0, err
	}
	var rtts []time.Duration
	for i := 0; i < count; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp4", dst, 2*time.Second)
		if err != nil {
			return 0, errTimeout
		}
		conn.Close()
		rtts = append(rtts, time.Since(start))
	}
	return calcAvgRtt(rtts)
}

func TCPingList(dstList []string, count int) ([]time.Duration, []error) {
	var res []time.Duration
	var errs []error
	for _, dst := range dstList {
		rtt, err := TCPing(dst, count)
		res = append(res, rtt)
		errs = append(errs, err)
	}
	return res, errs
}
