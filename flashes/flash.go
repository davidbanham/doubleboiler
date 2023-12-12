package flashes

import (
	scumflashes "github.com/davidbanham/scum/flash"
)

type Flash = scumflashes.Flash
type Flashes = scumflashes.Flashes
type FlashAction = scumflashes.FlashAction
type FlashLevel = scumflashes.FlashLevel
type Flashable = scumflashes.Flashable

const (
	Warn FlashLevel = 1 + iota
	Success
	Info
)
