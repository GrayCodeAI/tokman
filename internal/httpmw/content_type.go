package httpmw

import (
	"net/http"
	"strings"
)

// RequireContentType middleware validates Content-Type on write methods
func RequireContentType(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				ct := r.Header.Get("Content-Type")
				if !strings.HasPrefix(ct, contentType) {
					http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
