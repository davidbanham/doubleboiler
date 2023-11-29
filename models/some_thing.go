package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/davidbanham/scum/search"
	uuid "github.com/satori/go.uuid"
)

func init() {
	SearchTargets = append(SearchTargets, (SomeThings{}).Searchable())
}

type SomeThing struct {
	ID             string
	Name           string
	Description    string
	OrganisationID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Revision       string
}

func (this *SomeThing) colmap() *Colmap {
	return &Colmap{
		"id":              &this.ID,
		"name":            &this.Name,
		"description":     &this.Description,
		"organisation_id": &this.OrganisationID,
		"created_at":      &this.CreatedAt,
		"updated_at":      &this.UpdatedAt,
		"revision":        &this.Revision,
	}
}

func (this *SomeThing) New(name, description, organisationID string) {
	this.ID = uuid.NewV4().String()
	this.Name = name
	this.Description = description
	this.OrganisationID = organisationID
	this.CreatedAt = time.Now()
	this.UpdatedAt = time.Now()
}

func (this *SomeThing) FindByColumn(ctx context.Context, col, val string) error {
	q, props := StandardFindByColumn("some_things", this.colmap(), col)
	return StandardExecFindByColumn(ctx, q, val, props)
}

func (this *SomeThing) FindByID(ctx context.Context, id string) error {
	return this.FindByColumn(ctx, "id", id)
}

func (this *SomeThing) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "some_things", this.ID, this.OrganisationID)
}

func (this *SomeThing) Save(ctx context.Context) error {
	q, props, newRev := StandardSave("some_things", this.colmap(), this.auditQuery(ctx, "U"))

	if err := ExecSave(ctx, q, props); err != nil {
		return err
	}

	this.Revision = newRev

	return nil
}

func (this SomeThing) Label() string {
	return this.Name
}

type SomeThings struct {
	Data     []SomeThing
	Criteria Criteria
}

func (this SomeThings) colmap() *Colmap {
	r := SomeThing{}
	return r.colmap()
}

func (SomeThings) AvailableFilters() Filters {
	return standardFilters("some_things")
}

func (SomeThings) Searchable() Searchable {
	return Searchable{
		EntityType: "SomeThing",
		Label:      "name || ' - ' || description",
		Path:       "some-things",
		Tablename:  "some_things",
		Permitted:  search.BasicRoleCheck("admin"),
	}
}

func (this SomeThings) ByID() map[string]SomeThing {
	ret := map[string]SomeThing{}
	for _, t := range this.Data {
		ret[t.ID] = t
	}
	return ret
}

func (this *SomeThings) FindAll(ctx context.Context, criteria Criteria) error {
	this.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	cols, _ := this.colmap().Split()

	var rows *sql.Rows
	var err error

	switch v := criteria.Query.(type) {
	case Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "some_things", criteria.Filters, criteria.Pagination, "name"), v.Args()...)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		someThing := SomeThing{}
		props := someThing.colmap().ByKeys(cols)
		if err := rows.Scan(props...); err != nil {
			return err
		}
		(*this).Data = append((*this).Data, someThing)
	}
	return err
}
