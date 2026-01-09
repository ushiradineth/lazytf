package terraform

import (
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
