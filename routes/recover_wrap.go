package routes

import (
	"doubleboiler/config"
	"doubleboiler/logger"
	"errors"
	"net/http"
	"runtime/debug"
)

func recoverWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer func() {
			rec := recover()
			if rec != nil {
				switch t := rec.(type) {
				case string:
					err = errors.New(t)
				case error:
					err = t
				default:
					err = errors.New("Unknown error")
				}
				debug.PrintStack()
				logger.Log(r.Context(), logger.Error, "UNHANDLED PANIC", err)
				errText := ""
				if config.RENDER_ERRORS {
					errText = err.Error()
				}
				errRes(w, r, 500, errText, err)
			}
		}()
		h.ServeHTTP(w, r)
	})
}
