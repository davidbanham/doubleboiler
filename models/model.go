package models

import (
	"database/sql/driver"
	"doubleboiler/config"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"

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

var ErrRelationships = fmt.Errorf("This entity has active relationships")
var ErrOrgLive = fmt.Errorf("This action is not permitted once an organisation is live")

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

type StringMap map[string]string

func (p StringMap) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, err
}

func (p *StringMap) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("Type assertion .([]byte) failed.")
	}

	return json.Unmarshal(source, p)
}

type IntMap map[string]int

func (p IntMap) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, err
}

func (p *IntMap) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return errors.New("Type assertion .([]byte) failed.")
	}

	return json.Unmarshal(source, p)
}
