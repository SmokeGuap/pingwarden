{{- define "pingwarden.image" -}}
{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}
{{- end }}

{{- define "pingwarden.labels" -}}
app.kubernetes.io/name: pingwarden
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "pingwarden.podSecurity" -}}
runAsNonRoot: true
runAsUser: 65532
runAsGroup: 65532
seccompProfile:
  type: RuntimeDefault
{{- end }}

{{- define "pingwarden.containerSecurity" -}}
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
capabilities:
  drop: ["ALL"]
{{- end }}

{{- define "pingwarden.env" -}}
- name: POSTGRES_PASSWORD
  valueFrom:
    secretKeyRef:
      name: {{ .Values.secretName }}
      key: POSTGRES_PASSWORD
- name: DATABASE_URL
  value: postgres://pingwarden:$(POSTGRES_PASSWORD)@{{ .Release.Name }}-postgres:5432/pingwarden?sslmode=disable
- name: REDIS_ADDR
  value: {{ .Release.Name }}-redis:6379
{{- range $k, $v := .Values.config }}
- name: {{ $k }}
  value: {{ $v | quote }}
{{- end }}
{{- end }}