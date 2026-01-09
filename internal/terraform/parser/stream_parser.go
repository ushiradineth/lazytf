package parser

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"strings"

	"github.com/ushiradineth/tftui/internal/terraform"
)

// StreamParser parses line-delimited JSON terraform output.
type StreamParser struct {
	accumulator *planAccumulator
}

// NewStreamParser creates a streaming JSON parser.
func NewStreamParser() *StreamParser {
	return &StreamParser{
		accumulator: newPlanAccumulator(),
	}
}

// Parse reads line-delimited JSON and emits typed messages.
func (p *StreamParser) Parse(input io.Reader, msgChan chan<- terraform.StreamMessage) error {
	if p == nil {
		return nil
	}
	scanner := bufio.NewScanner(input)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		msg, err := parseStreamMessage([]byte(line))
		if err != nil {
			log.Printf("stream parse error: %v", err)
			continue
		}

		if p.accumulator != nil {
			p.accumulator.apply(msg)
		}

		if msgChan != nil {
			select {
			case msgChan <- msg:
			default:
			}
		}
	}
	return scanner.Err()
}

// GetAccumulatedPlan returns the plan built from streamed messages.
func (p *StreamParser) GetAccumulatedPlan() *terraform.Plan {
	if p == nil || p.accumulator == nil {
		return nil
	}
	return p.accumulator.snapshot()
}

type baseMessage struct {
	Type terraform.StreamMessageType `json:"type"`
}

func parseStreamMessage(line []byte) (terraform.StreamMessage, error) {
	var base baseMessage
	if err := json.Unmarshal(line, &base); err != nil {
		return terraform.StreamMessage{}, err
	}

	msg := terraform.StreamMessage{Type: base.Type}
	switch base.Type {
	case terraform.MessageTypeVersion:
		var payload struct {
			TerraformVersion string `json:"terraform_version"`
			ProtocolVersion  string `json:"protocol_version"`
		}
		if err := json.Unmarshal(line, &payload); err != nil {
			return msg, err
		}
		msg.Version = &terraform.VersionInfo{
			TerraformVersion: payload.TerraformVersion,
			ProtocolVersion:  payload.ProtocolVersion,
		}
	case terraform.MessageTypePlannedChange:
		var payload struct {
			Change   terraform.Change           `json:"change"`
			Resource terraform.ResourceInstance `json:"resource"`
		}
		if err := json.Unmarshal(line, &payload); err != nil {
			return msg, err
		}
		msg.PlannedChange = &terraform.PlannedChange{
			Resource: payload.Resource,
			Change:   payload.Change,
		}
	case terraform.MessageTypeChangeSummary:
		var payload struct {
			Changes       terraform.ChangeCounts `json:"changes"`
			ResourceDrift int                    `json:"resource_drift"`
			OutputChanges terraform.ChangeCounts `json:"outputs"`
		}
		if err := json.Unmarshal(line, &payload); err != nil {
			return msg, err
		}
		msg.ChangeSummary = &terraform.ChangeSummary{
			Changes:       payload.Changes,
			ResourceDrift: payload.ResourceDrift,
			OutputChanges: payload.OutputChanges,
		}
	case terraform.MessageTypeApplyStart,
		terraform.MessageTypeApplyProgress,
		terraform.MessageTypeApplyComplete,
		terraform.MessageTypeApplyErrored:
		hook, err := decodeHook(line)
		if err != nil {
			return msg, err
		}
		msg.Hook = hook
	case terraform.MessageTypeDiagnostic:
		var payload struct {
			Diagnostic terraform.Diagnostic `json:"diagnostic"`
		}
		if err := json.Unmarshal(line, &payload); err != nil {
			return msg, err
		}
		msg.Diagnostic = &payload.Diagnostic
	case terraform.MessageTypeResourceDrift:
		var payload struct {
			Change   terraform.Change           `json:"change"`
			Resource terraform.ResourceInstance `json:"resource"`
		}
		if err := json.Unmarshal(line, &payload); err != nil {
			return msg, err
		}
		msg.ResourceDrift = &terraform.ResourceDrift{
			Resource: payload.Resource,
			Change:   payload.Change,
		}
	case terraform.MessageTypeOutputs:
		var payload struct {
			Outputs map[string]terraform.Output `json:"outputs"`
		}
		if err := json.Unmarshal(line, &payload); err != nil {
			return msg, err
		}
		msg.Outputs = payload.Outputs
	}
	return msg, nil
}

func decodeHook(line []byte) (*terraform.HookMessage, error) {
	var payload struct {
		Hook terraform.HookMessage `json:"hook"`
	}
	if err := json.Unmarshal(line, &payload); err == nil && (payload.Hook.Address != "" || payload.Hook.Resource.Address != "") {
		return &payload.Hook, nil
	}

	var hook terraform.HookMessage
	if err := json.Unmarshal(line, &hook); err != nil {
		return nil, err
	}
	return &hook, nil
}

type planAccumulator struct {
	planData  *terraform.Plan
	resources map[string]int
}

func newPlanAccumulator() *planAccumulator {
	return &planAccumulator{
		planData: &terraform.Plan{
			Resources:     []terraform.ResourceChange{},
			OutputChanges: []terraform.OutputChange{},
		},
		resources: make(map[string]int),
	}
}

func (p *planAccumulator) apply(msg terraform.StreamMessage) {
	if p == nil {
		return
	}
	switch msg.Type {
	case terraform.MessageTypePlannedChange:
		if msg.PlannedChange == nil {
			return
		}
		address := msg.PlannedChange.Resource.Address
		if address == "" {
			return
		}
		change := msg.PlannedChange.Change
		rc := terraform.ResourceChange{
			Address:      address,
			ModulePath:   msg.PlannedChange.Resource.ModulePath,
			Mode:         "",
			ResourceType: msg.PlannedChange.Resource.ResourceType,
			ResourceName: msg.PlannedChange.Resource.ResourceName,
			ProviderName: msg.PlannedChange.Resource.ProviderName,
			Change:       &change,
		}
		rc.Action = terraform.GetActionType(rc.Change.Actions)
		if rc.Action == terraform.ActionNoOp {
			return
		}
		if idx, ok := p.resources[address]; ok && idx < len(p.planData.Resources) {
			p.planData.Resources[idx] = rc
		} else {
			p.resources[address] = len(p.planData.Resources)
			p.planData.Resources = append(p.planData.Resources, rc)
		}
	case terraform.MessageTypeOutputs:
		if msg.Outputs == nil {
			return
		}
		p.planData.OutputChanges = p.planData.OutputChanges[:0]
		for name, output := range msg.Outputs {
			outputChange := terraform.OutputChange{
				Name:      name,
				Action:    terraform.GetActionType(output.Actions),
				Sensitive: output.Sensitive,
				Change: &terraform.OutputChangeDetail{
					Actions: output.Actions,
					Before:  output.Before,
					After:   output.After,
				},
			}
			p.planData.OutputChanges = append(p.planData.OutputChanges, outputChange)
		}
	}
}

func (p *planAccumulator) snapshot() *terraform.Plan {
	if p == nil || p.planData == nil {
		return nil
	}
	if len(p.planData.Resources) == 0 && len(p.planData.OutputChanges) == 0 {
		return nil
	}
	planCopy := *p.planData
	return &planCopy
}
