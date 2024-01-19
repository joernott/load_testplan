package testplan

var generate_template = `
jobs:
  load_testplan:
    runs-on: 'ubuntu-latest'
    steps:
      - name: 'Load Testplan'
        id: ltp
        uses: 'joernott/load_testplan@v1'
        with:
          files: '{{ range $i, $file := .Files }}{{ if gt $i 0 }},{{ end }}{{ $file }}{{ end }}'
          input_type: '{{ .InputType }}'
          separator: '{{ .Separator }}'
          set_output: {{ .SetOutput }}
          set_env: {{ .SetEnv }}
          set_print: {{ .SetPrint }}
          yaml: '{{ .YamlName }}'
          json: '{{ .JsonName }}'
          loglevel: '{{ .LogLevel }}'
          logfile: '{{ .LogFile }}'
    outputs:
{{- range $key, $value := .Outputs }}
      {{ $key }}: ${{"{{"}} steps.ltp.outputs.{{ $key }} {{"}}"}}
{{- end }}
`
