package parser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestTextParserComplexMultilineLists(t *testing.T) {
	plan, err := NewTextParser().ParseFile("../../../testdata/plans/complex.txt")
	if err != nil {
		t.Fatal(err)
	}
	changes := make(map[string]*terraform.Change)
	for _, resource := range plan.Resources {
		changes[resource.Address] = resource.Change
	}
	web := changes["aws_instance.web"]
	if web == nil {
		t.Fatal("missing instance change")
	}
	assertListValue(t, web.Before["vpc_security_group_ids"], []any{"sg-111", "sg-222"})
	assertListValue(t, web.After["vpc_security_group_ids"], []any{"sg-111", "sg-333"})
	assertListValue(t, web.After["tags"], map[string]any{"Environment": "prod", "CostCenter": "platform"})

	helm := changes["module.app.helm_release.app"]
	if helm == nil {
		t.Fatal("missing Helm change")
	}
	assertListValue(t, helm.Before["set"], []any{"image.tag=1.2.3", "resources.requests.cpu=250m"})
	assertListValue(t, helm.After["set"], []any{"image.tag=1.2.4", "resources.requests.cpu=250m"})
	before := `applications:\n  name: web\n  retry: true\n  rollout:\n    maxUnavailable: 1\n  env:\n    LOG_LEVEL: info\n`
	after := `applications:\n  name: web\n  retry: false\n  rollout:\n    maxUnavailable: 1\n  env:\n    LOG_LEVEL: debug\n`
	assertListValue(t, helm.Before["values"], []any{before})
	assertListValue(t, helm.After["values"], []any{after})

	legacy := changes["aws_security_group.legacy"]
	if legacy == nil {
		t.Fatal("missing security group change")
	}
	assertListValue(t, legacy.Before["ingress"], []any{map[string]any{
		"from_port": float64(22), "to_port": float64(22), "protocol": "tcp",
		"cidr_blocks": []any{"0.0.0.0/0"},
	}})
	assertListValue(t, legacy.Before["tags"], map[string]any{"Environment": "dev"})
}

func TestTextParserMultilineListObjectsAndNeighboringMaps(t *testing.T) {
	input := `# example.item will be updated in-place
~ resource "example" "item" {
  ~ settings = {
      ~ rules = [
          ~ {
                name = "retained"
              ~ port = 80 -> 443
              ~ sources = [
                    "shared",
                  - "old",
                  + "new",
                ]
            },
          - {
              - name = "removed"
            },
          + {
              + name = "added"
            },
        ]
      ~ labels = {
          ~ "tier" = "old" -> "new"
        }
      + enabled = true
    }
  ~ description = "before" -> "after"
}
`
	for _, stream := range []bool{false, true} {
		t.Run(map[bool]string{false: "reader", true: "stream"}[stream], func(t *testing.T) {
			var plan *terraform.Plan
			var err error
			if stream {
				lines := strings.Split(input, "\n")
				ch := make(chan string, len(lines))
				for _, line := range lines {
					ch <- line
				}
				close(ch)
				plan, err = NewTextParser().ParseStream(ch)
			} else {
				plan, err = NewTextParser().Parse(strings.NewReader(input))
			}
			if err != nil {
				t.Fatal(err)
			}
			change := plan.Resources[0].Change
			assertListValue(t, change.Before, map[string]any{
				"description": "before",
				"settings": map[string]any{
					"rules": []any{
						map[string]any{"name": "retained", "port": float64(80), "sources": []any{"shared", "old"}},
						map[string]any{"name": "removed"},
					},
					"labels": map[string]any{"tier": "old"},
				},
			})
			assertListValue(t, change.After, map[string]any{
				"description": "after",
				"settings": map[string]any{
					"rules": []any{
						map[string]any{"name": "retained", "port": float64(443), "sources": []any{"shared", "new"}},
						map[string]any{"name": "added"},
					},
					"labels": map[string]any{"tier": "new"}, "enabled": true,
				},
			})
		})
	}
}

func TestTextParserMultilineListScalarValues(t *testing.T) {
	input := `# example.item will be updated in-place
~ resource "example" "item" {
  ~ values = [
        -2,
        true,
        null,
        "key=value # literal -> unchanged }",
      ~ "old -> literal" -> "new -> literal", # forces replacement
      - false,
      + 3,
    ]
  + empty = [
    ]
  - removed = [
      - "gone",
    ] -> null
  + added = [
      + "new",
    ]
}
`
	plan, err := NewTextParser().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	change := plan.Resources[0].Change
	if change == nil {
		t.Fatal("missing change data")
	}
	assertListValue(t, change.Before, map[string]any{
		"values":  []any{float64(-2), true, nil, "key=value # literal -> unchanged }", "old -> literal", false},
		"removed": []any{"gone"},
	})
	assertListValue(t, change.After, map[string]any{
		"values": []any{float64(-2), true, nil, "key=value # literal -> unchanged }", "new -> literal", float64(3)},
		"empty":  []any{}, "added": []any{"new"},
	})
}

func TestTextParserMultilineListUnknown(t *testing.T) {
	input := `# example.item will be created
+ resource "example" "item" {
  + values = [
      + (known after apply),
    ]
}
`
	plan, err := NewTextParser().Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	change := plan.Resources[0].Change
	if change == nil {
		t.Fatal("missing change data")
	}
	assertListValue(t, change.Before, map[string]any{})
	assertListValue(t, change.After, map[string]any{"values": []any{nil}})
	assertListValue(t, change.AfterUnknown, map[string]any{"values": true})
}

func assertListValue(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}
}
