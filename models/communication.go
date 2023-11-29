package models

import (
	"context"
	"database/sql"
	"doubleboiler/util"
	"log"
	"time"

	"github.com/davidbanham/scum/search"
	uuid "github.com/satori/go.uuid"
)

type Communication struct {
	ID             string
	Revision       string
	OrganisationID string
	Sent           time.Time
	UpdatedAt      time.Time
	UserID         sql.NullString
	Channel        string
	Subject        string
}

func (this *Communication) colmap() *Colmap {
	return &Colmap{
		"id":              &this.ID,
		"revision":        &this.Revision,
		"organisation_id": &this.OrganisationID,
		"created_at":      &this.Sent,
		"updated_at":      &this.UpdatedAt,
		"user_id":         &this.UserID,
		"channel":         &this.Channel,
		"subject":         &this.Subject,
	}
}

func (this *Communication) New(organisationID, channel, subject string) {
	this.ID = uuid.NewV4().String()
	this.Revision = uuid.NewV4().String()
	this.OrganisationID = organisationID
	this.Channel = channel
	this.Subject = subject
	this.Sent = time.Now()
}

func LogUserCommunication(ctx context.Context, organisationID string, user User, channel, subject string) error {
	communication := Communication{}
	communication.New(organisationID, channel, subject)
	communication.UserID = sql.NullString{
		Valid:  true,
		String: user.ID,
	}
	return communication.Save(ctx)
}

func (communication *Communication) auditQuery(ctx context.Context, action string) string {
	return auditQuery(ctx, action, "communications", communication.ID, communication.OrganisationID)
}

func (this *Communication) Save(ctx context.Context) error {
	q, props, newRev := StandardSave("communications", this.colmap(), this.auditQuery(ctx, "U"))

	if err := ExecSave(ctx, q, props); err != nil {
		return err
	}

	this.Revision = newRev

	return nil
}

func (communication *Communication) FindByID(ctx context.Context, id string) error {
	return communication.FindByColumn(ctx, "id", id)
}

func (this *Communication) FindByColumn(ctx context.Context, col, val string) error {
	q, props := StandardFindByColumn("communications", this.colmap(), col)
	return StandardExecFindByColumn(ctx, q, val, props)
}

type Communications struct {
	Data     []Communication
	Criteria Criteria
}

func (this Communications) colmap() *Colmap {
	r := Communication{}
	return r.colmap()
}

func (Communications) AvailableFilters() Filters {
	sentBetween := CreatedBetween{}
	if err := sentBetween.Hydrate(DateFilterOpts{
		Label: "Sent Between",
		ID:    "created-between",
		Table: "communications",
		Col:   "created_at",
		Period: util.Period{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
	}); err != nil {
		log.Fatal(err)
	}
	viaEmail := HasProp{}
	if err := viaEmail.Hydrate(HasPropOpts{
		Label: "Email",
		ID:    "communication-via-email",
		Table: "communications",
		Col:   "channel",
		Value: "email",
	}); err != nil {
		log.Fatal(err)
	}
	return Filters{
		&sentBetween,
		&viaEmail,
	}
}

func (Communications) Searchable() Searchable {
	return Searchable{
		EntityType: "Communication",
		Label:      "subject",
		Path:       "communications",
		Tablename:  "communications",
		Permitted:  search.BasicRoleCheck("admin"),
	}
}

func (this *Communications) FindAll(ctx context.Context, criteria Criteria) error {
	this.Criteria = criteria

	db := ctx.Value("tx").(Querier)

	cols, _ := this.colmap().Split()

	var rows *sql.Rows
	var err error

	switch v := criteria.Query.(type) {
	default:
		return ErrInvalidQuery{Query: v, Model: "communications"}
	case custom:
		switch v := criteria.customQuery.(type) {
		default:
			return ErrInvalidQuery{Query: v, Model: "communications"}
		}
	case Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "communications", criteria.Filters, criteria.Pagination, "subject"), v.Args()...)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		communication := Communication{}
		props := communication.colmap().ByKeys(cols)
		if err := rows.Scan(props...); err != nil {
			return err
		}
		(*this).Data = append((*this).Data, communication)
	}
	return err
}

func (this Communications) Users(ctx context.Context) (Users, error) {
	userIDs := []string{}
	for _, comm := range this.Data {
		if comm.UserID.Valid {
			userIDs = append(userIDs, comm.UserID.String)
		}
	}

	users := Users{}
	if err := users.FindAll(ctx, Criteria{
		Query:      &ByIDs{IDs: util.Uniq(userIDs)},
		Filters:    Filters{},
		Pagination: Pagination{},
	}); err != nil {
		return Users{}, err
	}

	return users, nil
}
