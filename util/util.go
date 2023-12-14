package util

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/logger"
	"fmt"

	scumutil "github.com/davidbanham/scum/util"
)

type Period = scumutil.Period

var IsBetween = scumutil.IsBetween
var DaysBetween = scumutil.DaysBetween
var FirstFiveChars = scumutil.FirstFiveChars
var FirstNonEmptyString = scumutil.FirstNonEmptyString
var StripBlankStrings = scumutil.StripBlankStrings
var CalcExpiry = scumutil.CalcExpiry
var CalcToken = scumutil.CalcToken
var CheckToken = scumutil.CheckToken
var Uniq = scumutil.Uniq
var NextDay = scumutil.NextDay
var IsWeekday = scumutil.IsWeekday
var IsWeekend = scumutil.IsWeekend
var Prefix = scumutil.Prefix
var HashPassword = scumutil.HashPassword
var PrettyJsonString = scumutil.PrettyJsonString
var Diff = scumutil.Diff
var DiffOnly = scumutil.DiffOnly
var Hash = scumutil.Hash
var Contains = scumutil.Contains
var NextFlow = scumutil.NextFlow
var RootPath = scumutil.RootPath

func GetTxCtx() (context.Context, *sql.Tx, error) {
	return scumutil.GetTxCtx(config.Db)
}

func RollbackTx(ctx context.Context) {
	if err := scumutil.RollbackTx(ctx); err != nil {
		logger.Log(ctx, logger.Error, fmt.Sprintf("Error rolling back tx: %+v", err.Error()))
	}
}
