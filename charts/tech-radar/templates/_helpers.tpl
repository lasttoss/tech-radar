{{- define "tech-radar.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tech-radar.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "tech-radar.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tech-radar.labels" -}}
app.kubernetes.io/name: {{ include "tech-radar.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{- define "tech-radar.selectorLabels" -}}
app.kubernetes.io/name: {{ include "tech-radar.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "tech-radar.image" -}}
{{- printf "%s:%s" .Values.image.repository (default .Chart.AppVersion .Values.image.tag) -}}
{{- end -}}

{{- define "tech-radar.secretName" -}}
{{- if .Values.telegram.existingSecret -}}
{{- .Values.telegram.existingSecret -}}
{{- else -}}
{{- include "tech-radar.fullname" . -}}
{{- end -}}
{{- end -}}
