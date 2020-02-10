package routes

import (
	"doubleboiler/config"

	"github.com/gorilla/securecookie"
)

var secureCookie = securecookie.New([]byte(config.HASH_KEY), []byte(config.BLOCK_KEY))
