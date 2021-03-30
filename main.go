package main

import (
	"context"
	"crypto/tls"
	"doubleboiler/config"
	"doubleboiler/logger"
	"doubleboiler/routes"
	"doubleboiler/workers"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	certCache "github.com/davidbanham/certcache"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	addr := ":" + config.PORT

	app := routes.Init()

	if config.START_WORKERS {
		fmt.Println("INFO Starting workers")
		workers.Init()
	}

	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.MAINTENANCE_MODE {
			maintHandler.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/health" {
			healthHandler.ServeHTTP(w, r)
		} else {
			app.ServeHTTP(w, r)
		}
	})

	s := &http.Server{
		Handler: router,
		Addr:    addr,
	}

	if config.TLS {
		if config.AUTOCERT {
			sc, err := certCache.Init("certs-doubleboiler", config.GOOGLE_PROJECT_ID)
			if err != nil {
				log.Fatalf("ERROR Error instantitating storage cache: %+v", err)
			}

			certMgr := autocert.Manager{
				Cache: sc,
				Prompt: func(_ string) bool {
					return true
				},
				HostPolicy: func(_ context.Context, host string) error {
					if strings.Contains(host, config.DOMAIN) {
						return nil
					}
					return fmt.Errorf("Domain not valid")
				},
			}

			httpsRedirector := certMgr.HTTPHandler(nil)

			s = &http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/health" {
						healthHandler.ServeHTTP(w, r)
					} else {
						httpsRedirector.ServeHTTP(w, r)
					}
				}),
				Addr: addr,
			}

			s2 := &http.Server{
				Handler:   router,
				Addr:      ":https",
				TLSConfig: &tls.Config{GetCertificate: certMgr.GetCertificate},
			}

			logger.Log(context.Background(), logger.Info, "Listening on 443 with TLS")
			go func() {
				log.Fatalf("ERROR %+v", s2.ListenAndServeTLS("", ""))
			}()

			logger.Log(context.Background(), logger.Info, fmt.Sprintf("Listening on: %s", addr))
			log.Fatalf("ERROR %+v", s.ListenAndServe())
		} else {
			logger.Log(context.Background(), logger.Info, "Starting self signed server on", os.Getenv("PORT"))

			log.Fatalf("ERROR %+v", s.ListenAndServeTLS("./local_dev/server.crt", "./local_dev/server.key"))
		}
	} else {
		logger.Log(context.Background(), logger.Info, "Starting plain http server on", os.Getenv("PORT"))

		log.Fatalf("ERROR %+v", s.ListenAndServe())
	}
}

var healthHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if config.MAINTENANCE_MODE {
		w.Write([]byte("ok"))
		return
	}

	err := config.Db.Ping()
	if err != nil {
		logger.Log(context.Background(), logger.Error, fmt.Sprintf("db connection error: %+v \n", err))
		w.WriteHeader(500)
		w.Write([]byte("db connection error"))
		return
	}
	w.Write([]byte("ok"))
})

type maintPageData struct {
	Context context.Context
}

var maintHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	routes.Tmpl.ExecuteTemplate(w, "maint.html", maintPageData{
		Context: r.Context(),
	})
})
