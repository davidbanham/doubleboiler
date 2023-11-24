package models

import (
	"context"
	"database/sql"
	"doubleboiler/util"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
)

type OrganisationUser struct {
	ID             string
	UserID         string
	OrganisationID string
	Email          string
	Revision       string
	Roles          Roles
	Name           string
	FamilyName     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

var organisationUserCols = []string{
	"user_id",
	"organisation_id",
	"roles",
	"name",
	"family_name",
}

func (orguser *OrganisationUser) New(userID, organisationID string, roles Roles) {
	orguser.ID = uuid.NewV4().String()
	orguser.UserID = userID
	orguser.OrganisationID = organisationID
	orguser.Roles = roles
	orguser.CreatedAt = time.Now()
	orguser.UpdatedAt = time.Now()
}

func (orguser *OrganisationUser) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "organisations_users", orguser.ID, orguser.OrganisationID)
}

func (orguser OrganisationUser) checkRolesAreValid() error {
	for _, role := range orguser.Roles {
		if _, ok := ValidRoles[role.Name]; !ok {
			return ClientSafeError{fmt.Sprintf("Invalid Role: %s", role)}
		}
	}
	return nil
}

func (this *OrganisationUser) Save(ctx context.Context) error {
	if err := this.checkRolesAreValid(); err != nil {
		return err
	}

	props := []any{
		this.Revision,
		this.ID,
		this.UserID,
		this.OrganisationID,
		this.Roles,
		this.Name,
		this.FamilyName,
	}

	newRev, err := StandardSave(ctx, "organisations_users", organisationUserCols, this.auditQuery(ctx, "U"), props)
	if err == nil {
		this.Revision = newRev
	}
	return err
}

func (this *OrganisationUser) FindByColumn(ctx context.Context, col, val string) error {
	props := []any{
		&this.Revision,
		&this.ID,
		&this.CreatedAt,
		&this.UpdatedAt,
		&this.UserID,
		&this.OrganisationID,
		&this.Roles,
		&this.Name,
		&this.FamilyName,
		&this.Email,
	}

	cols := append(organisationUserCols, "(SELECT email FROM users WHERE id = user_id)")

	return StandardFindByColumn(ctx, "organisations_users", cols, col, val, props)
}

func (orguser *OrganisationUser) FindByID(ctx context.Context, id string) error {
	return orguser.FindByColumn(ctx, "id", id)
}

func (orguser OrganisationUser) Delete(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	_, err := db.ExecContext(ctx, orguser.auditQuery(ctx, "D")+"DELETE FROM organisations_users WHERE id = $1 AND revision = $2", orguser.ID, orguser.Revision)
	return err
}

func (orguser OrganisationUser) FullName() string {
	return orguser.Name + " " + orguser.FamilyName
}

type OrganisationUsers struct {
	Data     []OrganisationUser
	Criteria Criteria
}

func (organisationusers *OrganisationUsers) FindAll(ctx context.Context, criteria Criteria) error {
	cols := util.Prefix(append([]string{
		"id",
		"revision",
		"created_at",
		"updated_at",
	}, organisationUserCols...), "organisations_users.")
	cols = append(cols, "(SELECT email FROM users WHERE id = user_id)")

	organisationusers.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := criteria.Query.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "organisations_users", criteria.Filters, criteria.Pagination, "name"), v.Args()...)
	}
	if err != nil {

		return err
	}
	defer rows.Close()

	for rows.Next() {
		ou := OrganisationUser{}
		ou.Roles = Roles{}
		if err := rows.Scan(
			&ou.ID,
			&ou.Revision,
			&ou.CreatedAt,
			&ou.UpdatedAt,
			&ou.UserID,
			&ou.OrganisationID,
			&ou.Roles,
			&ou.Name,
			&ou.FamilyName,
			&ou.Email,
		); err != nil {
			return err
		}

		(*organisationusers).Data = append((*organisationusers).Data, ou)
	}

	return nil
}
