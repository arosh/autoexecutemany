package autoexecutemany

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		query string
		want  []string
	}{
		{
			query: "INSERT t1 INTO (c1, c2) VALUES (?, ?)",
			want:  []string{"INSERT t1 INTO (c1, c2) VALUES ", "(?, ?)", ""},
		},
		{
			query: "INSERT t1 INTO (c1, c2) VALUES (?, ?) ON DUPLICATE KEY UPDATE c2=VALUES(c2)",
			want:  []string{"INSERT t1 INTO (c1, c2) VALUES ", "(?, ?)", " ON DUPLICATE KEY UPDATE c2=VALUES(c2)"},
		},
		{
			query: "insert t1 into (c1, c2) values (?, ?) on duplicate key update c2=values(c2)",
			want:  []string{"insert t1 into (c1, c2) values ", "(?, ?)", " on duplicate key update c2=values(c2)"},
		},
		{
			query: "INSERT t1 INTO (c1, c2) VALUES (?, NOW())",
			want:  []string{"INSERT t1 INTO (c1, c2) VALUES ", "(?, NOW())", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := ParseQuery(tt.query)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestExecuteMany(t *testing.T) {
	type exec struct {
		query string
		args  []interface{}
	}
	tests := []struct {
		name      string
		query     string
		args      [][]interface{}
		batchSize int
		execs     []exec
	}{
		{
			name:      "5/3",
			query:     `INSERT INTO t1(c1) VALUES (?)`,
			args:      [][]interface{}{{1}, {2}, {3}, {4}, {5}},
			batchSize: 3,
			execs: []exec{
				{
					query: `INSERT INTO t1(c1) VALUES (?), (?), (?)`,
					args:  []interface{}{1, 2, 3},
				},
				{
					query: `INSERT INTO t1(c1) VALUES (?), (?)`,
					args:  []interface{}{4, 5},
				},
			},
		},
		{
			name:      "1/3",
			query:     `INSERT INTO t1(c1) VALUES (?)`,
			args:      [][]interface{}{{1}},
			batchSize: 3,
			execs: []exec{
				{
					query: `INSERT INTO t1(c1) VALUES (?)`,
					args:  []interface{}{1},
				},
			},
		},
		{
			name:      "6/3",
			query:     `INSERT INTO t1(c1) VALUES (?)`,
			args:      [][]interface{}{{1}, {2}, {3}, {4}, {5}, {6}},
			batchSize: 3,
			execs: []exec{
				{
					query: `INSERT INTO t1(c1) VALUES (?), (?), (?)`,
					args:  []interface{}{1, 2, 3},
				},
				{
					query: `INSERT INTO t1(c1) VALUES (?), (?), (?)`,
					args:  []interface{}{4, 5, 6},
				},
			},
		},
		{
			name:      "0/3",
			query:     `INSERT INTO t1(c1) VALUES (?)`,
			args:      [][]interface{}{},
			batchSize: 3,
			execs:     nil,
		},
		{
			name: "5/3 (?, UTC_TIMESTAMP())",
			query: `
			INSERT INTO t1(c1, c2) VALUES(?, UTC_TIMESTAMP())
			ON DUPLICATE KEY UPDATE c2=VALUES(c2)
			`,
			args:      [][]interface{}{{1}, {2}, {3}, {4}, {5}},
			batchSize: 3,
			execs: []exec{
				{
					query: `INSERT INTO t1(c1, c2) VALUES(?, UTC_TIMESTAMP()), (?, UTC_TIMESTAMP()), (?, UTC_TIMESTAMP()) ON DUPLICATE KEY UPDATE c2=VALUES(c2)`,
					args:  []interface{}{1, 2, 3},
				},
				{
					query: `INSERT INTO t1(c1, c2) VALUES(?, UTC_TIMESTAMP()), (?, UTC_TIMESTAMP()) ON DUPLICATE KEY UPDATE c2=VALUES(c2)`,
					args:  []interface{}{4, 5},
				},
			},
		},
		{
			name: "5/3 (?, ?)",
			query: `
			INSERT INTO t1(c1, c2) VALUES(?, ?)
			ON DUPLICATE KEY UPDATE c2=VALUES(c2)
			`,
			args:      [][]interface{}{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}},
			batchSize: 3,
			execs: []exec{
				{
					query: `INSERT INTO t1(c1, c2) VALUES(?, ?), (?, ?), (?, ?) ON DUPLICATE KEY UPDATE c2=VALUES(c2)`,
					args:  []interface{}{1, 1, 2, 2, 3, 3},
				},
				{
					query: `INSERT INTO t1(c1, c2) VALUES(?, ?), (?, ?) ON DUPLICATE KEY UPDATE c2=VALUES(c2)`,
					args:  []interface{}{4, 4, 5, 5},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectBegin()
			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			for _, exc := range tt.execs {
				var args []driver.Value
				for _, arg := range exc.args {
					args = append(args, arg)
				}
				dummyResult := sqlmock.NewResult(0, 0)
				mock.ExpectExec(regexp.QuoteMeta(exc.query)).WithArgs(args...).WillReturnResult(dummyResult)
			}
			if err := ExecuteMany(tx, tt.query, tt.args, tt.batchSize); err != nil {
				t.Fatal(err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
