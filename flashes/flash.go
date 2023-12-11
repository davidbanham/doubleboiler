package flashes

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"

	uuid "github.com/satori/go.uuid"
)

type Flashable interface {
	PersistFlash(context.Context, Flash) error
}

type Flash struct {
	Persistent  bool          `json:"persistent"`
	Sticky      bool          `json:"sticky"`
	EntityKey   string        `json:"entity_key"`
	ID          string        `json:"id"`
	Text        string        `json:"text"`
	Actions     []FlashAction `json:"actions"`
	Type        FlashLevel    `json:"type"`
	OnceOnlyKey string        `json:"once_only_key"`
}

type Flashes []Flash

func (this Flashes) Value() (driver.Value, error) {
	if len(this) == 0 {
		return "[]", nil
	}
	return json.Marshal(this)
}

func (this *Flashes) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &this)
}

type FlashAction struct {
	Url  string
	Text string
}

func (this *Flash) Add(ctx context.Context) (context.Context, error) {
	this.ID = uuid.NewV4().String()

	flashes := Flashes{}
	unconv := ctx.Value("flashes")
	if unconv != nil {
		flashes = unconv.(Flashes)
	}
	flashes = append(flashes, *this)
	if this.Persistent {
		key := this.EntityKey
		if key == "" {
			key = "user"
		}
		unconv := ctx.Value(key)
		if unconv != nil {
			user := unconv.(Flashable)
			if err := user.PersistFlash(ctx, *this); err != nil {
				return ctx, err
			}
		}
	}
	return context.WithValue(ctx, "flashes", flashes), nil
}

type FlashLevel int

const (
	Warn FlashLevel = 1 + iota
	Success
	Info
)
