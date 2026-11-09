package textcase

import (
	"strings"
	"unicode"
)

func Kebab(name string) string {
	return strings.Join(words(name), "-")
}

func Snake(name string) string {
	return strings.Join(words(name), "_")
}

func Camel(name string) string {
	parts := words(name)
	for i := 1; i < len(parts); i++ {
		parts[i] = capitalize(parts[i])
	}
	return strings.Join(parts, "")
}

func Pascal(name string) string {
	parts := words(name)
	for i := range parts {
		parts[i] = capitalize(parts[i])
	}
	return strings.Join(parts, "")
}

func words(name string) []string {
	var parts []string
	for _, field := range strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || r == '.' }) {
		runes := []rune(field)
		start := 0
		for i := 1; i < len(runes); i++ {
			if unicode.IsUpper(runes[i]) && unicode.IsLower(runes[i-1]) {
				parts = append(parts, strings.ToLower(string(runes[start:i])))
				start = i
			}
		}
		parts = append(parts, strings.ToLower(string(runes[start:])))
	}
	return parts
}

func capitalize(word string) string {
	runes := []rune(word)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
