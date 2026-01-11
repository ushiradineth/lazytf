package diff

import (
	"fmt"
	"testing"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

func buildBenchmarkResourceChange(attrCount int) *terraform.ResourceChange {
	before := make(map[string]any, attrCount)
	after := make(map[string]any, attrCount)
	for i := 0; i < attrCount; i++ {
		key := fmt.Sprintf("attr_%d", i)
		before[key] = i
		if i%10 == 0 {
			after[key] = i + 1
		} else {
			after[key] = i
		}
	}

	change := terraform.Change{
		Actions: []string{"update"},
		Before:  before,
		After:   after,
	}

	return &terraform.ResourceChange{
		Address:      "bench.resource",
		ResourceType: "bench_type",
		ResourceName: "bench",
		ProviderName: "registry.terraform.io/hashicorp/null",
		Change:       &change,
	}
}

// Target: <= 1ms/op for 1k-attribute diff on developer hardware.
func BenchmarkEngineGetResourceDiffsCold(b *testing.B) {
	resource := buildBenchmarkResourceChange(1000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine := NewEngine()
		engine.GetResourceDiffs(resource)
	}
}

// Target: <= 50us/op for cached diff lookup on developer hardware.
func BenchmarkEngineGetResourceDiffsHot(b *testing.B) {
	resource := buildBenchmarkResourceChange(1000)
	engine := NewEngine()
	engine.GetResourceDiffs(resource)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.GetResourceDiffs(resource)
	}
}
