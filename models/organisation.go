package models

import (
	"context"
	"database/sql"
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
	Toggles   Toggles
}

var RequireAdmin2FA = Toggle{
	Category: "security",
	Label:    "Require 2FA for admins",
	Key:      "require_admin_2fa",
	HelpText: `Require all organisation admins to have 2-step authentication enabled.`,
}

var ValidToggles = []Toggle{RequireAdmin2FA}

func (this *Organisation) colmap() *Colmap {
	return &Colmap{
		"id":         &this.ID,
		"name":       &this.Name,
		"country":    &this.Country,
		"revision":   &this.Revision,
		"created_at": &this.CreatedAt,
		"updated_at": &this.UpdatedAt,
		"toggles":    &this.Toggles,
	}
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

func (this *Organisation) Save(ctx context.Context) error {
	this.Toggles.Populate(ValidToggles)

	q, props, newRev := StandardSave("organisations", this.colmap(), this.auditQuery(ctx, "U"))

	if err := ExecSave(ctx, q, props); err != nil {
		return err
	}

	this.Revision = newRev

	return nil
}

func (this *Organisation) FindByColumn(ctx context.Context, col, val string) error {
	q, props := StandardFindByColumn("organisations", this.colmap(), col)
	if err := StandardExecFindByColumn(ctx, q, val, props); err != nil {
		return err
	}
	this.Toggles.Populate(ValidToggles)
	return nil
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

func (this Organisations) colmap() *Colmap {
	r := Organisation{}
	return r.colmap()
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

	cols, _ := this.colmap().Split()

	var rows *sql.Rows
	var err error

	switch v := criteria.Query.(type) {
	default:
		return ErrInvalidQuery{Query: v, Model: "organisations"}
	case custom:
		switch v := criteria.customQuery.(type) {
		default:
			return ErrInvalidQuery{Query: v, Model: "organisations"}
		case OrganisationsContainingUser:
			filterQuery, filterProps := criteria.Filters.Query(2)
			props := append([]any{v.ID}, filterProps...)

			rows, err = db.QueryContext(ctx, `
		SELECT
		`+strings.Join(cols, ",")+`
		FROM organisations
		`+filterQuery+`
		AND id IN (SELECT DISTINCT organisation_id FROM organisations_users WHERE user_id = $1)
		ORDER BY name`+criteria.Pagination.PaginationQuery(), props...)
			if err != nil {
				return err
			}
		}
	case Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "organisations", criteria.Filters, criteria.Pagination, "name"), v.Args()...)
	}

	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		org := Organisation{}
		props := org.colmap().ByKeys(cols)
		if err = rows.Scan(props...); err != nil {
			return err
		}

		org.Toggles.Populate(ValidToggles)

		(*this).Data = append((*this).Data, org)
	}

	return err
}
