package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/errorreporting"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/davidbanham/required_env"
)

var Db *firestore.Client

var PORT string
var SECRET string
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
var GOOGLE_PUBSUB_AUDIENCE string
var GOOGLE_PROJECT_ID string
var SAMPLEORG_ID string

var SEND_EMAIL_QUEUE_NAME string

var MAX_TIME, _ = time.Parse(time.RFC3339, "9999-05-05T15:04:05Z")
var MIN_TIME = time.Unix(0, 0)

var ErrorReporter *errorreporting.Client

var Bucket *storage.BucketHandle

func init() {
	required_env.Ensure(map[string]string{
		"PORT":                   "",
		"HASH_KEY":               "",
		"BLOCK_KEY":              "",
		"AWS_ACCESS_KEY_ID":      "",
		"AWS_SECRET_ACCESS_KEY":  "",
		"STAGE":                  "",
		"LOCAL":                  "false",
		"DOMAIN":                 "", //example.com
		"URI":                    "", //https://example.com:MAYBEPORT
		"NAME":                   "Doubleboiler",
		"SYSTEM_EMAIL":           "",
		"SUPPORT_EMAIL":          "",
		"SECRET":                 "",
		"AUTOCERT":               "false",
		"TLS":                    "true",
		"RENDER_ERRORS":          "false",
		"REPORT_ERRORS":          "true",
		"MAINTENANCE_MODE":       "false",
		"KEWPIE_BACKEND":         "",
		"GOOGLE_PROJECT_ID":      "",
		"SAMPLEORG_ID":           "3f815ebd-2eb7-4dae-be2d-460c726438e2",
		"GOOGLE_PUBSUB_AUDIENCE": "",
	})

	PORT = os.Getenv("PORT")
	GOOGLE_PROJECT_ID = os.Getenv("GOOGLE_PROJECT_ID")

	var err error
	ctx := context.Background()

	Db, err = firestore.NewClient(ctx, GOOGLE_PROJECT_ID)
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}

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

	DOMAIN = os.Getenv("DOMAIN")
	URI = os.Getenv("URI")
	SAMPLEORG_ID = os.Getenv("SAMPLEORG_ID")

	SECRET = os.Getenv("SECRET")

	NAME = os.Getenv("NAME")
	SYSTEM_EMAIL = fmt.Sprintf(`"%s" <%s>`, NAME, os.Getenv("SYSTEM_EMAIL"))
	SYSTEM_EMAIL_ONLY = os.Getenv("SYSTEM_EMAIL")
	SUPPORT_EMAIL = fmt.Sprintf(`"%s" <%s>`, NAME, os.Getenv("SUPPORT_EMAIL"))

	AUTOCERT = os.Getenv("AUTOCERT") == "true"
	TLS = os.Getenv("TLS") == "true"
	RENDER_ERRORS = os.Getenv("RENDER_ERRORS") == "true"
	REPORT_ERRORS = os.Getenv("REPORT_ERRORS") == "true"

	MAINTENANCE_MODE = os.Getenv("MAINTENANCE_MODE") == "true"

	LOCAL = os.Getenv("LOCAL") == "true"

	KEWPIE_BACKEND = os.Getenv("KEWPIE_BACKEND")

	GOOGLE_PUBSUB_AUDIENCE = os.Getenv("GOOGLE_PUBSUB_AUDIENCE")

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
