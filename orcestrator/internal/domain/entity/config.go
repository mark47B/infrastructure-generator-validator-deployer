package entity

type ConfigFile struct {
	JobID    string                 `json:"job_id"`
	Name     string                 `json:"name"`
	Content  string                 `json:"content"`
	Type     string                 `json:"type"` // terraform, kubernetes, ansible;
	HasError bool                   `json:"has_error"`
	ErrorMsg *ValidationConfigError `json:"error_msg,omitempty"`
}

type ValidationConfigError struct {
	File    string `json:"file"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}
