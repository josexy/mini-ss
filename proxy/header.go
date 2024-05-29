package proxy

import "net/http"

const (
	HttpHeaderContentType            = "Content-Type"
	HttpHeaderConnection             = "Connection"
	HttpHeaderKeepAlive              = "Keep-Alive"
	HttpHeaderProxyAuthenticate      = "Proxy-Authenticate"
	HttpHeaderProxyAuthorization     = "Proxy-Authorization"
	HttpHeaderProxyConnection        = "Proxy-Connection"
	HttpHeaderProxyAgent             = "Proxy-Agent"
	HttpHeaderTe                     = "Te"
	HttpHeaderTrailers               = "Trailers"
	HttpHeaderTransferEncoding       = "Transfer-Encoding"
	HttpHeaderUpgrade                = "Upgrade"
	HttpHeaderSecWebsocketKey        = "Sec-Websocket-Key"
	HttpHeaderSecWebsocketVersion    = "Sec-Websocket-Version"
	HttpHeaderSecWebsocketExtensions = "Sec-Websocket-Extensions"
	HttpHeaderContentEncoding        = "Content-Encoding"
	HttpHeaderContentLength          = "Content-Length"
)

var (
	// Hop-by-hop headers. These are removed when sent to the backend.
	// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
	hopByHopHeaders = []string{
		HttpHeaderConnection,
		HttpHeaderKeepAlive,
		HttpHeaderProxyAuthenticate,
		HttpHeaderProxyAuthorization,
		HttpHeaderTe,
		HttpHeaderTrailers,
		HttpHeaderTransferEncoding,
		HttpHeaderUpgrade,
		HttpHeaderProxyConnection,
	}
)

func RemoveHopByHopRequestHeaders(header http.Header) {
	for _, h := range hopByHopHeaders {
		header.Del(h)
	}
}

func RemoveWebsocketRequestHeaders(header http.Header) {
	header.Del(HttpHeaderUpgrade)
	header.Del(HttpHeaderConnection)
	header.Del(HttpHeaderSecWebsocketKey)
	header.Del(HttpHeaderSecWebsocketVersion)
	header.Del(HttpHeaderSecWebsocketExtensions)
}
