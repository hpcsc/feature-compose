# {{ .Name }}

The owner of this repo is {{ option "starter.owner" }}.
{{- range items "readme.sections" }}

{{ .text }}
{{- end }}
