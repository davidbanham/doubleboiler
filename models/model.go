package models

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var Searchables []Searchable

type Searchable struct {
	Label            string
	RequiredRole     Role
	searchFunc       func(ByPhrase) string
	availableFilters func() Filters
}

type Filterable struct {
	AvailableFilters func() Filters
}

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
var ErrWrongRev = ClientSafeError{Message: "This record has been changed by another request since you loaded it. Review the changes by going back and refreshing, and try again if appropriate."}

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

func currentUser(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	unconv := ctx.Value("user")

	if unconv != nil {
		return unconv.(User).ID
	}
	return ""
}

func auditQuery(ctx context.Context, action, tableName, entityID, organisationID string) string {
	return fmt.Sprintf("WITH audit_entry AS (INSERT INTO audit_log (entity_id, organisation_id, table_name, action, user_id, old_row_data) VALUES ('%s', '%s', '%s', '%s', '%s', (SELECT to_jsonb(%s) - 'ts' FROM %s WHERE id = '%s')))", entityID, organisationID, tableName, action, currentUser(ctx), tableName, tableName, entityID)
}

func filterQuery(q Query) string {
	filters := append(q.ActiveFilters(), q.CustomFilters()...)
	if len(filters) == 0 {
		return " WHERE true = true "
	}
	fragments := []string{}
	for _, filter := range filters {
		fragments = append(fragments, filter.query())
	}
	return " WHERE " + strings.Join(fragments, " AND ")
}
