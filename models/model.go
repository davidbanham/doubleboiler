package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"sync"
)

type ClientSafeError struct {
	Message string
}

func (err ClientSafeError) ClientSafeMessage() string {
	return err.Message
}

func (err ClientSafeError) Error() string {
	return err.Message
}

// For example
var ErrRelationships = ClientSafeError{Message: "This entity has active relationships"}
var ErrOrgLive = ClientSafeError{Message: "This action is not permitted once an organisation is live"}

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
