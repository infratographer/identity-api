{{/* vim: set filetype=mustache: */}}
{{/*
Create the SeedIssuer object to bootstrap the token exchange with an issuer.
*/}}
{{- define "im-sts.seedIssuer" }}
          - tenantID: {{ .tenantID }}
            id:  {{ .id }}
            name: {{ .name }}
            uri:  {{ .uri }}
            jwksURI: {{ .jwksURI }}
            claimMappings:
            {{- range $k, $v := .claimMappings }}
              {{ $k | quote }}: {{ $v | quote }}
            {{- end }}
{{- end }}

{{- define "im-sts.listenPort" }}
{{- .Values.identityManagerSTS.config.server.port | default 8080 }}
{{- end }}
