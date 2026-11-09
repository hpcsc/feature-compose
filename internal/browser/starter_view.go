package browser

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hpcsc/feature-compose/internal/catalog"
)

var heading = lipgloss.NewStyle().Bold(true)

type starterView struct {
	name         string
	definitions  catalog.Definitions
	featureNames []string
	selected     int
}

func newStarterView(definitions catalog.Definitions, starterName string) *starterView {
	narrowed, _ := definitions.ForStarter(starterName)
	return &starterView{name: starterName, definitions: narrowed, featureNames: slices.Sorted(maps.Keys(narrowed.Features))}
}

func (v *starterView) moved(by int) *starterView {
	moved := *v
	moved.selected = min(max(v.selected+by, 0), max(len(v.featureNames)-1, 0))
	return &moved
}

func (v *starterView) selectedFeature() *featureView {
	if len(v.featureNames) == 0 {
		return nil
	}
	name := v.featureNames[v.selected]
	return &featureView{name: name, definition: v.definitions.Features[name]}
}

func (v *starterView) view() string {
	starter := v.definitions.Starters[v.name]
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n\n", heading.Render(v.name), starter.Summary)

	b.WriteString(heading.Render("Folders") + "\n")
	if len(starter.Folders) == 0 {
		b.WriteString("  none\n")
	}
	for _, name := range slices.Sorted(maps.Keys(starter.Folders)) {
		folder := starter.Folders[name]
		fmt.Fprintf(&b, "  %s  path: %s\n", name, folder.Path)
		if len(folder.AfterRender) > 0 {
			fmt.Fprintf(&b, "    after_render: %s\n", strings.Join(folder.AfterRender, ", "))
		}
		if len(folder.Test) > 0 {
			fmt.Fprintf(&b, "    test: %s\n", strings.Join(folder.Test, ", "))
		}
	}

	b.WriteString("\n" + heading.Render("Options") + "\n")
	optionsSection(starter.Options).write(&b)

	b.WriteString("\n" + heading.Render("Extension points") + "\n")
	declarers := map[string]string{}
	extensionPoints := map[string]catalog.ExtensionPointDefinition{}
	for name, extensionPoint := range starter.ExtensionPoints {
		declarers[name], extensionPoints[name] = "the starter", extensionPoint
	}
	for _, featureName := range v.featureNames {
		for name, extensionPoint := range v.definitions.Features[featureName].ExtensionPoints {
			declarers[name], extensionPoints[name] = "the feature "+featureName, extensionPoint
		}
	}
	if len(extensionPoints) == 0 {
		b.WriteString("  none\n")
	}
	for _, name := range slices.Sorted(maps.Keys(extensionPoints)) {
		fmt.Fprintf(&b, "  %s  item: %s  declared by %s\n    %s\n", name, strings.Join(extensionPoints[name].Item, ", "), declarers[name], extensionPoints[name].Summary)
	}

	b.WriteString("\n" + heading.Render("Features") + "\n")
	if len(v.featureNames) == 0 {
		b.WriteString("  No feature targets this starter.\n")
	}
	for i, name := range v.featureNames {
		cursor := "  "
		if i == v.selected {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%s  %s\n", cursor, name, v.definitions.Features[name].Summary)
	}

	b.WriteString("\nh back · j/k move · l open · q quit\n")
	return b.String()
}
