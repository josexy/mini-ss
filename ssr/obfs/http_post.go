package obfs

func newHTTPPost(b *Base) Obfs {
	return &httpObfs{Base: b, post: true}
}
