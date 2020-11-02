package workers

import (
	"bytes"
	"context"
	"crypto/tls"
	"doubleboiler/config"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	kewpie "github.com/davidbanham/kewpie_go/v3"
)

type handler struct {
	path string
}

var queue kewpie.Kewpie

func init() {
	if config.KEWPIE_BACKEND == "google_pubsub" {
		return
	}
	// The expectation is that we're using a google pubsub queue configured in push mode.
	// If that's not the case (eg in testing) this shim takes any other queue and turns jobs into POST messages pointed at the relevant path
	if err := queue.Connect(config.KEWPIE_BACKEND, []string{
		config.SEND_EMAIL_QUEUE_NAME,
	}, config.Db); err != nil {
		log.Fatal("ERROR: workers init ", err)
	}

	if strings.Contains(config.URI, "localhost") {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	go (func() {
		time.Sleep(10 * time.Second)
		log.Println("Starting workers")
		go queue.Subscribe(context.Background(), config.SEND_EMAIL_QUEUE_NAME, handler{
			path: "/webhooks/send-email",
		})
		log.Println("Workers setup")
	})()
}

// Handle handles callback messages from the send of a message.
func (h handler) Handle(task kewpie.Task) (bool, error) {
	log.Printf("Handle invoked")
	payload, err := json.Marshal(task)
	if err != nil {
		log.Fatalf("Error in handle: %+v", err)
		return false, err
	}

	log.Printf("Handle: About to invoke handler for path %v", config.URI+h.path+"?webhook-secret="+config.WEBHOOK_SECRET)
	buf := bytes.NewBuffer(payload)
	resp, err := http.Post(config.URI+h.path+"?webhook-secret="+config.WEBHOOK_SECRET, "application/json", buf)
	if err != nil {
		return true, err
	}

	log.Printf("Handle: Response from handler: %v", resp.StatusCode)
	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return true, err
		}
		if resp.StatusCode >= 500 {
			return true, fmt.Errorf(string(body))
		}
		if resp.StatusCode >= 400 {
			return false, fmt.Errorf(string(body))
		}
	}

	return false, nil
}
