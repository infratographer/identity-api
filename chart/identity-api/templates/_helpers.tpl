{{/* vim: set filetype=mustache: */}}

{{- define "idapi.listenPort" }}
{{- .Values.config.server.port | default 8080 }}
{{- end }}
