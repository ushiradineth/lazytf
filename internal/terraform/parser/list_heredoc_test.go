package parser

import (
	"strings"
	"testing"
)

func TestTextParserListHeredocs(t *testing.T) {
	input := `# example.item will be updated in-place
~ resource "example" "item" {
  ~ settings = {
      ~ values = [ # forces replacement
          - <<-EOT
old
}
]
            EOT,
          + <<-EOT
new
}
]
            EOT,
          ~ <<-EOT
              -   old
              +   new
            EOT,
          ~ {
              ~ body = <<-EOT
                  -   before
                  +   after
                EOT
            },
        ]
      + enabled = true
    }
  + sibling = "present"
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
		"settings": map[string]any{"values": []any{
			"old\n}\n]", strings.Repeat(" ", 17) + "old", map[string]any{"body": strings.Repeat(" ", 21) + "before"},
		}},
	})
	assertListValue(t, change.After, map[string]any{
		"settings": map[string]any{"values": []any{
			"new\n}\n]", strings.Repeat(" ", 17) + "new", map[string]any{"body": strings.Repeat(" ", 21) + "after"},
		}, "enabled": true},
		"sibling": "present",
	})
}
