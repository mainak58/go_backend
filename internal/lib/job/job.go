package job

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
)

type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityDefault  Priority = "default"
	PriorityLow      Priority = "low"
)

type Job struct {
	Name     string
	Handler  func() error
	Priority Priority
}

type JobService struct {
	critical chan Job
	defaultQ chan Job
	low      chan Job

	logger *zerolog.Logger
	wg     sync.WaitGroup
}

func NewJobService(logger *zerolog.Logger) *JobService {
	return &JobService{
		critical: make(chan Job, 10),
		defaultQ: make(chan Job, 10),
		low:      make(chan Job, 10),
		logger:   logger,
	}
}

func (j *JobService) Enque(job Job) {
	switch job.Priority {
	case PriorityCritical:
		j.critical <- job
	case PriorityDefault:
		j.defaultQ <- job
	case PriorityLow:
		j.low <- job
	}
}

func (j *JobService) worker(ctx context.Context) {
	defer j.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case job := <-j.critical:
			j.execute(job)

		default:
			select {
			case job := <-j.defaultQ:
				j.execute(job)
			case job := <-j.low:
				j.execute(job)
			case <-ctx.Done():
				return
			}
		}
	}
}

func (j *JobService) execute(job Job) {
	if err := job.Handler(); err != nil {
		j.logger.Error().
			Err(err).
			Str("job", job.Name).
			Msg("job failed")
	}
}

func (j *JobService) Start(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		j.wg.Add(1)
		go j.worker(ctx)
	}
}

func (j *JobService) Stop() {
	close(j.critical)
	close(j.defaultQ)
	close(j.low)
	j.wg.Wait()
}
