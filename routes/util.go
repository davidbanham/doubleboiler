package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/flashes"
	"doubleboiler/logger"
	"doubleboiler/models"
	"doubleboiler/util"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
)

var templateFuncMap = template.FuncMap{
	"humanDate": func(t time.Time) string {
		return t.Format("02 Jan 2006")
	},
	"dateonly": func(in time.Time) string {
		return in.Format("2006-01-02")
	},
	"breakLines": func(in string) []string {
		return strings.Split(in, "\n")
	},
	"contains": func(str []string, target string) bool {
		for _, s := range str {
			if s == target {
				return true
			}
		}
		return false
	},
	"firstFiveChars":       util.FirstFiveChars,
	"loggedIn":             isLoggedIn,
	"user":                 userFromContext,
	"orgsFromContext":      orgsFromContext,
	"flashes":              flashesFromContext,
	"searchQuery":          searchQueryFromContext,
	"activeOrgFromContext": activeOrgFromContext,
	"can": func(ctx context.Context, role string) bool {
		org := activeOrgFromContext(ctx)
		return can(ctx, org, role)
	},
	"csrf": func(ctx context.Context) string {
		unconv := ctx.Value("user")
		if unconv == nil {
			return ""
		}

		user := unconv.(models.User)
		return util.CalcToken(config.SECRET, 0, user.ID).String()
	},
	"isLocal": func() bool { return config.LOCAL },
	"logoLink": func(ctx context.Context) string {
		if !isLoggedIn(ctx) {
			return "/"
		}
		unconv := ctx.Value("url")
		if unconv == nil {
			return "/"
		}
		url := unconv.(*url.URL)
		if strings.Contains(url.Path, "/welcome") {
			return "/"
		}
		return "/welcome"
	},
	"subComponent": func(name string, data interface{}) (template.HTML, error) {
		return Tmpl.Component(name, data)
	},
	"searchableEntities": func(ctx context.Context, query models.SearchQuery) models.Searchables {
		org := activeOrgFromContext(ctx)
		roles := orgUserFromContext(ctx, org).Roles

		return models.SearchTargets.FilterByRole(roles, query)
	},
}

func checkFormInput(required []string, form url.Values, w http.ResponseWriter, r *http.Request) bool {
	for _, val := range required {
		if len(form[val]) < 1 {
			errRes(w, r, 400, "Invalid "+val, nil)
			return false
		}
		if form[val][0] == "" {
			errRes(w, r, 400, "Invalid "+val, nil)
			return false
		}
	}
	return true
}

type errorPageData struct {
	basePageData
	Message string
	Context context.Context
}

func errRes(w http.ResponseWriter, r *http.Request, code int, message string, err error) {
	sendErr := func() {
		if err != nil && err.Error() == "http2: stream closed" {
			return
		}
		reportableErr := err
		if err == nil {
			reportableErr = errors.New(strconv.Itoa(code) + " " + message)
		}
		if err != nil {
			config.ReportError(reportableErr)
		}

		if clientSafe, addendum := isClientSafe(err); clientSafe {
			message += " " + addendum
		}

		logger.Log(r.Context(), logger.Warning, fmt.Sprintf("Sending Error Response: %+v, %+v, %+v, %+v", code, message, r.URL.String(), err))
		if code == 500 {
			logger.Log(r.Context(), logger.Error, err)
			logger.Log(r.Context(), logger.Debug, string(debug.Stack()))
		}

		w.WriteHeader(code)
		if r.Header.Get("Accept") == "application/json" {
			w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, message)))
			return
		}

		ohshit := Tmpl.ExecuteTemplate(w, "error.html", errorPageData{
			Message: message,
			Context: r.Context(),
		})
		if ohshit != nil {
			w.Write([]byte("Error rendering the error template. Oh dear."))
			return
		}
	}

	tx := r.Context().Value("tx")
	switch v := tx.(type) {
	case *sql.Tx:
		rollbackErr := v.Rollback()
		if rollbackErr != nil {
			logger.Log(r.Context(), logger.Error, fmt.Sprintf("Error rolling back tx: %+v", rollbackErr))
		}
	default:
		//fmt.Printf("DEBUG no transaction on error\n")
	}
	sendErr()
}

func shortDur(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = s[:len(s)-2]
	}
	if strings.HasSuffix(s, "h0m") {
		s = s[:len(s)-2]
	}
	return s
}

func redirToDefaultOrg(w http.ResponseWriter, r *http.Request) {
	orgs := orgsFromContext(r.Context())
	if len(orgs.Data) < 1 {
		http.Redirect(w, r, "/create-organisation", http.StatusFound)
		return
	} else {
		query := r.URL.Query()
		query.Set("organisationid", orgs.Data[0].ID)
		r.URL.RawQuery = query.Encode()
	}

	http.Redirect(w, r, r.URL.String(), http.StatusFound)
}

