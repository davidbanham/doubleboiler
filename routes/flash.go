package routes

import "context"

type Flash struct {
	Lines   []string
	Actions []FlashAction
	Type    FlashLevel
}

type FlashAction struct {
	Url  string
	Text string
}

func (this Flash) Add(ctx context.Context) context.Context {
	flashes := []Flash{}
	unconv := ctx.Value("flashes")
	if unconv != nil {
		flashes = unconv.([]Flash)
	}
	flashes = append(flashes, this)
	return context.WithValue(ctx, "flashes", flashes)
}

type FlashLevel int

const (
	Warn FlashLevel = 1 + iota
	Success
	Info
)
