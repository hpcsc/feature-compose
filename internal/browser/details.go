package browser

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/hpcsc/feature-compose/internal/catalog"
	"github.com/hpcsc/feature-compose/internal/screen"
)

type declaredExtensionPoint struct {
	catalog.ExtensionPointDefinition
	declarer string
}

func extensionPointsOf(narrowed catalog.Definitions, starterName string) map[string]declaredExtensionPoint {
	extensionPoints := map[string]declaredExtensionPoint{}
	for name, extensionPoint := range narrowed.Starters[starterName].ExtensionPoints {
		extensionPoints[name] = declaredExtensionPoint{ExtensionPointDefinition: extensionPoint, declarer: "the starter"}
	}
	for _, featureName := range slices.Sorted(maps.Keys(narrowed.Features)) {
		for name, extensionPoint := range narrowed.Features[featureName].ExtensionPoints {
			extensionPoints[name] = declaredExtensionPoint{ExtensionPointDefinition: extensionPoint, declarer: "the feature " + shortName(featureName)}
		}
	}
	return extensionPoints
}

func detailsOf(definitions catalog.Definitions, r row, width int) string {
	if r.isCatalog() {
		return catalogDetails(definitions, r, width)
	}
	narrowed, _ := definitions.ForStarter(r.starter)
	switch {
	case r.isStarter():
		return starterDetails(narrowed.Starters[r.starter], width)
	case r.isGroup() && r.group == featuresGroup:
		return featuresDetails(narrowed, width)
	case r.isGroup():
		return extensionPointsDetails(narrowed, r.starter, width)
	case r.group == featuresGroup:
		return featureDetails(narrowed.Features[r.name], width)
	}
	return extensionPointDetails(narrowed, r, width)
}

func catalogDetails(definitions catalog.Definitions, r row, width int) string {
	starters := screen.Section{Heading: "Starters", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(definitions.Starters)) {
		if (row{starter: name}).catalogName() == r.catalog {
			starters.Entries = append(starters.Entries, screen.Entry{Name: shortName(name), Summary: definitions.Starters[name].Summary})
		}
	}
	features := screen.Section{Heading: "Features", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(definitions.Features)) {
		if catalogName, _ := catalog.SplitFullName(name); catalogName == r.catalog {
			features.Entries = append(features.Entries, screen.Entry{Name: shortName(name), Summary: definitions.Features[name].Summary})
		}
	}
	origin := definitions.Catalogs[r.catalog]
	path, ref, commit := origin.Path, origin.Ref, origin.Commit
	if path == "" {
		path = "the root of the source"
	}
	switch {
	case commit == "":
		ref, commit = "none: a local folder", "none: a local folder"
	case ref == "":
		ref = "the default branch"
	}
	source := screen.Entry{Fields: []screen.Field{
		{Label: "source", Values: []string{origin.Source}},
		{Label: "path", Values: []string{path}},
		{Label: "ref", Values: []string{ref}},
		{Label: "commit", Values: []string{commit}},
	}}
	return strings.Join(append(source.Lines(width), ""), "\n") + "\n" + screen.Details("", width, starters, features)
}

func starterDetails(starter catalog.StarterDefinition, width int) string {
	folders := screen.Section{Heading: "Folders", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(starter.Folders)) {
		folder := starter.Folders[name]
		fields := []screen.Field{{Label: "path", Values: []string{folder.Path}}}
		if folder.Generate != "" {
			fields = append(fields, screen.Field{Label: "generate", Values: []string{folder.Generate}})
		}
		fields = append(fields, screen.Field{Label: "after_render", Values: folder.AfterRender}, screen.Field{Label: "test", Values: folder.Test})
		folders.Entries = append(folders.Entries, screen.Entry{Name: name, Fields: fields})
	}
	sections := []screen.Section{folders, optionsSection(starter.Options)}
	if len(starter.Remove) > 0 {
		removals := screen.Section{Heading: "Removals", None: "none"}
		for _, r := range starter.Remove {
			removals.Entries = append(removals.Entries, screen.Entry{Name: r.File, Fields: queryFields(r.Pattern, r.Rule)})
		}
		sections = append(sections, removals)
	}
	return screen.Details(starter.Summary, width, sections...)
}

func queryFields(pattern string, rule map[string]any) []screen.Field {
	switch {
	case pattern != "":
		return []screen.Field{{Label: "pattern", Values: []string{pattern}}}
	case rule != nil:
		encoded, _ := json.Marshal(rule)
		return []screen.Field{{Label: "rule", Values: []string{string(encoded)}}}
	}
	return nil
}

