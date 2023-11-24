package models

import (
	scumsearch "github.com/davidbanham/scum/search"
)

type SearchResult = scumsearch.SearchResult
type SearchResults = scumsearch.SearchResults
type SearchCriteria = scumsearch.SearchCriteria

var BasicRoleCheck = scumsearch.BasicRoleCheck

type ByPhrase = scumsearch.ByPhrase
