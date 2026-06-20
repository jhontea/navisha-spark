package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// Scheduler wraps the cron scheduler with additional functionality.
type Scheduler struct {
	cron        *cron.Cron
	jobs        map[string]cron.EntryID
	mu          sync.RWMutex
	log         *logrus.Entry
	timezone    *time.Location
	activeStart int
	activeEnd   int
	running     bool
}

// Config holds scheduler configuration.
type Config struct {
	Cron        string
	Timezone    string
	ActiveStart int
	ActiveEnd   int
}

// New creates a new Scheduler.
func New(cfg Config, log *logrus.Entry) (*Scheduler, error) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", cfg.Timezone, err)
	}

	c := cron.New(
		cron.WithLocation(loc),
		cron.WithLogger(cron.PrintfLogger(log)),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.PrintfLogger(log)),
			cron.Recover(cron.PrintfLogger(log)),
		),
	)

	log.WithFields(logrus.Fields{
		"timezone": cfg.Timezone,
	}).Info("scheduler initialized")

	return &Scheduler{
		cron:        c,
		jobs:        make(map[string]cron.EntryID),
		log:         log,
		timezone:    loc,
		activeStart: cfg.ActiveStart,
		activeEnd:   cfg.ActiveEnd,
	}, nil
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.log.Warn("scheduler already running")
		return
	}

	s.cron.Start()
	s.running = true
	s.log.Info("scheduler started")
}

// Stop stops the scheduler gracefully.
func (s *Scheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		s.log.Warn("scheduler not running")
		return nil
	}

	ctx := s.cron.Stop()
	s.running = false
	s.log.Info("scheduler stopped")
	return ctx
}

// AddJob adds a new job to the scheduler.
func (s *Scheduler) AddJob(name, expression string, job cron.FuncJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if job already exists
	if _, exists := s.jobs[name]; exists {
		return fmt.Errorf("job %s already exists", name)
	}

	entryID, err := s.cron.AddFunc(expression, func() {
		s.runJob(name, job)
	})
	if err != nil {
		return fmt.Errorf("failed to add job %s: %w", name, err)
	}

	s.jobs[name] = entryID
	s.log.WithFields(logrus.Fields{
		"job":        name,
		"expression": expression,
	}).Info("job added to scheduler")

	return nil
}

// RemoveJob removes a job from the scheduler.
func (s *Scheduler) RemoveJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job %s not found", name)
	}

	s.cron.Remove(entryID)
	delete(s.jobs, name)

	s.log.WithFields(logrus.Fields{
		"job": name,
	}).Info("job removed from scheduler")

	return nil
}

// runJob runs a job with active hours check.
func (s *Scheduler) runJob(name string, job cron.FuncJob) {
	if !s.isActiveHours() {
		s.log.WithFields(logrus.Fields{
			"job":          name,
			"current_hour": time.Now().In(s.timezone).Hour(),
		}).Debug("skipping job outside active hours")
		return
	}

	s.log.WithFields(logrus.Fields{
		"job": name,
	}).Info("running job")

	job()
}

// isActiveHours checks if the current time is within active hours.
func (s *Scheduler) isActiveHours() bool {
	now := time.Now().In(s.timezone)
	hour := now.Hour()

	if s.activeStart <= s.activeEnd {
		return hour >= s.activeStart && hour <= s.activeEnd
	}

	// Handle overnight active hours (e.g., 22:00 - 06:00)
	return hour >= s.activeStart || hour <= s.activeEnd
}

// GetActiveJobs returns the list of active job names.
func (s *Scheduler) GetActiveJobs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]string, 0, len(s.jobs))
	for name := range s.jobs {
		jobs = append(jobs, name)
	}

	return jobs
}

// GetNextRun returns the next scheduled run time for a job.
func (s *Scheduler) GetNextRun(name string) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entryID, exists := s.jobs[name]
	if !exists {
		return time.Time{}, fmt.Errorf("job %s not found", name)
	}

	entry := s.cron.Entry(entryID)
	if entry.Next.IsZero() {
		return time.Time{}, fmt.Errorf("job %s has no next run time", name)
	}

	return entry.Next, nil
}

// IsRunning checks if the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetEntry returns a cron entry by job name.
func (s *Scheduler) GetEntry(name string) (cron.Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entryID, exists := s.jobs[name]
	if !exists {
		return cron.Entry{}, fmt.Errorf("job %s not found", name)
	}

	return s.cron.Entry(entryID), nil
}
