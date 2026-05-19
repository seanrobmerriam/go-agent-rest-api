// store.go is for async job storage and retrieval.
package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

type Job struct {
	ID        string    `json:"id"`
	Tool      string    `json:"tool"`
	Status    Status    `json:"status"`
	Result    any       `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewStore() *Store {
	return &Store{jobs: make(map[string]*Job)}
}

func (s *Store) Create(tool string) *Job {
	job := &Job{
		ID:        uuid.NewString(),
		Tool:      tool,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	return job
}

func (s *Store) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

func (s *Store) SetRunning(id string) {
	s.update(id, func(j *Job) {
		j.Status = StatusRunning
	})
}

func (s *Store) SetDone(id string, result any) {
	s.update(id, func(j *Job) {
		j.Status = StatusDone
		j.Result = result
	})
}

func (s *Store) SetFailed(id string, err error) {
	s.update(id, func(j *Job) {
		j.Status = StatusFailed
		j.Error = err.Error()
	})
}

func (s *Store) update(id string, fn func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		fn(j)
		j.UpdatedAt = time.Now()
	}
}

// Run executes fn in a goroutine and tracks the job through its lifecycle.
func (s *Store) Run(ctx context.Context, id string, fn func(ctx context.Context) (any, error)) {
	s.SetRunning(id)
	go func() {
		result, err := fn(ctx)
		if err != nil {
			s.SetFailed(id, err)
			return
		}
		s.SetDone(id, result)
	}()
}
