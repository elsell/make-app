// Package content adapts the standard MIME sniffer to the blob inspection port.
package content

import "net/http"

type Inspector struct{}

func (Inspector) ContentType(data []byte) string { return http.DetectContentType(data) }
