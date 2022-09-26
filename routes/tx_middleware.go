package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/logger"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type codeCapturedResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (this codeCapturedResponseWriter) drainQueueUnlessError(ctx context.Context) {
	if this.statusCode >= 200 && this.statusCode < 400 {
		if err := config.QUEUE.Drain(ctx); err != nil {
			config.ReportError(fmt.Errorf("%w: draining buffered queue items", err), ctx)
			msg := fmt.Sprintf("draining buffered queue items: %+v", err)
			logger.Log(ctx, logger.Error, msg)
		}
	}
}

func NewCodeCapturedResponseWriter(w http.ResponseWriter) *codeCapturedResponseWriter {
	// WriteHeader(int) is not called if our response implicitly returns 200 OK, so
	// we default to that status code.
	return &codeCapturedResponseWriter{w, http.StatusOK}
}

func (this *codeCapturedResponseWriter) WriteHeader(code int) {
	this.statusCode = code
	this.ResponseWriter.WriteHeader(code)
}

func init() {
	taskMatcher = regexp.MustCompile(`webhooks\/tasks`)
}

var taskMatcher *regexp.Regexp

func txMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method == "GET" {
			ctx = context.WithValue(ctx, "tx", config.Db)
			ctx = context.WithValue(ctx, "parallelizable", true)
			h.ServeHTTP(w, r.WithContext(ctx))
		} else {
			codeWrapper := NewCodeCapturedResponseWriter(w)

			ctx = config.QUEUE.PrepareContext(ctx)

			if taskMatcher.MatchString(r.URL.Path) {
				ctx = context.WithValue(ctx, "tx", config.Db)
				ctx = context.WithValue(ctx, "parallelizable", true)
				h.ServeHTTP(w, r.WithContext(ctx))

				codeWrapper.drainQueueUnlessError(ctx)
				return
			}

			var tx *sql.Tx
			var err error

			if strings.Contains(r.URL.Path, "bookings") {
				opts := sql.TxOptions{
					Isolation: sql.LevelReadUncommitted,
				}
				tx, err = config.Db.BeginTx(ctx, &opts)
			} else {
				tx, err = config.Db.BeginTx(ctx, nil)
			}
			if err != nil {
				errRes(w, r, 500, "Cannot begin transaction", err)
				return
			}

			if c, err := r.Cookie("doubleboiler-user"); err == nil {
				cookieValue := make(map[string]string)
				if err := secureCookie.Decode("doubleboiler-user", c.Value, &cookieValue); err == nil {
					tx.ExecContext(ctx, "SET application_name = '"+cookieValue["Id"]+"'")
				} else {
					tx.ExecContext(ctx, "SET application_name = 'system_user'")
				}
			}

			ctx = context.WithValue(ctx, "tx", tx)
			ctx = context.WithValue(ctx, "parallelisable", false)

			h.ServeHTTP(codeWrapper, r.WithContext(ctx))

			if err := tx.Commit(); err != nil {
				if err != sql.ErrTxDone {
					logger.Log(r.Context(), logger.Error, fmt.Sprintf("committing transaction: %+v", err))
				}
			}

			ctx = context.WithValue(ctx, "tx", config.Db)

			codeWrapper.drainQueueUnlessError(ctx)
		}
	})
}
