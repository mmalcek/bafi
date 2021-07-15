# Examples
## Command line
### Basic  
Get data from *testdata.xml* -> process using *template.tmpl* -> save output as *output.txt*
```sh
bafi.exe -i testdata.xml -t template.tmpl -o output.txt
```
### Inline template
Get data from *testdata.xml* -> process using inline template -> save output as *output.json*
```sh
bafi.exe -i testdata.xml -o output.json -t "?{{toJSON .}}"
```
note: BaFi inline template must start with **?** e.g. **"?{{toJSON .}}"**
[How to format inline templates](https://pkg.go.dev/text/template#hdr-Examples)

### Stdin/REST
Get data from REST api -> convert to XML -> output to Stdout. 
```sh
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toXML .}}"
```
More info about curl [here](https://curl.se/) but you can of course use any tool with stdout

## Template
Examples are based on testdata.tmpl included in project

### XML to CSV
- command
```
bafi.exe -i testdata.xml -t myTemplate.tmpl -o output.csv
```
- myTemplate.tmpl
```
Employee,Date,val1,val2,val3,SUM,LuaMultiply,linkedText
{{- range .TOP_LEVEL.DATA_LINE}}
{{index .Employee "-ID"}},
{{- dateFormat .Trans_Date "2006-01-02" "02.01.2006"}},
{{- .val1}},{{.val2}},{{.val3}},
{{- add .val1 .val2}},
{{- lua "mul" .val1 .val2}},"{{index .Linked_Text "-VALUE"}}"
{{- end}}
```

### REST to HTML
- command
```
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t myTemplate.tmpl -o output.html
```
- myTemplate.tmpl
```
<html>
    <body>
        <table>
            <tr><th>Name</th><th>Surname</th></tr>
            {{- range .customers}}
            <tr><td>{{.firstname}}</td><td>{{.lastname}}</td></tr>
            {{- end }}
        </table>
    </body>
</html>

<style>
table, th, td { border: 1px solid black; width: 400px; }
</style>
```
### REST to custom XML
- command 
```
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t myTemplate.tmpl -o output.xml
```
- myTemplate.tmpl
```
<?xml version="1.0" encoding="utf-8"?>
<MY_DATA>
    {{- range .customers}}
    <CUSTOMMER>
        <NAME>{{.firstname}}</NAME>
        <SURNAME>{{.lastname}}</SURNAME>
    </CUSTOMMER>
    {{- end }}
</MY_DATA>
```

### XML to custom JSON
- command 
```
bafi.exe -i testdata.xml -t myTemplate.tmpl -o output.json
```

- myTemplate.tmpl
```
{{- $new := "{\"employees\": [" }}
{{- range .TOP_LEVEL.DATA_LINE}}
{{- $new = print $new "{\"employeeID\":\"" (index .Employee "-ID") "\", \"val1\":" .val1 "}," }}
{{- end}}
{{- /* "slice $new 0 (sub (len $new) 1" - remove trailing comma  */}}
{{- $new = print (slice $new 0 (sub (len $new) 1)) "]}" }}
{{ $new}}
```
JSON in $new variable can be mapped to struct and autoformatted to other formats like:

- Transform $new to YAML
```
{{toYAML (mapJSON $new) -}}
```
- Transform $new to XML
```
{{toXML (mapJSON $new) -}}
```
### Input autoformat to???
Input data can be easily fomated to oher formats by functions **toXML,toJSON,toBSON,toYAML**. In this case its not necesarry add template file because it's as easy as 
```
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toXML .}}" -o output.xml
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toJSON .}}" -o output.json
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toBSON .}}" -o output.bson
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toYAML .}}" -o output.yml
```
