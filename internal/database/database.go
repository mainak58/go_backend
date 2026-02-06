package database

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"time"

	pgxzero "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/mainak58/go_backend/internal/config"
	loggerConfig "github.com/mainak58/go_backend/internal/logger"
	"github.com/rs/zerolog"
)

type Database struct {
	Pool *pgxpool.Pool
	log  *zerolog.Logger
}

type multiTracer struct {
	tracers []any
}

func (mt *multiTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, tracer := range mt.tracers {
		if t, ok := tracer.(interface {
			TraceQueryStart(context.Context, *pgx.Conn, pgx.TraceQueryStartData) context.Context
		}); ok {
			ctx = t.TraceQueryStart(ctx, conn, data)
		}
	}
	return ctx
}

// TraceQueryEnd implements pgx tracer interface
func (mt *multiTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, tracer := range mt.tracers {
		if t, ok := tracer.(interface {
			TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData)
		}); ok {
			t.TraceQueryEnd(ctx, conn, data)
		}
	}
}

const DatabasePingTime = 10

func New(cfg *config.Config, logger *zerolog.Logger, loggerService *loggerConfig.LoggerService) (*Database, error) {
	hostPort := net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))
	encodedPassword := url.QueryEscape(cfg.Database.Password)

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Database.User,
		encodedPassword,
		hostPort,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx pool config: %w", err)
	}

	if cfg.Primary.Env == "local" {
		globalLevel := logger.GetLevel()
		pgxLogger := loggerConfig.NewPgxLogger(globalLevel)
		// Chain tracers - New Relic first, then local logging
		if config.ConnConfig.Tracer != nil {
			// If New Relic tracer exists, create a multi-tracer
			localTracer := &tracelog.TraceLog{
				Logger:   pgxzero.NewLogger(pgxLogger),
				LogLevel: tracelog.LogLevel(loggerConfig.GetPgxTraceLogLevel(globalLevel)),
			}
			config.ConnConfig.Tracer = &multiTracer{
				tracers: []any{config.ConnConfig.Tracer, localTracer},
			}
		} else {
			config.ConnConfig.Tracer = &tracelog.TraceLog{
				Logger:   pgxzero.NewLogger(pgxLogger),
				LogLevel: tracelog.LogLevel(loggerConfig.GetPgxTraceLogLevel(globalLevel)),
			}
		}
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}

	database := &Database{
		Pool: pool,
		log:  logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), DatabasePingTime*time.Second)
	defer cancel()
	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().Msg("connected to the database")

	return database, nil
}
