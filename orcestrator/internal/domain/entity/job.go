package entity

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusPending      JobStatus = "pending"
	JobStatusRunning      JobStatus = "running"
	JobStatusFailed       JobStatus = "failed"
	JobStatusReady2Deploy JobStatus = "ready_to_deploy"
	JobStatusCanceled     JobStatus = "canceled"
	JobStatusDeploying    JobStatus = "deploying"
	JobStatusDeployed     JobStatus = "deployed"
)

type Job struct {
	ID          string    `json:"id" db:"id"`
	Description string    `json:"description" db:"description"`
	Target      string    `json:"target" db:"target"` // terraform, kubernetes, ansible
	Status      JobStatus `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func NewJob(description, target string) *Job {
	return &Job{
		ID:          uuid.New().String(),
		Description: description,
		Target:      target,
		Status:      JobStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (j *Job) UpdateStatus(status JobStatus) {
	j.Status = status
	j.UpdatedAt = time.Now()
}

func (j *Job) IsReadyForDeploy() bool {
	return j.Status == JobStatusReady2Deploy
}
