package ping

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/josexy/mini-ss/util"
	"golang.org/x/net/proxy"
)

const (
	socksMode = "socks"
	httpMode  = "http"
	tunMode   = "tun"
)

var (
	downloadFile = "http://cachefly.cachefly.net/10mb.test"

	errStopped   = errors.New("speed test stopped")
	errCompleted = errors.New("speed test completed")
)

type (
	proxyFunc  func(*http.Request) (*url.URL, error)
	dialerFunc func(network, address string) (net.Conn, error)
)

type calcWriter struct {
	once  sync.Once
	start time.Time
	state
}

type state struct {
	current   int64
	total     int64
	completed bool
	rtt       time.Duration
	rate      string
}

func (w *calcWriter) Write(data []byte) (int, error) {
	w.once.Do(func() {
		w.start = time.Now()
	})
	n := len(data)
	w.current += int64(n)

	if w.current == w.total {
		rtt := time.Since(w.start)
		// average download rate = total size / total cost time
		w.rate = util.FormatSpeedRate(w.total, int64(rtt.Seconds()))
		w.completed = true
		w.rtt = rtt
		return 0, errCompleted
	}
	return n, nil
}

type SpeedTest struct {
	mode       string
	httpProxy  string
	socksProxy string
	result     state
	running    uint32
	err        chan error
	stop       chan struct{}
	auth       *url.Userinfo
}

func NewSpeedTest() *SpeedTest {
	return &SpeedTest{}
}

// SetTestProxy reset context, must be called before Start()
func (st *SpeedTest) SetTestProxy(mode string, httpProxy, socksProxy string) {
	st.mode = mode
	st.httpProxy = httpProxy
	st.socksProxy = socksProxy
	st.result = state{}
	atomic.StoreUint32(&st.running, 0)
	st.err = make(chan error)
	st.stop = make(chan struct{})
	st.auth = nil
}

func (st *SpeedTest) SetAuth(auth string) {
	if auth == "" {
		return
	}
	u, p, _ := strings.Cut(auth, ":")
	st.auth = url.UserPassword(u, p)
}

func (st *SpeedTest) Status() (rrt time.Duration, rate string) {
	return st.result.rtt, st.result.rate
}

func (st *SpeedTest) Start(timeout time.Duration) error {
	if atomic.LoadUint32(&st.running) != 0 {
		return errors.New("speed test is running")
	}

	atomic.StoreUint32(&st.running, 1)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// run completed
	defer func() { atomic.StoreUint32(&st.running, 0) }()

	go st.run()

	select {
	case <-st.stop: // stopped by user
		return errStopped
	case err := <-st.err: // error occurred
		return err
	case <-ctx.Done(): // timeout or cancel
		return errTimeout
	}
}

func (st *SpeedTest) Stop() {
	if atomic.LoadUint32(&st.running) == 0 {
		return
	}
	atomic.StoreUint32(&st.running, 0)
	st.stop <- struct{}{}
}

func (st *SpeedTest) run() {
	baseDialer := &net.Dialer{Timeout: 4 * time.Second}

	var httpProxyFunc proxyFunc
	var dialerFunc dialerFunc = baseDialer.Dial

	switch st.mode {
	case httpMode:
		httpProxyFunc = func(r *http.Request) (*url.URL, error) {
			// http proxy start with http://
			return &url.URL{
				Scheme: "http",
				Host:   st.httpProxy,
				User:   st.auth,
			}, nil
		}
	case socksMode:
		var auth *proxy.Auth
		if st.auth != nil {
			p, _ := st.auth.Password()
			auth = &proxy.Auth{User: st.auth.Username(), Password: p}
		}
		dialer, err := proxy.SOCKS5("tcp", st.socksProxy, auth, baseDialer)
		if err != nil {
			st.err <- err
			return
		}
		dialerFunc = dialer.Dial
	case tunMode:
		// default dialer
	}
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:                 httpProxyFunc,
			DialContext:           func(ctx context.Context, network, addr string) (net.Conn, error) { return dialerFunc(network, addr) },
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	res, err := client.Get(downloadFile)
	if err != nil {
		st.err <- err
		return
	}

	cw := &calcWriter{state: state{total: res.ContentLength}}
	io.Copy(cw, res.Body)
	res.Body.Close()

	if !cw.completed {
		st.err <- fmt.Errorf("download failed, cur: %d, total: %d", cw.current, cw.total)
		return
	}
	st.result = cw.state
	st.err <- nil
}

func SetSpeedTestUrl(url string) {
	downloadFile = url
}

func TunSpeedTest(timeout time.Duration) (time.Duration, string, error) {
	st := NewSpeedTest()
	st.SetTestProxy(tunMode, "", "")
	err := st.Start(timeout)
	return st.result.rtt, st.result.rate, err
}

func SocksSpeedTest(proxy string, timeout time.Duration) (time.Duration, string, error) {
	st := NewSpeedTest()
	st.SetTestProxy(socksMode, "", proxy)
	err := st.Start(timeout)
	return st.result.rtt, st.result.rate, err
}

func HttpSpeedTest(proxy string, timeout time.Duration) (time.Duration, string, error) {
	st := NewSpeedTest()
	st.SetTestProxy(httpMode, proxy, "")
	err := st.Start(timeout)
	return st.result.rtt, st.result.rate, err
}
