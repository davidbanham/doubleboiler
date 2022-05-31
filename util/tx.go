package util

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/logger"
	"fmt"
)

func GetTxCtx() (context.Context, *sql.Tx, error) {
	var tx *sql.Tx
	var err error

	ctx := context.Background()

	tx, err = config.Db.BeginTx(ctx, nil)
	if err != nil {
		return ctx, nil, err
	}
	tx.ExecContext(ctx, "SET application_name = 'system_user'")

	ctx = context.WithValue(ctx, "tx", tx)

	return ctx, tx, err
}

func RollbackTx(ctx context.Context) {
	tx := ctx.Value("tx")
	switch v := tx.(type) {
	case *sql.Tx:
		rollbackErr := v.Rollback()
		if rollbackErr != nil {
			logger.Log(ctx, logger.Error, fmt.Sprintf("Error rolling back tx: %+v", rollbackErr))
		}
	default:
		//fmt.Printf("DEBUG no transaction on context\n")
	}
}
