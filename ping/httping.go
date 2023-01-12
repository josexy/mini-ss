package ping

import (
	"net"
	"net/http"
	"time"
)

// HTTPing use the HTTP GET method to connect to the url
// such as http://www.gstatic.com/generate_204
func HTTPing(url string, count int) (time.Duration, error) {
	if count <= 0 {
		return 0, nil
	}
	var rtts []time.Duration
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	d := &net.Dialer{Timeout: 2 * time.Second}
	transport := http.Transport{
		DialContext:         d.DialContext,
		IdleConnTimeout:     3 * time.Second,
		TLSHandshakeTimeout: 3 * time.Second,
	}
	for i := 0; i < count; i++ {
		start := time.Now()
		resp, err := transport.RoundTrip(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		rtts = append(rtts, time.Since(start))
	}
	return calcAvgRtt(rtts)
}

func HTTPingList(urlList []string, count int) ([]time.Duration, []error) {
	var res []time.Duration
	var errs []error
	for _, url := range urlList {
		rtt, err := HTTPing(url, count)
		res = append(res, rtt)
		errs = append(errs, err)
	}
	return res, errs
}
