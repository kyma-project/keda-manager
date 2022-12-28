package sha256

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var DefaultWriterSumerBuilder = WriterSumerBuilder(New)

type WriterSumer interface {
	io.Writer
	Sum(b []byte) []byte
}

func New() WriterSumer {
	return sha256.New()
}

type WriterSumerBuilder func() WriterSumer

func (w *WriterSumerBuilder) CalculateSHA256(obj unstructured.Unstructured) (string, error) {
	sha := DefaultWriterSumerBuilder()
	str := fmt.Sprintf("%s:%s:%s",
		obj.GetKind(),
		obj.GetObjectKind().GroupVersionKind().Group,
		obj.GetObjectKind().GroupVersionKind().Version)

	if _, err := sha.Write([]byte(str)); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(sha.Sum(nil)), nil
}
