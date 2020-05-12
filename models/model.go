package models

import (
	"doubleboiler/config"
	"fmt"
	"log"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"
	kewpie "github.com/davidbanham/kewpie_go/v3"
)

var queue kewpie.Kewpie

func init() {
	if err := queue.Connect(config.KEWPIE_BACKEND, []string{
		config.SEND_EMAIL_QUEUE_NAME,
	}, config.Db); err != nil {
		log.Fatal("ERROR", err)
	}
}

func getTable(name string) *firestore.CollectionRef {
	return config.Db.Collection(strings.Join([]string{config.NAME, config.STAGE, name}, "/"))
}

type Query int

const (
	All Query = 1 + iota
	OrganisationsContainingUser
	ByOrg
	ByCol
	ByUser
)

var ErrRelationships = fmt.Errorf("This entity has active relationships")
var ErrOrgLive = fmt.Errorf("This action is not permitted once an organisation is live")
var ErrNotFound = fmt.Errorf("Not found")

func Parallelize(functions ...func() error) (errors []error) {
	var waitGroup sync.WaitGroup
	mux := &sync.Mutex{}
	waitGroup.Add(len(functions))

	defer waitGroup.Wait()

	for _, function := range functions {
		// We can't do this with a transaction, but it should be safe with a standard read
		//go func(copy func() error) {
		func(copy func() error) {
			defer waitGroup.Done()
			err := copy()
			if err != nil {
				mux.Lock()
				errors = append(errors, err)
				mux.Unlock()
			}
		}(function)
	}
	return
}