func featuresDetails(narrowed catalog.Definitions, width int) string {
	features := screen.Section{Heading: "Features", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(narrowed.Features)) {
		features.Entries = append(features.Entries, screen.Entry{Name: shortName(name), Summary: narrowed.Features[name].Summary})
	}
	return screen.Details("", width, features)
}

func extensionPointsDetails(narrowed catalog.Definitions, starterName string, width int) string {
	extensionPoints := extensionPointsOf(narrowed, starterName)
	s := screen.Section{Heading: "Extension points", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(extensionPoints)) {
		s.Entries = append(s.Entries, screen.Entry{Name: name, Fields: []screen.Field{
			{Label: "item", Values: []string{strings.Join(extensionPoints[name].Item, ", ")}},
			{Label: "declared by", Values: []string{extensionPoints[name].declarer}},
		}})
	}
	return screen.Details("", width, s)
}

func featureDetails(feature catalog.FeatureDefinition, width int) string {
	byShortName := make(map[string]map[string]string, len(feature.Needs))
	for name, values := range feature.Needs {
		byShortName[shortName(name)] = values
	}
	needs := screen.ValuesSection("Needs", "none", byShortName)

	declared := screen.Section{Heading: "Extension points that it declares", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(feature.ExtensionPoints)) {
		extensionPoint := feature.ExtensionPoints[name]
		declared.Entries = append(declared.Entries, screen.Entry{Name: name, Summary: extensionPoint.Summary, Fields: []screen.Field{
			{Label: "item", Values: []string{strings.Join(extensionPoint.Item, ", ")}},
		}})
	}

	addedTo := screen.Section{Heading: "Extension points that it adds items to", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(feature.AddTo)) {
		addedTo.Entries = append(addedTo.Entries, screen.Entry{Name: name})
	}

	return screen.Details(feature.Summary, width, needs, optionsSection(feature.Options), declared, addedTo)
}

func extensionPointDetails(narrowed catalog.Definitions, r row, width int) string {
	extensionPoint := extensionPointsOf(narrowed, r.starter)[r.name]
	addedToBy := []string{}
	if _, ok := narrowed.Starters[r.starter].AddTo[r.name]; ok {
		addedToBy = append(addedToBy, "the starter")
	}
	for _, featureName := range slices.Sorted(maps.Keys(narrowed.Features)) {
		if _, ok := narrowed.Features[featureName].AddTo[r.name]; ok {
			addedToBy = append(addedToBy, shortName(featureName))
		}
	}
	if len(addedToBy) == 0 {
		addedToBy = []string{screen.NoneStyle.Render("none")}
	}
	fields := screen.Entry{Fields: []screen.Field{{Label: "file", Values: []string{extensionPoint.File}}}}
	if extensionPoint.Pattern != "" || extensionPoint.Rule != nil {
		fields.Fields = append(fields.Fields, screen.Field{Label: "language", Values: []string{extensionPoint.Language}})
		fields.Fields = append(fields.Fields, queryFields(extensionPoint.Pattern, extensionPoint.Rule)...)
		if extensionPoint.After != "" {
			fields.Fields = append(fields.Fields, screen.Field{Label: "after", Values: []string{extensionPoint.After}})
		}
		if extensionPoint.Before != "" {
			fields.Fields = append(fields.Fields, screen.Field{Label: "before", Values: []string{extensionPoint.Before}})
		}
	}
	fields.Fields = append(fields.Fields,
		screen.Field{Label: "item", Values: []string{strings.Join(extensionPoint.Item, ", ")}},
		screen.Field{Label: "declared by", Values: []string{extensionPoint.declarer}},
		screen.Field{Label: "added to by", Values: addedToBy},
	)
	var lines []string
	if extensionPoint.Summary != "" {
		lines = append(screen.Wrapped(extensionPoint.Summary, width), "")
	}
	return strings.Join(append(lines, fields.Lines(width)...), "\n")
}

func optionsSection(options map[string]catalog.OptionDefinition) screen.Section {
	s := screen.Section{Heading: "Options", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(options)) {
		option := options[name]
		fields := []screen.Field{{Label: "default", Values: []string{option.Default}}}
		if len(option.Values) > 0 {
			fields = append(fields, screen.Field{Label: "values", Values: []string{strings.Join(option.Values, ", ")}})
		}
		if option.Pattern != "" {
			fields = append(fields, screen.Field{Label: "pattern", Values: []string{option.Pattern}})
		}
		if option.When != "" {
			fields = append(fields, screen.Field{Label: "when", Values: []string{option.When}})
		}
		s.Entries = append(s.Entries, screen.Entry{Name: name, Fields: fields})
	}
	return s
}
