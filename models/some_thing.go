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

var someThingCols = []string{
	"organisation_id",
	"name",
	"description",
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
	props := []any{
		&this.Revision,
		&this.ID,
		&this.CreatedAt,
		&this.UpdatedAt,
		&this.OrganisationID,
		&this.Name,
		&this.Description,
	}

	return StandardFindByColumn(ctx, "some_things", someThingCols, col, val, props)
}

func (this *SomeThing) FindByID(ctx context.Context, id string) error {
	return this.FindByColumn(ctx, "id", id)
}

func (someThing SomeThing) Props() []any {
	return []any{
		someThing.OrganisationID,
		someThing.Name,
		someThing.Description,
	}
}

func (this *SomeThing) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "some_things", this.ID, this.OrganisationID)
}

func (this *SomeThing) Save(ctx context.Context) error {
	props := []any{
		this.Revision,
		this.ID,
		this.OrganisationID,
		this.Name,
		this.Description,
	}

	newRev, err := StandardSave(ctx, "some_things", someThingCols, this.auditQuery(ctx, "U"), props)
	if err == nil {
		this.Revision = newRev
	}
	return err
}

func (this SomeThing) Label() string {
	return this.Name
}

type SomeThings struct {
	Data     []SomeThing
	Criteria Criteria
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

	var rows *sql.Rows
	var err error

	cols := append([]string{
		"revision",
		"id",
		"created_at",
		"updated_at",
	}, someThingCols...)

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
		if err := rows.Scan(
			&someThing.Revision,
			&someThing.ID,
			&someThing.CreatedAt,
			&someThing.UpdatedAt,
			&someThing.OrganisationID,
			&someThing.Name,
			&someThing.Description,
		); err != nil {
			return err
		}
		(*this).Data = append((*this).Data, someThing)
	}
	return err
}
