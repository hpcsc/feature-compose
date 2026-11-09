package browser

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/catalog"
)

type featureView struct {
	name       string
	definition catalog.FeatureDefinition
}

func (v featureView) view() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n\n", heading.Render(v.name), v.definition.Summary)

	b.WriteString(heading.Render("Needs") + "\n")
	if len(v.definition.Needs) == 0 {
		b.WriteString("  none\n")
	} else {
		fmt.Fprintf(&b, "  %s\n", strings.Join(v.definition.Needs, ", "))
	}

	b.WriteString("\n" + heading.Render("Options") + "\n")
	optionsSection(v.definition.Options).write(&b)

	b.WriteString("\n" + heading.Render("Extension points that it declares") + "\n")
	if len(v.definition.ExtensionPoints) == 0 {
		b.WriteString("  none\n")
	}
	for _, name := range slices.Sorted(maps.Keys(v.definition.ExtensionPoints)) {
		extensionPoint := v.definition.ExtensionPoints[name]
		fmt.Fprintf(&b, "  %s  item: %s\n    %s\n", name, strings.Join(extensionPoint.Item, ", "), extensionPoint.Summary)
	}

	b.WriteString("\n" + heading.Render("Extension points that it adds items to") + "\n")
	if len(v.definition.AddTo) == 0 {
		b.WriteString("  none\n")
	}
	for _, name := range slices.Sorted(maps.Keys(v.definition.AddTo)) {
		fmt.Fprintf(&b, "  %s\n", name)
	}

	b.WriteString("\nh back · q quit\n")
	return b.String()
}