func parseFormDate(input string) (time.Time, error) {
	return time.Parse("2006-01-02", input)
}

func defaultedDatesFromQueryString(query url.Values, numDaysFromNowDefault int, weekBoundary bool) (startTime, endTime time.Time, err error) {
	start := query.Get("start")
	end := query.Get("end")

	format := "2006-01-02"
	begin := time.Now()

	if weekBoundary {
		begin = util.NextDay(begin, time.Sunday)
	}

	now := begin.Format(format)
	then := begin.Add(24 * time.Duration(numDaysFromNowDefault) * time.Hour).Format(format)

	startTime, _ = time.Parse(format, now)
	endTime, _ = time.Parse(format, then)

	if start != "" {
		parsed, err := time.Parse(format, start)
		if err != nil {
			return startTime, endTime, err
		}
		startTime = parsed
	}
	if end != "" {
		parsed, err := time.Parse(format, end)
		if err != nil {
			return startTime, endTime, err
		}
		endTime = parsed
	}

	return startTime, endTime, nil
}

func deblank(arr []string) (deblanked []string) {
	for _, v := range arr {
		if v != "" {
			deblanked = append(deblanked, v)
		}
	}
	return
}

func sendEmailChangedNotification(ctx context.Context, target, old string) error {
	emailHTML, emailText := copy.EmailChangedEmail(target, old)

	subject := fmt.Sprintf("%s email changed", config.NAME)

	recipients := []string{target, old}

	for _, recipient := range recipients {
		mail := notifications.Email{
			To:      recipient,
			From:    config.SYSTEM_EMAIL,
			ReplyTo: config.SUPPORT_EMAIL,
			Text:    emailText,
			HTML:    emailHTML,
			Subject: subject,
		}

		task := kewpie.Task{}
		if err := task.Marshal(mail); err != nil {
			return err
		}

		if err := config.QUEUE.Publish(ctx, config.SEND_EMAIL_QUEUE_NAME, &task); err != nil {
			return err
		}
	}
	return nil
}

func dollarsToCents(in string) (int, error) {
	dollars, err := strconv.ParseFloat(in, 64)
	return int((dollars * 1000) / 10), err
}

func redirToLogin(w http.ResponseWriter, r *http.Request) {
	values := url.Values{
		"next": []string{r.URL.String()},
	}

	http.Redirect(w, r, "/login?"+values.Encode(), 302)
	return
}

func nextFlow(defaultURL string, form url.Values) string {
	ret, _ := url.Parse(defaultURL)
	next := form.Get("next")
	if next != "" {
		parsed, _ := url.Parse(next)
		if parsed.Path != "login" && parsed.Path != "/login" {
			ret.Path = parsed.Path
		}
		for k, v := range parsed.Query() {
			q := ret.Query()
			q[k] = v
			ret.RawQuery = q.Encode()
		}
	}
	if form.Get("flow") != "" {
		q := ret.Query()
		q.Set("flow", form.Get("flow"))
		ret.RawQuery = q.Encode()
	}
	if form.Get("next_fragment") != "" {
		ret.Fragment = form.Get("next_fragment")
	}
	return ret.String()
}

func isClientSafe(err error) (bool, string) {
	type clientSafe interface {
		ClientSafeMessage() string
	}
	cse, ok := err.(clientSafe)
	if ok {
		return ok, cse.ClientSafeMessage()
	} else {
		return false, ""
	}
}

func userFromContext(ctx context.Context) models.User {
	if !isLoggedIn(ctx) {
		return models.User{}
	}
	return ctx.Value("user").(models.User)
}

func orgUserFromContext(ctx context.Context, org models.Organisation) models.OrganisationUser {
	if v, ok := ctx.Value("organisation_users").(models.OrganisationUsers); ok {
		for _, ou := range v.Data {
			if ou.OrganisationID == org.ID {
				return ou
			}
		}
	}
	return models.OrganisationUser{}
}

func flashesFromContext(ctx context.Context) flashes.Flashes {
	if ctx == nil {
		return flashes.Flashes{}
	}
	unconv := ctx.Value("flashes")
	unconvUser := ctx.Value("user")
	if unconv == nil && unconvUser == nil {
		return flashes.Flashes{}
	}

	f := flashes.Flashes{}
	if unconv != nil {
		f = unconv.(flashes.Flashes)
	}
	if unconvUser != nil {
		user := unconvUser.(models.User)
		for _, flash := range user.Flashes {
			if !flash.Sticky {
				user.DeleteFlash(ctx, flash)
			}
		}
		f = append(f, user.Flashes...)
	}
	return f
}

func searchQueryFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	unconv := ctx.Value("searchquery")
	if unconv == nil {
		return ""
	}

	return unconv.(string)
}
