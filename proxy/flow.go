package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/andybalholm/brotli"
	"github.com/golang/snappy"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

type (
	HTTPFlow struct {
		Request  *http.Request
		Response *http.Response
	}
	WSFlow struct {
		Direction  WSDirection
		Request    *http.Request
		MsgType    int
		FramedData []byte
	}
	Flow struct {
		FlowID    uint64
		Timestamp int64
		HTTP      *HTTPFlow
		WS        *WSFlow
	}
)

type (
	HTTPRequestView struct {
		Method string
		Uri    string
		Proto  string
		Header http.Header
		Body   []byte
	}

	HTTPResponseView struct {
		Proto      string
		StatusCode int
		Header     http.Header
		Body       []byte
	}

	HTTPView struct {
		Request  *HTTPRequestView
		Response *HTTPResponseView
	}
)

func (v *HTTPRequestView) Encode() []byte {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s %s %s\r\n", v.Method, v.Uri, v.Proto))
	v.Header.Write(&buf)
	io.WriteString(&buf, "\r\n")
	buf.ReadFrom(bytes.NewReader(v.Body))
	return buf.Bytes()
}

func (v *HTTPResponseView) Encode() []byte {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s %d %s\r\n", v.Proto, v.StatusCode, http.StatusText(v.StatusCode)))
	v.Header.Write(&buf)
	io.WriteString(&buf, "\r\n")
	buf.ReadFrom(bytes.NewReader(v.Body))
	return buf.Bytes()
}

func (f *HTTPFlow) DumpHTTPView() (*HTTPView, error) {
	reqView, err := f.DumpHTTPRequestView()
	if err != nil {
		return nil, err
	}
	rspView, err := f.DumpHTTPResponseView()
	if err != nil {
		return nil, err
	}
	return &HTTPView{
		Request:  reqView,
		Response: rspView,
	}, nil
}

func (f *HTTPFlow) DumpHTTPRequestView() (*HTTPRequestView, error) {
	req := f.Request
	var err error
	var view = &HTTPRequestView{
		Method: req.Method,
		Proto:  req.Proto,
		Header: req.Header.Clone(),
	}
	reqURI := req.RequestURI
	if reqURI == "" {
		reqURI = req.URL.RequestURI()
	}
	view.Uri = reqURI

	save := req.Body
	if req.Body != nil {
		save, req.Body, err = decodeBody(req.Header.Get(HttpHeaderContentEncoding), req.Body)
		if err != nil {
			return nil, err
		}
	}
	if req.Body != nil {
		view.Body, _ = io.ReadAll(req.Body)
		if len(view.Body) > 0 {
			view.Header.Set(HttpHeaderContentLength, strconv.FormatInt(int64(len(view.Body)), 10))
		}
	}
	req.Body = save
	return view, nil
}

func (f *HTTPFlow) DumpHTTPResponseView() (*HTTPResponseView, error) {
	resp := f.Response
	var err error
	var view = &HTTPResponseView{
		Proto:      resp.Proto,
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
	}
	save := resp.Body
	if resp.Body != nil {
		save, resp.Body, err = decodeBody(resp.Header.Get(HttpHeaderContentEncoding), resp.Body)
		if err != nil {
			return nil, err
		}
	}
	if resp.Body != nil {
		view.Body, _ = io.ReadAll(resp.Body)
		if len(view.Body) > 0 {
			view.Header.Set(HttpHeaderContentLength, strconv.FormatInt(int64(len(view.Body)), 10))
		}
	}
	resp.Body = save
	return view, nil
}

func decodeBody(enc string, b io.ReadCloser) (r1, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		return http.NoBody, http.NoBody, nil
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}
	if err = b.Close(); err != nil {
		return nil, b, err
	}
	reader := bytes.NewReader(buf.Bytes())
	switch enc {
	case "br":
		b = io.NopCloser(brotli.NewReader(reader))
	case "snappy":
		b = io.NopCloser(snappy.NewReader(reader))
	case "deflate":
		b = flate.NewReader(reader)
	case "gzip":
		reader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, b, err
		}
		b = reader
	case "zstd":
		decoder, err := zstd.NewReader(reader)
		if err != nil {
			return nil, b, err
		}
		b = decoder.IOReadCloser()
	default:
		b = io.NopCloser(reader)
	}
	// raw body buf, decompression body buf
	return io.NopCloser(&buf), b, nil
}
