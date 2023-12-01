package models

import (
	"context"
	"database/sql"
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

func (this *OrganisationUser) colmap() *Colmap {
	return &Colmap{
		"organisations_users.id":              &this.ID,
		"organisations_users.user_id":         &this.UserID,
		"organisations_users.organisation_id": &this.OrganisationID,
		"users.email":                         &this.Email,
		"organisations_users.revision":        &this.Revision,
		"organisations_users.roles":           &this.Roles,
		"organisations_users.name":            &this.Name,
		"organisations_users.family_name":     &this.FamilyName,
		"organisations_users.created_at":      &this.CreatedAt,
		"organisations_users.updated_at":      &this.UpdatedAt,
	}
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
		validMap := ValidRoles.ByName()
		if _, ok := validMap[role.Name]; !ok {
			return ClientSafeError{fmt.Sprintf("Invalid Role: %s", role.Name)}
		}
	}
	return nil
}

func (this *OrganisationUser) Save(ctx context.Context) error {
	if err := this.checkRolesAreValid(); err != nil {
		return err
	}

	q, props, newRev := StandardSave("organisations_users", this.colmap(), this.auditQuery(ctx, "U"))

	if err := ExecSave(ctx, q, props); err != nil {
		return err
	}

	this.Revision = newRev

	return nil
}

func (this *OrganisationUser) FindByColumn(ctx context.Context, col, val string) error {
	q, props := StandardFindByColumn("organisations_users JOIN users ON organisations_users.user_id = users.id", this.colmap(), col)
	if err := StandardExecFindByColumn(ctx, q, val, props); err != nil {
		return err
	}

	return nil
}

func (orguser *OrganisationUser) FindByID(ctx context.Context, id string) error {
	return orguser.FindByColumn(ctx, "organisations_users.id", id)
}

func (orguser OrganisationUser) Delete(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	_, err := db.ExecContext(ctx, orguser.auditQuery(ctx, "D")+"DELETE FROM organisations_users WHERE id = $1 AND revision = $2", orguser.ID, orguser.Revision)
	return err
}

func (orguser OrganisationUser) FullName() string {
	return orguser.Name + " " + orguser.FamilyName
}

func (orguser OrganisationUser) Label() string {
	fn := orguser.FullName()
	if fn != " " {
		return fn
	}
	return orguser.Email
}

type OrganisationUsers struct {
	Data     []OrganisationUser
	Criteria Criteria
}

func (this OrganisationUsers) colmap() *Colmap {
	r := OrganisationUser{}
	return r.colmap()
}

func (this *OrganisationUsers) FindAll(ctx context.Context, criteria Criteria) error {
	this.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	cols, _ := this.colmap().Split()

	var rows *sql.Rows
	var err error

	switch v := criteria.Query.(type) {
	default:
		return ErrInvalidQuery{Query: v, Model: "audit_log"}
	case custom:
		switch v := criteria.customQuery.(type) {
		default:
			return ErrInvalidQuery{Query: v, Model: "audit_log"}
		}
	case Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "organisations_users JOIN users ON organisations_users.user_id = users.id", criteria.Filters, criteria.Pagination, "name"), v.Args()...)
	}
	if err != nil {

		return err
	}
	defer rows.Close()

	for rows.Next() {
		ou := OrganisationUser{}
		ou.Roles = Roles{}
		props := ou.colmap().ByKeys(cols)
		if err := rows.Scan(props...); err != nil {
			return err
		}

		(*this).Data = append((*this).Data, ou)
	}

	return nil
}
