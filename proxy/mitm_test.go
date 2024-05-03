package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	caCrt = "../certs/ca.crt"
	caKey = "../certs/ca.key"
)

func parseHostPort(req *http.Request) (host, port string) {
	var target string
	if req.Method != http.MethodConnect {
		target = req.Host
	} else {
		target = req.RequestURI
	}
	host, port, err := net.SplitHostPort(target)
	if err != nil || port == "" {
		host = target
		if req.Method != http.MethodConnect {
			port = "80"
		}
		// ipv6
		if host[0] == '[' {
			host = target[1 : len(host)-1]
		}
	}
	return
}

func startSimpleHttpServer(addr string, handler http.Handler) *http.Server {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	return server
}

func testHTTPRequest(addr string, isHttps bool) {
	caCrtPem, err := os.ReadFile(caCrt)
	if err != nil {
		panic(err)
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCrtPem)

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				return url.Parse("http://" + addr)
			},
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}
	data := `{"data":"hello world"}`
	scheme := "http"
	if isHttps {
		scheme = "https"
	}
	rsp, err := client.Post(scheme+"://www.httpbin.org/post", "application/json", strings.NewReader(data))
	if err != nil {
		panic(err)
	}
	defer rsp.Body.Close()
	res, err := io.ReadAll(rsp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("status code: %d, headers: %+v, body: %s\n", rsp.StatusCode, rsp.Header, res)
}

func TestMitmHandlerForHTTPTraffic(t *testing.T) {
	handler, err := NewMitmHandler(MimtOption{
		Enable:  true,
		CaPath:  caCrt,
		KeyPath: caKey,
	})
	handler.SetMutableHTTPInterceptor(func(req *http.Request, invoker HTTPDelegatedInvoker) (*http.Response, error) {
		data, _ := httputil.DumpRequest(req, true)
		t.Log(string(data))

		// Modify the request
		req.Header.Add("TestReqKey", "TestReqValue")
		// Get the invoked response
		rsp, err := invoker.Invoke(req)
		if err == nil {
			rsp.Header.Add("TestRspKey", "TestRspValue")
		}

		data, _ = httputil.DumpResponse(rsp, true)
		t.Log(string(data))

		// Modify the response
		rsp.Body.Close()
		rspData := strings.NewReader("hello world")
		rsp.ContentLength = rspData.Size()
		rsp.Body = io.NopCloser(rspData)
		// Return the modified response
		return rsp, err
	})

	handler.SetImmutableHTTPInterceptor(func(req *http.Request, rsp *http.Response) {
		reqData, _ := io.ReadAll(req.Body)
		rspData, _ := io.ReadAll(rsp.Body)
		t.Logf("======== url: %s, status code: %d, req data: %s, rsp data: %s", req.URL.String(), rsp.StatusCode, string(reqData), string(rspData))
	})

	if err != nil {
		t.Fatal(err)
	}
	// CLI test: curl --proxy http://127.0.0.1:10087 "http://www.httpbin.org/get?key=value" -v
	//		     curl --proxy http://127.0.0.1:10087 "https://www.httpbin.org/get?key=value" -v --cacert certs/ca.crt
	addr := "127.0.0.1:10087"
	server := startSimpleHttpServer(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		host, port := parseHostPort(r)
		reqCtx := ReqContext{
			ConnMethod: true,
			Host:       host,
			Port:       port,
			Addr:       net.JoinHostPort(host, port),
		}
		t.Log("target addr:", reqCtx.Addr)
		if r.Method != http.MethodConnect {
			reqCtx.ConnMethod = false
			reqCtx.Request = r
		} else {
			conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		}
		err = handler.HandleMIMT(context.WithValue(context.Background(), ReqCtxKey, reqCtx), conn)
		if err != nil {
			t.Log(err)
		}
	}))
	time.Sleep(time.Second * 1)
	go testHTTPRequest(addr, false)
	go testHTTPRequest(addr, true)
	time.Sleep(time.Second * 4)
	server.Close()
}
