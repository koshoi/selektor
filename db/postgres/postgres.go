package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/koshoi/selektor/config"
	"github.com/koshoi/selektor/db/options"
)

const templateDSN = "postgres://%s:%s@%s/%s?sslmode=disable"

type PGClient struct {
	Env config.EnvConfig
}

func convert(v interface{}, t sql.ColumnType) interface{} {
	if v == nil {
		return v
	}

	tName := t.DatabaseTypeName()

	switch tName {
	case "JSONB":
		var vv interface{}
		err := json.Unmarshal(v.([]byte), &vv)
		if err != nil {
			return v
		}

		return vv

	case "NUMERIC":
		barr, ok := v.([]byte)
		if !ok {
			return v
		}

		s := string(barr)
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return s
		}

		return f

	case "TEXT", "TIMESTAMP", "INT4", "BOOL":
		return v
	}

	byteArr, ok := v.([]byte)
	if !ok {
		return v
	}

	return string(byteArr)
}

func (pgc *PGClient) Query(ctx context.Context, query string, opt options.Options) (io.Reader, error) {
	opt.Debugf("Executing query: %s", query)

	dsn := fmt.Sprintf(templateDSN, pgc.Env.User, pgc.Env.Password, pgc.Env.Endpoint, pgc.Env.Database)
	opt.Debugf("DSN='%s'", dsn)

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	res, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %s", err)
	}

	cols, err := res.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %s", err)
	}

	types, err := res.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %s", err)
	}

	pretty := make([]map[string]interface{}, 0, len(cols))
	for res.Next() {
		out := make([]interface{}, len(cols))
		scanable := make([]interface{}, len(cols))
		for i := 0; i < len(cols); i++ {
			scanable[i] = &out[i]
		}
		res.Scan(scanable...)
		resstruct := make(map[string]interface{})
		for i, c := range cols {
			resstruct[c] = convert(out[i], *types[i])
		}
		pretty = append(pretty, resstruct)
	}

	result, err := json.Marshal(pretty)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %s", err)
	}

	return bytes.NewReader(result), nil
}
