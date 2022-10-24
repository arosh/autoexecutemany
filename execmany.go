package autoexecutemany

import (
	"database/sql"
	"strings"
)

func ExecMany(tx *sql.Tx, prefix, values, postfix string, args [][]interface{}, batchSize int) error {
	for i := 0; i < len(args); i += batchSize {
		end := i + batchSize
		if end > len(args) {
			end = len(args)
		}
		nArgs := end - i

		sql := prefix + values + strings.Repeat(", "+values, nArgs-1) + postfix
		var params []interface{}
		for _, arg := range args[i:end] {
			params = append(params, arg...)
		}
		if _, err := tx.Exec(sql, params...); err != nil {
			return err
		}
	}
	return nil
}
