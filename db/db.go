package db

import (
	"context"
	"fmt"
	"io"

	"github.com/koshoi/selektor/config"
	"github.com/koshoi/selektor/db/clickhouse"
	"github.com/koshoi/selektor/db/options"
	"github.com/koshoi/selektor/db/postgres"
)

type Queriable interface {
	Query(ctx context.Context, query string, opt options.Options) (io.Reader, error)
}

func NewQueriable(cfg config.EnvConfig) (Queriable, error) {
	switch cfg.Type {
	case config.DBClickHouse:
		return &clickhouse.CHClient{Env: cfg}, nil
	case config.DBPostgres, config.DBType(""):
		return &postgres.PGClient{Env: cfg}, nil
	}

	return nil, fmt.Errorf("unknown env type='%s'", cfg.Type)
}
