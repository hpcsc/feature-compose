package browser

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/catalog"
)

type optionsSection map[string]catalog.OptionDefinition

func (o optionsSection) write(b *strings.Builder) {
	if len(o) == 0 {
		b.WriteString("  none\n")
	}
	for _, name := range slices.Sorted(maps.Keys(o)) {
		option := o[name]
		fmt.Fprintf(b, "  %s  default: %s", name, option.Default)
		if len(option.Values) > 0 {
			fmt.Fprintf(b, "  values: %s", strings.Join(option.Values, ", "))
		}
		if option.Pattern != "" {
			fmt.Fprintf(b, "  pattern: %s", option.Pattern)
		}
		b.WriteString("\n")
	}
}
