package ping

import (
	"errors"
	"net"
	"strconv"
	"time"
)

var ErrTimeout = errors.New("timeout")

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
			return 0, ErrTimeout
		}
		conn.Close()
		rtts = append(rtts, time.Since(start))
	}
	return calcAvgRtt(rtts)
}

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
