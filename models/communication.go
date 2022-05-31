package models

import (
	"context"
	"database/sql"
	"doubleboiler/util"
	"fmt"
	"time"

	uuid "github.com/satori/go.uuid"
)

func init() {
	requiredRole := ValidRoles["admin"]
	Searchables = append(Searchables, Searchable{
		Label:            "Communications",
		RequiredRole:     requiredRole,
		searchFunc:       searchCommunications(requiredRole),
		availableFilters: communicationFilters,
	})
}

type Communication struct {
	ID             string
	Revision       string
	OrganisationID string
	UserID         sql.NullString
	Channel        string
	Subject        string
	Sent           time.Time
}

func (communication *Communication) New(organisationID, channel, subject string) {
	communication.ID = uuid.NewV4().String()
	communication.Revision = uuid.NewV4().String()
	communication.OrganisationID = organisationID
	communication.Channel = channel
	communication.Subject = subject
	communication.Sent = time.Now()
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

func (communication *Communication) Save(ctx context.Context) error {
	db := ctx.Value("tx").(Querier)

	row := db.QueryRowContext(ctx, communication.auditQuery(ctx, "U")+`INSERT INTO communications (
		id,
		revision,
		organisation_id,
		user_id,
		channel,
		subject,
		created_at
	) VALUES (
		$1, $2, $4, $5, $6, $7, $8
	) ON CONFLICT (revision) DO UPDATE SET (
		revision,
		organisation_id,
		user_id,
		channel,
		subject,
		created_at
	) = (
		$3, $4, $5, $6, $7, $8
	) RETURNING revision`,
		communication.ID,
		communication.Revision,
		uuid.NewV4().String(),
		communication.OrganisationID,
		communication.UserID,
		communication.Channel,
		communication.Subject,
		communication.Sent,
	)
	return row.Scan(&communication.Revision)
}

func (communication *Communication) FindByID(ctx context.Context, id string) error {
	return communication.FindByColumn(ctx, "id", id)
}

func (communication *Communication) FindByColumn(ctx context.Context, col, val string) error {
	db := ctx.Value("tx").(Querier)

	return db.QueryRowContext(ctx, `SELECT
	id,
	revision,
	organisation_id,
	user_id,
	channel,
	subject,
	created_at
	FROM communications WHERE `+col+` = $1`, val).Scan(
		&communication.ID,
		&communication.Revision,
		&communication.OrganisationID,
		&communication.UserID,
		&communication.Channel,
		&communication.Subject,
		&communication.Sent,
	)
}

type Communications struct {
	Data  []Communication
	Query Query
}

func (communications *Communications) FindAll(ctx context.Context, q Query) error {
	communications.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case ByUser:
		rows, err = db.QueryContext(ctx, `SELECT
		id,
		revision,
		organisation_id,
		user_id,
		channel,
		subject,
		created_at
		FROM communications `+filterQuery(v)+`
		AND user_id = $1
		ORDER BY created_at DESC`, v.Pagination(), v.ID)
	case ByOrg:
		rows, err = db.QueryContext(ctx, `SELECT
		communications.id,
		communications.revision,
		communications.organisation_id,
		communications.user_id,
		communications.channel,
		communications.subject,
		communications.created_at
		FROM communications
		LEFT JOIN organisations_users ON communications.user_id = organisations_users.user_id
		`+filterQuery(v)+`
		AND organisations_users.organisation_id = $1
		OR organisations_users.organisation_id IS NULL
		ORDER BY communications.created_at DESC
		`+v.Pagination(), v.ID)
	case All:
		rows, err = db.QueryContext(ctx, `SELECT
		id,
		revision,
		organisation_id,
		user_id,
		channel,
		subject,
		created_at
		FROM communications `+filterQuery(v)+`
		ORDER BY created_at DESC`+v.Pagination())
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		communication := Communication{}
		err = rows.Scan(
			&communication.ID,
			&communication.Revision,
			&communication.OrganisationID,
			&communication.UserID,
			&communication.Channel,
			&communication.Subject,
			&communication.Sent,
		)
		if err != nil {
			return err
		}
		(*communications).Data = append((*communications).Data, communication)
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
	if err := users.FindAll(ctx, ByIDs{IDs: util.Uniq(userIDs)}); err != nil {
		return Users{}, err
	}

	return users, nil
}

func (communications Communications) AvailableFilters() Filters {
	return communicationFilters()
}

func communicationFilters() Filters {
	return append(standardFilters(),
		HasProp{
			key:   "channel",
			value: "email",
			label: "Via Email",
			id:    "communication-via-email",
		},
		CreatedAfter{
			label: "Sent After",
			id:    "communication-sent-after",
		},
		CreatedBefore{
			label: "Sent Before",
			id:    "communication-sent-before",
		},
	)
}

func searchCommunications(requiredRole Role) func(ByPhrase) string {
	return func(query ByPhrase) string {
		if query.User.Admin || query.Roles.Can(requiredRole.Name) {
			return `SELECT
				text 'Communication' AS entity_type, text 'communications' AS uri_path, id AS id, subject AS label, ts_rank_cd(ts, query) AS rank
				FROM
				communications, plainto_tsquery('english', $2) query WHERE organisation_id = $1 AND query @@ ts`
		}
		return ""
	}
}
