package routes

import (
	"doubleboiler/config"
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
)

func txMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method == "GET" {
			ctx = context.WithValue(ctx, "tx", config.Db)
			h.ServeHTTP(w, r.WithContext(ctx))
		} else {
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
				}
			}

			ctx = context.WithValue(ctx, "tx", tx)

			h.ServeHTTP(w, r.WithContext(ctx))

			if err := tx.Commit(); err != nil {
				if err != sql.ErrTxDone {
					log.Printf("ERROR committing transaction: %+v", err)
				}
			}
		}
	})
}
