package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/logger"
	"doubleboiler/models"
	"doubleboiler/views"
	"log"
	"net/http"
	"os"

	_ "time/tzdata"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	fsnotify "gopkg.in/fsnotify.v1"
)

var r = mux.NewRouter()
var Tmpl views.Templater

func Init() (h http.Handler) {
	t, err := views.Tmpl(templateFuncMap)
	if err != nil {
		log.Fatal(err)
	}

	Tmpl = t

	h = csrfMiddleware(r)
	h = formParsingMiddleware(h)
	h = orgMiddleware(h)
	h = pathMiddleware(h)
	h = loginMiddleware(h)
	h = userMiddleware(h)
	h = txMiddleware(h)
	h = traceMiddleware(h)
	h = recoverWrap(h)
	h = handlers.LoggingHandler(os.Stdout, h)

	r.Path("/dashboard").
		Methods("GET").
		HandlerFunc(serveDashboard)

	r.Path("/").
		Methods("GET").
		HandlerFunc(serveIndex)

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
	if err := Tmpl.ExecuteTemplate(w, "index.html", basePageData{
		Context: r.Context(),
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

type basePageData struct {
	PageTitle string
	Context   context.Context
	Next      string
}

func (pd basePageData) Title() string {
	if pd.PageTitle == "" {
		return config.NAME
	}
	return pd.PageTitle
}

type dashboardPageData struct {
	basePageData
	Organisations models.Organisations
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	if err := Tmpl.ExecuteTemplate(w, "dashboard.html", dashboardPageData{
		basePageData: basePageData{
			Context:   r.Context(),
			PageTitle: "DoubleBoiler - Dashboard",
		},
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

var upgrader = websocket.Upgrader{}

func serveChangeWatcher(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log(r.Context(), logger.Debug, "upgrade:", err)
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
		return models.Organisation{}
	}

	targetOrg := targetOrgIDFromContext(ctx)

	orgs := orgsFromContext(ctx)

	for _, org := range orgs.Data {
		if org.ID == targetOrg {
			return org
		}
	}

	return models.Organisation{}
}

func orgsFromContext(ctx context.Context) models.Organisations {
	if ctx == nil {
		return models.Organisations{}
	}

	unconv := ctx.Value("organisations")

	if unconv == nil {
		return models.Organisations{}
	}

	orgs := unconv.(models.Organisations)

	return orgs
}
