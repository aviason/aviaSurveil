package documents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	maximumPDFResponseSize = 16 << 20
	nativeRendererName     = "avia-native-gopdf"
	nativeRendererVersion  = "gopdf@v0.38.0"
	nativeLayoutVersion    = "a4-canonical-report-v1"
	nativeTemplateVersion  = "report-content-v1"
	nativeModuleChecksum   = "github.com/signintech/gopdf@v0.38.0"
)

type RenderedArtifact struct {
	FileName     string
	MediaType    string
	Body         []byte
	RendererHash string
	TemplateHash string
	SourceHash   string
}

type Renderer interface {
	Render(context.Context, RenderSnapshot) (RenderedArtifact, error)
}

type NativeProvenance struct {
	RendererHash   string
	TemplateHash   string
	FontHash       string
	Renderer       string
	ModuleChecksum string
	Layout         string
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validSHA256(value string) bool {
	if len(value) != len("sha256:")+64 || value[:len("sha256:")] != "sha256:" {
		return false
	}
	_, err := hex.DecodeString(value[len("sha256:"):])
	return err == nil
}

func rendererProvenance(fontHash string) (rendererHash, templateHash string) {
	rendererHash = digest([]byte(fmt.Sprintf("%s|%s|%s|%s|%s", nativeRendererName,
		nativeRendererVersion, nativeLayoutVersion, nativeModuleChecksum, fontHash)))
	templateHash = digest([]byte(fmt.Sprintf("%s|%s|%s", nativeTemplateVersion,
		nativeLayoutVersion, ReportContentSchema)))
	return rendererHash, templateHash
}
