package terraform

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// OperationStatus reflects the lifecycle state of a resource operation.
type OperationStatus string

const (
	StatusPending    OperationStatus = "pending"
	StatusInProgress OperationStatus = "in_progress"
	StatusComplete   OperationStatus = "complete"
	StatusErrored    OperationStatus = "errored"
)

// ResourceOperation tracks the lifecycle of a single resource.
type ResourceOperation struct {
	Address     string
	Action      ActionType
	Status      OperationStatus
	StartTime   time.Time
	EndTime     time.Time
	ElapsedTime time.Duration
	Error       string
	IDValue     string
}

// OperationState tracks operation progress and diagnostics.
type OperationState struct {
	resources      map[string]*ResourceOperation
	diagnostics    []Diagnostic
	currentAddress string
	currentAction  ActionType
	totalResources int
	completed      int
	mu             sync.RWMutex
}

// NewOperationState creates an empty operation state.
func NewOperationState() *OperationState {
	return &OperationState{
		resources: make(map[string]*ResourceOperation),
	}
}

// InitializeFromPlan sets up pending operations based on a plan.
func (s *OperationState) InitializeFromPlan(plan *Plan) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resources = make(map[string]*ResourceOperation)
	s.diagnostics = nil
	s.currentAddress = ""
	s.currentAction = ActionNoOp
	s.completed = 0
	s.totalResources = 0

	if plan == nil {
		return
	}

	for _, resource := range plan.Resources {
		if resource.Address == "" {
			continue
		}
		if resource.Action == ActionNoOp {
			continue
		}
		s.resources[resource.Address] = &ResourceOperation{
			Address: resource.Address,
			Action:  resource.Action,
			Status:  StatusPending,
		}
		s.totalResources++
	}
}

// StartResource marks a resource as in-progress.
func (s *OperationState) StartResource(address string, action ActionType) {
	if s == nil || address == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.resources[address]
	if !ok {
		op = &ResourceOperation{Address: address}
		s.resources[address] = op
		s.totalResources++
	}
	// Don't reset status if resource has already errored
	// (Complete can transition back to InProgress for replace operations)
	if op.Status == StatusErrored {
		return
	}
	op.Action = action
	op.Status = StatusInProgress
	op.StartTime = time.Now()
	op.EndTime = time.Time{}
	op.ElapsedTime = 0
	op.Error = ""
	s.currentAddress = address
	s.currentAction = action
}

// CompleteResource marks a resource as complete.
func (s *OperationState) CompleteResource(address, idValue string) {
	if s == nil || address == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.resources[address]
	if !ok {
		op = &ResourceOperation{Address: address}
		s.resources[address] = op
		s.totalResources++
	}
	// Don't overwrite errored status
	if op.Status == StatusErrored {
		return
	}
	wasDone := op.Status == StatusComplete || op.Status == StatusErrored
	op.Status = StatusComplete
	op.EndTime = time.Now()
	if !op.StartTime.IsZero() {
		op.ElapsedTime = op.EndTime.Sub(op.StartTime)
	}
	op.IDValue = idValue
	if !wasDone {
		s.completed++
	}
	if s.currentAddress == address {
		s.currentAddress = ""
		s.currentAction = ActionNoOp
	}
}

// ErrorResource marks a resource as errored.
func (s *OperationState) ErrorResource(address string, err error) {
	if s == nil || address == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	op, ok := s.resources[address]
	if !ok {
		op = &ResourceOperation{Address: address}
		s.resources[address] = op
		s.totalResources++
	}
	wasDone := op.Status == StatusComplete || op.Status == StatusErrored
	op.Status = StatusErrored
	op.EndTime = time.Now()
	if !op.StartTime.IsZero() {
		op.ElapsedTime = op.EndTime.Sub(op.StartTime)
	}
	if err != nil {
		op.Error = err.Error()
	}
	if !wasDone {
		s.completed++
	}
	if s.currentAddress == address {
		s.currentAddress = ""
		s.currentAction = ActionNoOp
	}
}

