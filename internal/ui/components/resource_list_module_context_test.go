package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/ushiradineth/lazytf/internal/styles"
	"github.com/ushiradineth/lazytf/internal/terraform"
)

func TestSingleResourceModuleRetainsContextBesideOtherModules(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 20)
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.alpha.aws_instance.one", Action: terraform.ActionUpdate},
		{Address: "module.alpha.aws_instance.two", Action: terraform.ActionUpdate},
		{Address: "module.app.helm_release.app", Action: terraform.ActionUpdate},
	})

	view := ansi.Strip(r.View())
	alpha := strings.Index(view, "module.alpha")
	app := strings.Index(view, "module.app")
	helm := strings.Index(view, "helm_release.app")
	if alpha < 0 || app <= alpha || helm <= app {
		t.Fatalf("expected Helm resource beneath its own module header:\n%s", view)
	}

	r.SetSelectedIndex(indexOfGroup(r.visibleItems, "module.alpha"))
	r.ToggleGroup()
	if !strings.Contains(ansi.Strip(r.View()), "helm_release.app") {
		t.Fatal("collapsing alpha must not hide app's Helm resource")
	}

	r.SetSelectedIndex(indexOfGroup(r.visibleItems, "module.app"))
	r.ToggleGroup()
	view = ansi.Strip(r.View())
	if !strings.Contains(view, "module.app") || strings.Contains(view, "helm_release.app") {
		t.Fatalf("expected collapsed app header without its resource:\n%s", view)
	}
	r.ToggleGroup()
	r.MoveDown()
	resource := r.GetSelectedResource()
	if resource == nil || resource.Address != "module.app.helm_release.app" {
		t.Fatalf("expected selecting app's resource after expanding, got %#v", resource)
	}
}

func TestNestedSingleResourceModuleRetainsEveryHeader(t *testing.T) {
	r := NewResourceList(styles.DefaultStyles())
	r.SetSize(80, 20)
	r.SetResources([]terraform.ResourceChange{
		{Address: "module.app.module.chart.helm_release.app", Action: terraform.ActionUpdate},
	})
	view := ansi.Strip(r.View())
	app := strings.Index(view, "module.app")
	chart := strings.Index(view, "module.chart")
	helm := strings.Index(view, "helm_release.app")
	if app < 0 || chart <= app || helm <= chart {
		t.Fatalf("expected every parent module before the resource:\n%s", view)
	}
}
