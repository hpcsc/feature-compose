package browser

import (
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
			extensionPoints[name] = declaredExtensionPoint{ExtensionPointDefinition: extensionPoint, declarer: "the feature " + featureName}
		}
	}
	return extensionPoints
}

func detailsOf(definitions catalog.Definitions, r row, width int) string {
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
	return extensionPointDetails(narrowed, r.starter, r.name, width)
}

func starterDetails(starter catalog.StarterDefinition, width int) string {
	folders := screen.Section{Heading: "Folders", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(starter.Folders)) {
		folder := starter.Folders[name]
		folders.Entries = append(folders.Entries, screen.Entry{Name: name, Fields: []screen.Field{
			{Label: "path", Values: []string{folder.Path}},
			{Label: "after_render", Values: folder.AfterRender},
			{Label: "test", Values: folder.Test},
		}})
	}
	return screen.Details(starter.Summary, width, folders, optionsSection(starter.Options))
}

func featuresDetails(narrowed catalog.Definitions, width int) string {
	features := screen.Section{Heading: "Features", None: "none"}
	for _, name := range slices.Sorted(maps.Keys(narrowed.Features)) {
		features.Entries = append(features.Entries, screen.Entry{Name: name, Summary: narrowed.Features[name].Summary})
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
	needs := screen.Section{Heading: "Needs", None: "none"}
	for _, name := range feature.Needs {
		needs.Entries = append(needs.Entries, screen.Entry{Name: name})
	}

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

func extensionPointDetails(narrowed catalog.Definitions, starterName, name string, width int) string {
	extensionPoint := extensionPointsOf(narrowed, starterName)[name]
	addedToBy := []string{}
	for _, featureName := range slices.Sorted(maps.Keys(narrowed.Features)) {
		if _, ok := narrowed.Features[featureName].AddTo[name]; ok {
			addedToBy = append(addedToBy, featureName)
		}
	}
	if len(addedToBy) == 0 {
		addedToBy = []string{screen.NoneStyle.Render("none")}
	}
	fields := screen.Entry{Fields: []screen.Field{
		{Label: "file", Values: []string{extensionPoint.File}},
		{Label: "item", Values: []string{strings.Join(extensionPoint.Item, ", ")}},
		{Label: "declared by", Values: []string{extensionPoint.declarer}},
		{Label: "added to by", Values: addedToBy},
	}}
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
		s.Entries = append(s.Entries, screen.Entry{Name: name, Fields: fields})
	}
	return s
}
