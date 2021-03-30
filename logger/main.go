package logger

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"context"
	"fmt"

	"github.com/davidbanham/required_env"
)

type Level string

const Warning = Level("WARN")
const Error = Level("ERROR")
const Info = Level("INFO")
const Debug = Level("DEBUG")
const Audit = Level("NOTICE")

type entry struct {
	Message   string `json:"message"`
	Severity  Level  `json:"severity,omitempty"`
	Trace     string `json:"logging.googleapis.com/trace,omitempty"`
	RequestID string
}

var googleProjectID string
var jsonLogging bool

func init() {
	required_env.Ensure(map[string]string{
		"GOOGLE_PROJECT_ID": "",
		"JSON_LOGGING":      "false",
	})
	googleProjectID = os.Getenv("GOOGLE_PROJECT_ID")
	jsonLogging = os.Getenv("JSON_LOGGING") == "true"
}

func (this entry) toPlainText() string {
	tag := this.Trace
	if this.Trace == "" {
		tag = this.RequestID
	}
	return fmt.Sprintf("%s %s %s", this.Severity, tag, this.Message)
}

func Log(ctx context.Context, level Level, data ...interface{}) {
	elems := []string{}
	for _, datum := range data {
		elems = append(elems, fmt.Sprint(datum))
	}

	message := strings.ReplaceAll(strings.Join(elems, " "), "\n", "")

	entry := entry{
		Severity:  level,
		Trace:     "",
		RequestID: "background",
		Message:   message,
	}

	unconvTrace := ctx.Value("trace")
	if unconvTrace != nil {
		traceID := unconvTrace.(string)
		entry.Trace = fmt.Sprintf("projects/%s/traces/%s", googleProjectID, traceID)
	}

	unconvRequestID := ctx.Value("requestID")
	if unconvRequestID != nil {
		entry.RequestID = unconvRequestID.(string)
	}

	if jsonLogging {
		out, err := json.Marshal(entry)
		if err != nil {
			log.Println("ERROR marshalling log", err)
			log.Println(entry.toPlainText())
		}

		println(string(out))
	} else {
		log.Println(entry.toPlainText())
	}
}