// AddDiagnostic records a diagnostic message.
func (s *OperationState) AddDiagnostic(diag Diagnostic) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.diagnostics = append(s.diagnostics, diag)
}

// GetProgress returns current progress snapshot.
func (s *OperationState) GetProgress() (current, total int, currentAddress string, currentAction ActionType) {
	if s == nil {
		return 0, 0, "", ActionNoOp
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.completed, s.totalResources, s.currentAddress, s.currentAction
}

// GetResourceStatus returns state for a resource.
func (s *OperationState) GetResourceStatus(address string) *ResourceOperation {
	if s == nil || address == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	op, ok := s.resources[address]
	if !ok {
		return nil
	}
	copyOp := *op
	return &copyOp
}

// GetDiagnostics returns all diagnostics.
func (s *OperationState) GetDiagnostics() []Diagnostic {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Diagnostic, len(s.diagnostics))
	copy(out, s.diagnostics)
	return out
}

// Patterns for parsing terraform apply output.
var (
	// resourceStartPattern matches lines like "null_resource.example: Creating...".
	resourceStartPattern = regexp.MustCompile(`^(\S+): (Creating|Destroying|Modifying|Reading|Refreshing)\.\.\.`)
	// resourceCompletePattern matches lines like "null_resource.example: Creation complete after 0s [id=123]".
	resourceCompletePattern = regexp.MustCompile(`^(\S+): (Creation|Destruction|Modifications|Read) complete`)
	// idPattern matches id value in brackets: [id=123].
	idPattern = regexp.MustCompile(`\[id=([^\]]*)\]`)
	// ansiPattern matches ANSI escape sequences.
	ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[mGKHF]`)
)

// ParseApplyLine parses a line of terraform apply output and updates state.
func (s *OperationState) ParseApplyLine(line string) {
	if s == nil {
		return
	}

	// Strip ANSI escape codes
	line = ansiPattern.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// Check for resource start
	if matches := resourceStartPattern.FindStringSubmatch(line); matches != nil {
		address := matches[1]
		actionStr := matches[2]
		action := parseActionFromVerb(actionStr)
		s.StartResource(address, action)
		return
	}

	// Check for resource complete
	if matches := resourceCompletePattern.FindStringSubmatch(line); matches != nil {
		address := matches[1]
		idValue := ""
		// Extract id from [id=...] if present
		if idMatches := idPattern.FindStringSubmatch(line); idMatches != nil {
			idValue = idMatches[1]
		}
		s.CompleteResource(address, idValue)
		return
	}

	// Check for error - either "Error:" or "Apply failed:"
	// Use Contains because error messages may have leading box drawing characters (│)
	if strings.Contains(line, "Error:") || strings.HasPrefix(line, "Apply failed:") {
		s.markLastInProgressAsErrored()
		return
	}

	// "Still creating..." lines don't change state, resource is already in progress
}

// markLastInProgressAsErrored marks the current in-progress resource as errored.
func (s *OperationState) markLastInProgressAsErrored() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find any in-progress resource and mark it as errored
	for _, op := range s.resources {
		if op.Status == StatusInProgress {
			op.Status = StatusErrored
			op.EndTime = time.Now()
			if !op.StartTime.IsZero() {
				op.ElapsedTime = op.EndTime.Sub(op.StartTime)
			}
			s.completed++
			if s.currentAddress == op.Address {
				s.currentAddress = ""
				s.currentAction = ActionNoOp
			}
			return
		}
	}
}

// parseActionFromVerb converts terraform output verbs to ActionType.
func parseActionFromVerb(verb string) ActionType {
	switch strings.ToLower(verb) {
	case "creating":
		return ActionCreate
	case "destroying":
		return ActionDelete
	case "modifying":
		return ActionUpdate
	case "reading", "refreshing":
		return ActionRead
	default:
		return ActionNoOp
	}
}
