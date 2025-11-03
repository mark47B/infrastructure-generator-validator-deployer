package entity

import "time"

type GenerateRequest struct {
	Description string `json:"description" validate:"required"`
	Target      string `json:"target" validate:"required"`
}

type GenerateResponse struct {
	Files     []*ConfigFile `json:"files"`
	RequestID string        `json:"request_id"`
	CreatedAt time.Time     `json:"created_at"`
	Status    string        `json:"status"`
}
