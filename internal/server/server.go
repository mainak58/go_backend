package server

import (
	"net/http"

	zerolog "github.com/jackc/pgx-zerolog"
	"github.com/mainak58/go_backend/internal/config"
	"github.com/mainak58/go_backend/internal/database"
	"github.com/mainak58/go_backend/internal/lib/job"
)

type Server struct {
	Config     *config.Config
	Logger     *zerolog.Logger
	DB         *database.Database
	httpServer *http.Server
	Job        *job.JobService
}
