package models

import (
	"context"
	"database/sql"
	"doubleboiler/util"
	"strings"
	"time"

	"github.com/davidbanham/scum/search"
	uuid "github.com/satori/go.uuid"
)

type Organisation struct {
	ID        string
	Name      string
	Country   string
	Revision  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var organisationCols = []string{
	"name",
	"country",
}

func (this *Organisation) New(name, country string) {
	this.ID = uuid.NewV4().String()
	this.Name = name
	this.Country = country
	this.CreatedAt = time.Now()
	this.UpdatedAt = time.Now()
}

func (org *Organisation) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "organisations", org.ID, org.ID)
}

func (this Organisation) Props() []any {
	return []any{}
}

func (this *Organisation) Save(ctx context.Context) error {
	props := []any{
		this.Revision,
		this.ID,
		this.Name,
		this.Country,
	}

	newRev, err := StandardSave(ctx, "organisations", organisationCols, this.auditQuery(ctx, "U"), props)
	if err == nil {
		this.Revision = newRev
	}
	return err
}

func (this *Organisation) FindByColumn(ctx context.Context, col, val string) error {
	props := []any{
		&this.Revision,
		&this.ID,
		&this.CreatedAt,
		&this.UpdatedAt,
		&this.Name,
		&this.Country,
	}

	return StandardFindByColumn(ctx, "organisations", organisationCols, col, val, props)
}

func (this *Organisation) FindByID(ctx context.Context, id string) error {
	return this.FindByColumn(ctx, "id", id)
}

func (this Organisation) Label() string {
	return this.Name
}

type Organisations struct {
	Data     []Organisation
	Criteria Criteria
}

func (Organisations) AvailableFilters() Filters {
	return standardFilters("organisations")
}

func (Organisations) Searchable() Searchable {
	return Searchable{
		EntityType: "Organisation",
		Label:      "name",
		Path:       "organisations",
		Tablename:  "organisations",
		Permitted:  search.BasicRoleCheck("admin"),
	}
}

func (this Organisations) ByID() map[string]Organisation {
	ret := map[string]Organisation{}
	for _, t := range this.Data {
		ret[t.ID] = t
	}
	return ret
}

func (this *Organisations) FindAll(ctx context.Context, criteria Criteria) error {
	this.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	cols := util.Prefix(append([]string{
		"id",
		"revision",
		"created_at",
		"updated_at",
	}, organisationCols...), "organisations.")

	switch v := criteria.Query.(type) {
	case Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "organisations", criteria.Filters, criteria.Pagination, "name"), v.Args()...)
	case OrganisationsContainingUser:
		rows, err = db.QueryContext(ctx, `
		SELECT
		`+strings.Join(cols, ",")+`
		FROM organisations
		JOIN organisations_users
		ON organisations_users.organisation_id = organisations.id
		`+criteria.Filters.Query()+`
		AND organisations_users.user_id = $1
		ORDER BY name`+criteria.Pagination.PaginationQuery(), v.ID)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		org := Organisation{}
		if err = rows.Scan(
			&org.ID,
			&org.Revision,
			&org.CreatedAt,
			&org.UpdatedAt,
			&org.Name,
			&org.Country,
		); err != nil {
			return err
		}

		(*this).Data = append((*this).Data, org)
	}

	return err
}
