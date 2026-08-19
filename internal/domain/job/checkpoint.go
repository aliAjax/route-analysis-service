package job

import (
	"errors"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// Checkpoint is a resumable cursor for long-running matrix jobs.
type Checkpoint struct {
	JobID      model.JobID  `json:"jobId"`
	LastOrigin model.NodeID `json:"lastOrigin"`
	LastDest   model.NodeID `json:"lastDestination"`
	Completed  int          `json:"completed"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}

func (c Checkpoint) Validate() error {
	if c.JobID == "" || c.Completed < 0 {
		return errors.New("invalid checkpoint")
	}
	return nil
}

// ResumeCursor converts a checkpoint into the next (origin, destination) pair
// to compute. An empty origin means the job starts from the beginning.
type ResumeCursor struct {
	NextOrigin model.NodeID
	NextDest   model.NodeID
	Done       bool
}

// Cursor returns the resume cursor derived from this checkpoint.
func (c Checkpoint) Cursor() ResumeCursor {
	return ResumeCursor{NextOrigin: c.LastOrigin, NextDest: c.LastDest}
}

// Advance moves the checkpoint forward by one origin/destination pair.
func (c *Checkpoint) Advance(origin, dest model.NodeID, total int) {
	c.LastOrigin = origin
	c.LastDest = dest
	c.Completed++
	if c.Completed > total {
		c.Completed = total
	}
	c.UpdatedAt = time.Now().UTC()
}

// Persist captures a checkpoint as a JSON-friendly map for job results.
func (c Checkpoint) Persist() map[string]any {
	return map[string]any{
		"jobId":      string(c.JobID),
		"lastOrigin": string(c.LastOrigin),
		"lastDest":   string(c.LastDest),
		"completed":  c.Completed,
		"updatedAt":  c.UpdatedAt,
	}
}

// Restore rebuilds a checkpoint from persisted job parameters.
func Restore(jobID model.JobID, data map[string]any) Checkpoint {
	ck := Checkpoint{JobID: jobID, UpdatedAt: time.Now().UTC()}
	if v, ok := data["lastOrigin"].(string); ok {
		ck.LastOrigin = model.NodeID(v)
	}
	if v, ok := data["lastDest"].(string); ok {
		ck.LastDest = model.NodeID(v)
	}
	if v, ok := data["completed"].(float64); ok {
		ck.Completed = int(v)
	}
	return ck
}
