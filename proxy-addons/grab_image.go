package proxyaddons

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/josexy/mini-ss/util/logger"
)

type grabImage struct {
	outputDir string
}

func newGrabImage(outputDir string) *grabImage {
	return &grabImage{outputDir: outputDir}
}

func (g *grabImage) Response(ctx *Context) {
	contentType := ctx.HTTP.Response.Header.Get("Content-Type")
	// filter none-image content
	if !strings.HasPrefix(contentType, "image") {
		return
	}
	path := ctx.HTTP.Request.URL.Path
	logger.Logger.Errorf("addons[image]: response: %s, %s", contentType, filepath.Base(path))
	filename := filepath.Join(g.outputDir, filepath.Base(path))

	view, err := ctx.HTTP.DumpHTTPResponseView()
	if err != nil {
		return
	}
	os.WriteFile(filename, view.Body, 0644)
}
