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

var communicationCols = []string{
	"organisation_id",
	"user_id",
	"channel",
	"subject",
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
	props := []any{
		this.Revision,
		this.ID,
		this.OrganisationID,
		this.UserID,
		this.Channel,
		this.Subject,
	}

	newRev, err := StandardSave(ctx, "communications", communicationCols, this.auditQuery(ctx, "U"), props)
	if err == nil {
		this.Revision = newRev
	}
	return err
}

func (communication *Communication) FindByID(ctx context.Context, id string) error {
	return communication.FindByColumn(ctx, "id", id)
}

func (this *Communication) FindByColumn(ctx context.Context, col, val string) error {
	props := []any{
		&this.Revision,
		&this.ID,
		&this.Sent,
		&this.UpdatedAt,
		&this.OrganisationID,
		&this.UserID,
		&this.Channel,
		&this.Subject,
	}

	return StandardFindByColumn(ctx, "communications", communicationCols, col, val, props)
}

type Communications struct {
	Data     []Communication
	Criteria Criteria
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

	var rows *sql.Rows
	var err error

	cols := append([]string{
		"revision",
		"id",
		"created_at",
		"updated_at",
	}, communicationCols...)

	switch v := criteria.Query.(type) {
	case Query:
		rows, err = db.QueryContext(ctx, v.Construct(cols, "communications", criteria.Filters, criteria.Pagination, "subject"), v.Args()...)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		communication := Communication{}
		if err := rows.Scan(
			&communication.Revision,
			&communication.ID,
			&communication.Sent,
			&communication.UpdatedAt,
			&communication.OrganisationID,
			&communication.UserID,
			&communication.Channel,
			&communication.Subject,
		); err != nil {
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
		Query:      ByIDs{IDs: util.Uniq(userIDs)},
		Filters:    Filters{},
		Pagination: Pagination{},
	}); err != nil {
		return Users{}, err
	}

	return users, nil
}
