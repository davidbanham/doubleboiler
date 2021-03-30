package routes

import (
	"context"
	"net/http"
	"strings"

	uuid "github.com/satori/go.uuid"
)

func traceMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceHeader := r.Header.Get("X-Cloud-Trace-Context")
		if traceHeader != "" {
			//X-Cloud-Trace-Context: <trace-id>/<span-id>;<trace-options> (requests only)
			traceBits := strings.Split(traceHeader, "/")
			if len(traceBits) > 0 {
				traceID := traceBits[0]
				ctx := context.WithValue(r.Context(), "trace", traceID)
				ctx = context.WithValue(ctx, "requestID", traceID)
				h.ServeHTTP(w, r.WithContext(ctx))
			}
		} else {
			requestID := uuid.NewV4().String()
			ctx := context.WithValue(r.Context(), "requestID", requestID)
			h.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
