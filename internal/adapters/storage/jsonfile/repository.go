package jsonfile

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"spotjspf/internal/domain/job"
)

const (
	WarningParse    string = "warning: could not parse existing state file"
	WarningCreate   string = "warning: could not create data dir"
	WarningMarshall string = "warning: could not marshal state"
	WarningWrite    string = "warning: could not write state file"
	WarningFinalize string = "warning: could not finalize state file"
)

type Repository struct {
	mu       sync.RWMutex
	jobs     []*job.Job
	maxItems int
	filePath string
}

func NewRepository(maxItems int, filePath string) *Repository {
	r := &Repository{maxItems: maxItems, filePath: filePath}
	r.load()
	return r
}

func (r *Repository) load() {
	if r.filePath == "" {
		return
	}
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return
	}
	var jobs []*job.Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		log.Printf("%s %s: %v", WarningParse, r.filePath, err)
		return
	}
	r.jobs = jobs
}

func (r *Repository) Add(j *job.Job) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = append([]*job.Job{j}, r.jobs...)
	if len(r.jobs) > r.maxItems {
		r.jobs = r.jobs[:r.maxItems]
	}
	r.persist()
}

func (r *Repository) Update(id string, fn func(*job.Job)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, j := range r.jobs {
		if j.ID == id {
			fn(j)
			break
		}
	}
	r.persist()
}

func (r *Repository) Get(id string) *job.Job {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, j := range r.jobs {
		if j.ID == id {
			cp := *j
			return &cp
		}
	}
	return nil
}

func (r *Repository) List() []*job.Job {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*job.Job, len(r.jobs))
	copy(out, r.jobs)
	return out
}

func (r *Repository) persist() {
	if r.filePath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0o755); err != nil {
		log.Printf("%s: %v", WarningCreate, err)
		return
	}
	data, err := json.MarshalIndent(r.jobs, "", "  ")
	if err != nil {
		log.Printf("%s: %v", WarningMarshall, err)
		return
	}
	tmp := r.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("%s: %v", WarningWrite, err)
		return
	}
	if err := os.Rename(tmp, r.filePath); err != nil {
		log.Printf("%s: %v", WarningFinalize, err)
	}
}
