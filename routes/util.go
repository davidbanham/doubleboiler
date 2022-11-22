package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/flashes"
	"doubleboiler/heroicons"
	"doubleboiler/logger"
	"doubleboiler/models"
	"doubleboiler/util"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/davidbanham/english_conjoin"
	"github.com/davidbanham/human_duration"
	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
	uuid "github.com/satori/go.uuid"
)

// Tmpl exports the compiled templates
var Tmpl *template.Template

func init() {
	templateFuncMap := template.FuncMap{
		"hash": calcHash,
		"despace": func(s string) string {
			return strings.Replace(s, " ", "_", -1)
		},
		"ToLower": strings.ToLower,
		"humanTime": func(t time.Time) string {
			loc, err := time.LoadLocation("Australia/Sydney")
			if err != nil {
				loc, _ = time.LoadLocation("UTC")
			}
			return t.In(loc).Format(time.RFC822)
		},
		"humanDate": func(t time.Time) string {
			loc, err := time.LoadLocation("Australia/Sydney")
			if err != nil {
				loc, _ = time.LoadLocation("UTC")
			}
			return t.In(loc).Format("02 Jan 2006")
		},
		"humanDayDate": func(t time.Time) string {
			loc, err := time.LoadLocation("Australia/Sydney")
			if err != nil {
				loc, _ = time.LoadLocation("UTC")
			}
			return t.In(loc).Format("Mon 02 Jan 2006")
		},
		"isoTime": func(t time.Time) string {
			return t.Format(time.RFC3339)
		},
		"stringToTime": func(d string) time.Time {
			t, _ := time.Parse(time.RFC3339, d)
			return t
		},
		"weekdayOffset": func(s string) int {
			t, _ := time.Parse(time.RFC3339, s)
			return int(t.Weekday())
		},
		"diff": func(a, b int) int {
			return a - b
		},
		"breakMonths": func(nights []string) [][]string {
			monthNights := [][]string{}
			target := 0
			monthNights = append(monthNights, []string{})
			for i, n := range nights {
				night, _ := time.Parse(time.RFC3339, n)
				if i != 0 {
					lastNight, _ := time.Parse(time.RFC3339, nights[i-1])
					if night.Month() != lastNight.Month() {
						monthNights = append(monthNights, []string{})
						target += 1
					}
				}
				monthNights[target] = append(monthNights[target], night.Format(time.RFC3339))
			}
			return monthNights
		},
		"dollarise": func(in int) string {
			return util.Dollarise(in)
		},
		"dollarise_float": func(in float32) string {
			return util.Dollarise(int(in))
		},
		"dollarise_int64": func(in int64) string {
			return util.Dollarise(int(in))
		},
		"cents_to_dollars_int": func(in int) float64 {
			return float64(in) / 100
		},
		"cents_to_dollars_int64": func(in int64) float64 {
			return float64(in) / 100
		},
		"cents_to_dollars": func(in float32) float32 {
			return in / 100
		},
		"csv": func(in []string) string {
			return strings.Join(in, ",")
		},
		"ssv": func(in []string) string {
			return strings.Join(in, "; ")
		},
		"dateonly": func(in time.Time) string {
			return in.Format("2006-01-02")
		},
		"datetime": func(in time.Time) string {
			return in.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
		},
		"breakLines": func(in string) []string {
			return strings.Split(in, "\n")
		},
		"breakOnAnd": func(in string) []string {
			return strings.Split(in, " AND ")
		},
		"humanDuration": human_duration.String,
		"nextPeriodStart": func(start, end time.Time) time.Time {
			dur := end.Sub(start) + (24 * time.Hour)
			return start.Add(dur)
		},
		"nextPeriodEnd": func(start, end time.Time) time.Time {
			dur := end.Sub(start) + (24 * time.Hour)
			return end.Add(dur)
		},
		"prevPeriodStart": func(start, end time.Time) time.Time {
			dur := end.Sub(start) + (24 * time.Hour)
			return start.Add(-dur)
		},
		"prevPeriodEnd": func(start, end time.Time) time.Time {
			dur := end.Sub(start) + (24 * time.Hour)
			return end.Add(-dur)
		},
		"contains": func(str []string, target string) bool {
			for _, s := range str {
				if s == target {
					return true
				}
			}
			return false
		},
		"unix_to_time": func(in int64) time.Time {
			return time.Unix(in, 0)
		},
		"unrealDate": func(d time.Time) bool {
			tooLong := time.Date(1950, time.January, 0, 0, 0, 0, 0, time.Local)
			tooLate := time.Date(9000, time.January, 0, 0, 0, 0, 0, time.Local)
			if d.Before(tooLong) {
				return true
			}
			if d.After(tooLate) {
				return true
			}
			return false
		},
		"add": func(i, j int) int {
			return i + j
		},
		"firstFiveChars": util.FirstFiveChars,
		"toUpper":        strings.ToUpper,
		"randID": func() string {
			return util.FirstFiveChars(uuid.NewV4().String())
		},
		"auditActions": func(abbrev string) string {
			mapping := map[string]string{
				"I": "Created",
				"U": "Updated",
				"D": "Deleted",
				"T": "Truncated",
			}

			return mapping[abbrev]
		},
		"loggedIn": isLoggedIn,
		"userEmail": func(ctx context.Context) string {
			return ctx.Value("user").(models.User).Email
		},
		"user": func(ctx context.Context) models.User {
			return ctx.Value("user").(models.User)
		},
		"orgsFromContext": func(ctx context.Context) models.Organisations {
			return orgsFromContext(ctx)
		},
		"flashes": flashesFromContext,
		"activeOrgFromContext": func(ctx context.Context) models.Organisation {
			return activeOrgFromContext(ctx)
		},
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
			return util.CalcToken(user.ID, "")
		},
		"isAppAdmin": isAppAdmin,
		"chrome": func(ctx context.Context) bool {
			if ctx == nil {
				return true
			}
			val := ctx.Value("chrome")
			if val == nil {
				return true
			}
			return val.(bool)
		},
		"percentage": func(total, percentage int) int {
			return int(float64(total) * float64(percentage) / 100)
		},
		"percentify": func(in float32) string {
			return fmt.Sprintf("%.2f", in) + "%"
		},
		"thisYear": func() int {
			return time.Now().Year()
		},
		"mod":     func(i, j int) bool { return i%j == 0 },
		"numDays": func(d time.Duration) int { return int(d / (24 * time.Hour)) },
		"isProd":  func() bool { return config.STAGE == "production" },
		"isLocal": func() bool { return config.LOCAL },
		"now": func() string {
			return time.Now().Format("2006-01-02")
		},
		"nextWeekStart": func() string {
			return util.NextDay(time.Now(), time.Monday).Format("2006-01-02")
		},
		"conjoinAnd": func(in []string) string {
			return english_conjoin.ConjoinAnd(in)
		},
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
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"crumbs": func(values ...string) ([]Crumb, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			crumbs := []Crumb{}
			for i := 0; i < len(values); i += 2 {
				crumbs = append(crumbs, Crumb{
					Title: values[i],
					Path:  values[i+1],
				})
			}
			return crumbs, nil
		},
		"noescape": func(str string) template.HTML {
			return template.HTML(str)
		},
		"urlescape": func(input string) string {
			return url.QueryEscape(input)
		},
		"heroIcon": func(name string) string {
			return heroicons.Icons[name]
		},
		"uniq": func() string {
			return uuid.NewV4().String()
		},
		"queryString": func(vals url.Values) template.URL {
			return "?" + template.URL(vals.Encode())
		},
		"searchableEntities": func(ctx context.Context) []models.Searchable {
			ret := []models.Searchable{}
			org := activeOrgFromContext(ctx)
			for _, entity := range models.Searchables {
				if can(ctx, org, entity.RequiredRole.Name) {
					ret = append(ret, entity)
				}
			}
			return ret
		},
		"isOrgSettingsPage": func(ctx context.Context) bool {
			currentURL := ctx.Value("url")
			if v, ok := currentURL.(*url.URL); ok {
				parts := strings.Split(v.Path, "/")
				if len(parts) > 1 {
					if parts[1] == "organisations" {
						return true
					}
				}
			}
			return false
		},
		"selectorSafe": func(in string) string {
			return strings.ReplaceAll(in, ".", "-")
		},
	}

	Tmpl = template.Must(template.New("main").Funcs(templateFuncMap).ParseGlob(getPath() + "/*"))
}

func getPath() string {
	if _, err := os.Open("../views"); err == nil {
		return "../views/"
	}
	if _, err := os.Open("./views"); err == nil {
		return "./views/"
	}
	if _, err := os.Open("../../../views"); err == nil {
		return "../../../views"
	}
	if _, err := os.Open("../../views"); err == nil {
		return "../../views"
	}
	return ""
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

type Crumb struct {
	Title string
	Path  string
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

func isAppAdmin(ctx context.Context) bool {
	if !isLoggedIn(ctx) {
		return false
	}
	return ctx.Value("user").(models.User).Admin
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
