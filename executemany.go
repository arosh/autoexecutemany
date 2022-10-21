package autoexecutemany

import (
	"database/sql"
	"regexp"
	"strings"
)

var RegexpInsertValues = regexp.MustCompile(`(?is)\s*((?:INSERT|REPLACE)\b.+\bVALUES?\s*)` +
	`(\(\s*[A-Za-z_()?]+\s*(?:,\s*[A-Za-z_()?]+\s*)*\))` +
	`(\s*(?:ON DUPLICATE.*)?);?\s*\z`)

func ParseQuery(query string) []string {
	group := RegexpInsertValues.FindStringSubmatch(query)
	if group == nil {
		return nil
	}
	return group[1:]
}

func ExecuteMany(tx *sql.Tx, query string, args [][]interface{}, batchSize int) error {
	if args == nil {
		return nil
	}
	m := ParseQuery(query)
	if m == nil {
		for _, arg := range args {
			if _, err := tx.Exec(query, arg...); err != nil {
				return err
			}
		}
		return nil
	}

	prefix := m[0]
	values := m[1]
	postfix := m[2]

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
