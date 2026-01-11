package terraform

import "strings"

// StreamMessageType identifies the type of streaming JSON message.
type StreamMessageType string

const (
	MessageTypeVersion           StreamMessageType = "version"
	MessageTypePlannedChange     StreamMessageType = "planned_change"
	MessageTypeChangeSummary     StreamMessageType = "change_summary"
	MessageTypeApplyStart        StreamMessageType = "apply_start"
	MessageTypeApplyProgress     StreamMessageType = "apply_progress"
	MessageTypeApplyComplete     StreamMessageType = "apply_complete"
	MessageTypeApplyErrored      StreamMessageType = "apply_errored"
	MessageTypeProvisionStart    StreamMessageType = "provision_start"
	MessageTypeProvisionProgress StreamMessageType = "provision_progress"
	MessageTypeProvisionComplete StreamMessageType = "provision_complete"
	MessageTypeProvisionErrored  StreamMessageType = "provision_errored"
	MessageTypeRefreshStart      StreamMessageType = "refresh_start"
	MessageTypeRefreshComplete   StreamMessageType = "refresh_complete"
	MessageTypeInitOutput        StreamMessageType = "init_output"
	MessageTypeTestFile          StreamMessageType = "test_file"
	MessageTypeTestRun           StreamMessageType = "test_run"
	MessageTypeTestSummary       StreamMessageType = "test_summary"
	MessageTypeDiagnostic        StreamMessageType = "diagnostic"
	MessageTypeResourceDrift     StreamMessageType = "resource_drift"
	MessageTypeOutputs           StreamMessageType = "outputs"
)

// StreamMessage wraps a single streaming JSON event.
type StreamMessage struct {
	Type          StreamMessageType
	Version       *VersionInfo
	PlannedChange *PlannedChange
	ChangeSummary *ChangeSummary
	Hook          *HookMessage
	InitOutput    *InitOutputMessage
	TestFile      *TestFileMessage
	TestRun       *TestRunMessage
	TestSummary   *TestSummaryMessage
	Diagnostic    *Diagnostic
	ResourceDrift *ResourceDrift
	Outputs       map[string]Output
}

// VersionInfo captures terraform version metadata.
type VersionInfo struct {
	TerraformVersion string `json:"terraform_version"`
	ProtocolVersion  string `json:"protocol_version,omitempty"`
}

// ResourceInstance identifies a resource in streaming messages.
type ResourceInstance struct {
	Address      string        `json:"address"`
	ModulePath   ModuleAddress `json:"module_address,omitempty"`
	ResourceType string        `json:"type,omitempty"`
	ResourceName string        `json:"name,omitempty"`
	ProviderName string        `json:"provider_name,omitempty"`
}

// PlannedChange reports a single planned resource change.
type PlannedChange struct {
	Resource ResourceInstance `json:"resource"`
	Change   Change           `json:"change"`
}

// ChangeSummary reports aggregate changes.
type ChangeSummary struct {
	Changes       ChangeCounts `json:"changes"`
	ResourceDrift int          `json:"resource_drift,omitempty"`
	OutputChanges ChangeCounts `json:"outputs,omitempty"`
}

// ChangeCounts represents aggregate counts.
type ChangeCounts struct {
	Add    int `json:"add"`
	Change int `json:"change"`
	Remove int `json:"remove"`
}

// HookMessage reports apply progress events.
type HookMessage struct {
	Resource    ResourceInstance `json:"resource"`
	Address     string           `json:"address,omitempty"`
	Action      string           `json:"action,omitempty"`
	IDKey       string           `json:"id_key,omitempty"`
	IDValue     string           `json:"id_value,omitempty"`
	Provisioner string           `json:"provisioner,omitempty"`
	Output      string           `json:"output,omitempty"`
	Error       string           `json:"error,omitempty"`
	ElapsedSec  float64          `json:"elapsed_seconds,omitempty"`
}

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

// ResourceDrift reports detected drift.
type ResourceDrift struct {
	Resource ResourceInstance `json:"resource"`
	Change   Change           `json:"change"`
}

// Output represents output change data in streaming messages.
type Output struct {
	Actions   []string `json:"actions,omitempty"`
	Before    any      `json:"before,omitempty"`
	After     any      `json:"after,omitempty"`
	Sensitive bool     `json:"sensitive,omitempty"`
}

// InitOutputMessage captures init progress updates.
type InitOutputMessage struct {
	MessageCode string `json:"message_code,omitempty"`
}

// TestFileMessage reports progress for a test file.
type TestFileMessage struct {
	Path     string `json:"path"`
	Progress string `json:"progress"`
	Status   string `json:"status,omitempty"`
}

// TestRunMessage reports progress for a test run block.
type TestRunMessage struct {
	Path     string `json:"path"`
	Run      string `json:"run"`
	Progress string `json:"progress"`
	Status   string `json:"status,omitempty"`
	Elapsed  int64  `json:"elapsed,omitempty"`
}

// TestSummaryMessage reports overall test status.
type TestSummaryMessage struct {
	Status  string `json:"status"`
	Passed  int    `json:"passed"`
	Failed  int    `json:"failed"`
	Errored int    `json:"errored"`
	Skipped int    `json:"skipped"`
}

// ParseActionType normalizes action strings into ActionType values.
func ParseActionType(action string) ActionType {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "create":
		return ActionCreate
	case "delete", "destroy", "remove":
		return ActionDelete
	case "update", "change":
		return ActionUpdate
	case "replace":
		return ActionReplace
	case "read":
		return ActionRead
	case "no-op", "noop":
		return ActionNoOp
	default:
		return ActionNoOp
	}
}
