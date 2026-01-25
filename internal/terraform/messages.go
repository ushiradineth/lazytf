package terraform

// Diagnostic reports warnings or errors.
type Diagnostic struct {
	Severity string           `json:"severity"`
	Summary  string           `json:"summary"`
	Detail   string           `json:"detail,omitempty"`
	Address  string           `json:"address,omitempty"`
	Range    *DiagnosticRange `json:"range,omitempty"`
}

// DiagnosticRange describes a source location.
type DiagnosticRange struct {
	Filename string        `json:"filename,omitempty"`
	Start    *LinePosition `json:"start,omitempty"`
	End      *LinePosition `json:"end,omitempty"`
}

// LinePosition indicates a line/column.
type LinePosition struct {
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}
