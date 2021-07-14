# Examples
## Command line
- **Basic:** Get data from *testdata.xml* -> process using *template.tmpl* -> save output as *output.txt*
```sh
bafi.exe -i testdata.xml -t template.tmpl -o output.csv
```
- **Inline template:** Get data from *testdata.xml* -> process using inline template -> save output as *output.json*
```sh
bafi.exe -i testdata.xml -o output.json -t "?{{toJSON .}}"
```
note: Inline template must start with **?** e.g. **"?{{toJSON .}}"**

- **Stdin/REST:** Get data from REST api -> convert to YAML -> output to Stdout. More examples [here](https://pkg.go.dev/text/template#hdr-Examples)
```sh
curl.exe -s https://reqres.in/api/users | bafi.exe -f json -t "?{{toYAML .}}"
```
More info about curl [here](https://curl.se/) but you can of course use any tool with stdout

## Template
Examples are based on testdata.tmpl included in project

- Transform whole input to json
```
{{toJSON .}}
```

- Transform "TOP_LEVEL" node to json
```
{{toJSON .TOP_LEVEL}}
```

- Transform data selection to JSON
```
{{- $new := "{\"employees\": [" }}
{{- range .TOP_LEVEL.DATA_LINE}}
{{- $new = print $new "{\"employeeID\":\"" (index .Employee "-ID") "\", \"val1\":" .val1 "}," }}
{{- end}}
{{- /* "slice $new 0 (sub (len $new) 1" - remove trailing comma  */}}
{{- $new = print (slice $new 0 (sub (len $new) 1)) "]}" }}
{{ $new}}
```

- Transform new JSON to YAML
```
{{toYAML (mapJSON $new) -}}
```
- Transform new JSON to XML
```
{{toXML (mapJSON $new) -}}
```
