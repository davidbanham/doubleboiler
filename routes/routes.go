package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/models"
	m "doubleboiler/models"
	"log"
	"net/http"
	"os"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	fsnotify "gopkg.in/fsnotify.v1"
)

var queue kewpie.Kewpie

var r = mux.NewRouter()

func Init() (h http.Handler) {
	if err := queue.Connect(config.KEWPIE_BACKEND, []string{
		config.SEND_EMAIL_QUEUE_NAME,
	}, config.Db); err != nil {
		log.Fatal("ERROR", err)
	}

	h = csrfMiddleware(r)
	h = formParsingMiddleware(h)
	h = orgMiddleware(h)
	h = pathMiddleware(h)
	h = loginMiddleware(h)
	h = userMiddleware(h)
	h = txMiddleware(h)
	h = authFreeMiddleware(h)
	h = recoverWrap(h)
	h = handlers.LoggingHandler(os.Stdout, h)

	r.Path("/welcome").
		Methods("GET").
		HandlerFunc(serveWelcome)

	r.Path("/").
		Methods("GET").
		HandlerFunc(serveIndex)

	r.Path("/pricing").
		Methods("GET").
		HandlerFunc(servePricing)

	r.Path("/features/{name}").
		Methods("GET").
		HandlerFunc(serveFeatureMarketingPage)

	if config.STAGE != "production" {
		r.Path("/change-watcher").
			Methods("GET").
			HandlerFunc(serveChangeWatcher)
	}

	r.Path("/privacy").
		Methods("GET").
		HandlerFunc(servePrivacy)

	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./assets/")))

	return
}

func servePrivacy(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/privacy_collection_statement.pdf", http.StatusFound)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	err := Tmpl.ExecuteTemplate(w, "index.html", marketingPageData{
		Context: r.Context(),
	})
	if err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

func servePricing(w http.ResponseWriter, r *http.Request) {
	err := Tmpl.ExecuteTemplate(w, "pricing.html", marketingPageData{
		Context: r.Context(),
	})
	if err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

func serveFeatureMarketingPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := Tmpl.ExecuteTemplate(w, "feature_"+vars["name"]+".html", marketingPageData{
		Context: r.Context(),
	})
	if err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

type welcomePageData struct {
	Organisations models.Organisations
	Context       context.Context
}

func serveWelcome(w http.ResponseWriter, r *http.Request) {
	err := Tmpl.ExecuteTemplate(w, "welcome.html", welcomePageData{
		Context: r.Context(),
	})
	if err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

var upgrader = websocket.Upgrader{}

func serveChangeWatcher(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()

	done := make(chan bool)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	defer watcher.Close()

	go func() {
		select {
		case _ = <-watcher.Events:
			c.WriteMessage(1, []byte("reload"))
		}
	}()

	if err := watcher.Add("assets/css/"); err != nil {
		panic(err)
	}
	if err := watcher.Add("assets/js/"); err != nil {
		panic(err)
	}

	<-done
}

type marketingPageData struct {
	Context context.Context
}

func isLoggedIn(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	unconv := ctx.Value("user")

	if unconv != nil {
		return true
	}
	return false
}

func orgFromContext(ctx context.Context, orgId string) models.Organisation {
	unconv := ctx.Value("organisations")
	if unconv == nil {
		return models.Organisation{}
	}

	organisations := unconv.(models.Organisations)

	for _, org := range organisations.Data {
		if org.ID == orgId {
			return org
		}
	}

	return models.Organisation{}
}

func targetOrgIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	unconv := ctx.Value("target_org")

	if unconv == nil {
		return ""
	}

	return unconv.(string)
}

func activeOrgFromContext(ctx context.Context) models.Organisation {
	if ctx == nil {
		return m.Organisation{}
	}

	targetOrg := targetOrgIDFromContext(ctx)

	orgs := orgsFromContext(ctx)

	for _, org := range orgs.Data {
		if org.ID == targetOrg {
			return org
		}
	}

	return m.Organisation{}
}

func orgsFromContext(ctx context.Context) models.Organisations {
	if ctx == nil {
		return m.Organisations{}
	}

	unconv := ctx.Value("organisations")

	if unconv == nil {
		return m.Organisations{}
	}

	orgs := unconv.(m.Organisations)

	return orgs
}
