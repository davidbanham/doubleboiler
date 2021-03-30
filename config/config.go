package config

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/errorreporting"
	"cloud.google.com/go/storage"
	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/required_env"
	_ "github.com/lib/pq"
)

var Db *sql.DB

var PORT string
var SECRET string
var WEBHOOK_SECRET string
var HASH_KEY string
var BLOCK_KEY string
var DOMAIN string
var URI string
var SYSTEM_EMAIL string
var SYSTEM_EMAIL_ONLY string
var SUPPORT_EMAIL string
var NAME string
var AUTOCERT bool
var TLS bool
var STAGE string
var LOCAL bool
var RENDER_ERRORS bool
var REPORT_ERRORS bool
var MAINTENANCE_MODE bool
var KEWPIE_BACKEND string
var GOOGLE_PROJECT_ID string
var SAMPLEORG_ID string
var START_WORKERS bool

var SEND_EMAIL_QUEUE_NAME string

var MAX_TIME, _ = time.Parse(time.RFC3339, "9999-05-05T15:04:05Z")
var MIN_TIME = time.Unix(0, 0)

var QUEUE kewpie.Kewpie

var ErrorReporter *errorreporting.Client

var Bucket *storage.BucketHandle

func init() {
	required_env.Ensure(map[string]string{
		"PORT":                  "",
		"DB_URI":                "",
		"HASH_KEY":              "",
		"BLOCK_KEY":             "",
		"AWS_ACCESS_KEY_ID":     "",
		"AWS_SECRET_ACCESS_KEY": "",
		"STAGE":                 "",
		"LOCAL":                 "false",
		"DOMAIN":                "", //example.com
		"URI":                   "", //https://example.com:MAYBEPORT
		"NAME":                  "Doubleboiler",
		"SYSTEM_EMAIL":          "",
		"SUPPORT_EMAIL":         "",
		"SECRET":                "",
		"WEBHOOK_SECRET":        "",
		"AUTOCERT":              "false",
		"TLS":                   "true",
		"RENDER_ERRORS":         "false",
		"REPORT_ERRORS":         "true",
		"MAINTENANCE_MODE":      "false",
		"KEWPIE_BACKEND":        "",
		"GOOGLE_PROJECT_ID":     "",
		"SAMPLEORG_ID":          "3f815ebd-2eb7-4dae-be2d-460c726438e2",
		"START_WORKERS":         "",
	})

	PORT = os.Getenv("PORT")

	var err error
	dbURI := os.Getenv("DB_URI")
	if strings.Contains(os.Getenv("DB_URI"), "%s") {
		dbURI = fmt.Sprintf(os.Getenv("DB_URI"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"))
	}
	if err := os.Setenv("DB_URI", dbURI); err != nil {
		log.Fatal(err)
	}
	Db, err = sql.Open("postgres", dbURI)
	if err != nil {
		log.Fatal(err)
	}

	if os.Getenv("MAX_OPEN_SQL_CONNS") != "" {
		num, err := strconv.Atoi(os.Getenv("MAX_OPEN_SQL_CONNS"))
		log.Printf("INFO setting maximum DB connections for the pool to %d", num)
		if err != nil {
			log.Printf("ERROR parsing max open conns string: %s", err)
		}
		Db.SetMaxOpenConns(num)
	}

	ctx := context.Background()

	// Sets your Google Cloud Platform project ID.
	//projectID := os.Getenv("GOOGLE_PROJECT_ID")

	// Creates a client.
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create Google Cloud Storage client: %v", err)
	}

	// Sets the name for the new bucket.
	bucketName := "files.doubleboiler.app"

	// Creates a Bucket instance.
	Bucket = client.Bucket(bucketName)

	HASH_KEY = os.Getenv("HASH_KEY")
	BLOCK_KEY = os.Getenv("BLOCK_KEY")

	queueNameTemplate := "doubleboiler_%s_%s"
	STAGE = os.Getenv("STAGE")

	SEND_EMAIL_QUEUE_NAME = fmt.Sprintf(queueNameTemplate, STAGE, "send_email")

	allQueues := []string{
		SEND_EMAIL_QUEUE_NAME,
	}

	QUEUE.AddPublishMiddleware(func(ctx context.Context, t *kewpie.Task, queueName string) error {
		fmt.Println("DEBUG Publishing kewpie task", t)
		return nil
	})

	QUEUE.AddPublishMiddleware(func(ctx context.Context, t *kewpie.Task, queueName string) error {
		// Default handler URL
		if t.Tags.Get("handler_url") == "" {
			t.Tags.Set("handler_url", fmt.Sprintf("%s/webhooks/tasks/%s", URI, queueName))
		}
		return nil
	})

	KEWPIE_BACKEND = os.Getenv("KEWPIE_BACKEND")

	if err := QUEUE.Connect(KEWPIE_BACKEND, allQueues, Db); err != nil {
		log.Fatal("ERROR", err)
	}

	DOMAIN = os.Getenv("DOMAIN")
	URI = os.Getenv("URI")
	SAMPLEORG_ID = os.Getenv("SAMPLEORG_ID")

	SECRET = os.Getenv("SECRET")
	WEBHOOK_SECRET = os.Getenv("WEBHOOK_SECRET")

	NAME = os.Getenv("NAME")
	SYSTEM_EMAIL = fmt.Sprintf(`"%s" <%s>`, NAME, os.Getenv("SYSTEM_EMAIL"))
	SYSTEM_EMAIL_ONLY = os.Getenv("SYSTEM_EMAIL")
	SUPPORT_EMAIL = fmt.Sprintf(`"%s" <%s>`, NAME, os.Getenv("SUPPORT_EMAIL"))

	AUTOCERT = os.Getenv("AUTOCERT") == "true"
	TLS = os.Getenv("TLS") == "true"
	RENDER_ERRORS = os.Getenv("RENDER_ERRORS") == "true"
	REPORT_ERRORS = os.Getenv("REPORT_ERRORS") == "true"

	START_WORKERS = os.Getenv("START_WORKERS") == "true"

	MAINTENANCE_MODE = os.Getenv("MAINTENANCE_MODE") == "true"

	LOCAL = os.Getenv("LOCAL") == "true"

	GOOGLE_PROJECT_ID = os.Getenv("GOOGLE_PROJECT_ID")

	ErrorReporter, err = errorreporting.NewClient(ctx, GOOGLE_PROJECT_ID, errorreporting.Config{
		ServiceName: "doubleboiler",
		OnError: func(err error) {
			log.Printf("ERROR Could not log error: %v", err)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ReportError(err error) {
	if REPORT_ERRORS {
		if err == nil {
			log.Println("ERROR ReportError called with a nil error")
			return
		}
		log.Println("ERROR", err.Error())
		if ErrorReporter != nil {
			ErrorReporter.Report(errorreporting.Entry{
				Error: err,
			})
		}
	}
}
