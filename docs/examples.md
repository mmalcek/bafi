# Examples
## Command line
note: in Powershell you must use <span style="color:red; font-weight: bold;">.\\</span>bafi.exe e.g.
```powershell
.\bafi.exe -i input.csv -f csv -t "?{{toXML .}}"
curl.exe -s someurl.com/api/xxx | .\bafi.exe -f json -t "?{{toXML .}}"
```
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

### Append output file
Redirect stdout to file and append ( > = replace, >> = apppend )
```sh
bafi.exe -i testdata.xml -t template.tmpl >> output.txt
```

## Template
Examples are based on testdata.tmpl included in project

### XML to CSV
- command
```sh
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

### JSON to CSV
- command
```sh
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t myTemplate.tmpl -o output.html
```
- myTemplate.tmpl
```
name,surname
{{- range .customers}}
"{{.firstname}}","{{.lastname}}"
{{- end}}
```

### JSON to HTML
- command
```sh
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
### JSON to custom XML
- command 
```sh
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
```sh
bafi.exe -i testdata.xml -t myTemplate.tmpl -o output.json
```

- myTemplate.tmpl
```
{{- $new := "{\"employees\": [" }}
{{- range .TOP_LEVEL.DATA_LINE}}
{{- $new = print $new "{\"employeeID\":\"" (index .Employee "-ID") "\", \"val1\":" .val1 "}," }}
{{- end}}
{{- /* Trim trailing comma, alternatively you can remove last char by "(slice $new 0 (sub (len $new) 1))" */}}
{{- $new = print (trimSuffix $new "," ) "]}"}}
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

### CSV to text
- command
```sh
bafi.exe -i users.csv -f csv -t myTemplate.tmpl -o output.txt
```
users.csv
```
name,surname
John,"Jack Doe"
```
- myTemplate.tmpl
```
Users:
{{- range .}}
Name: {{.name}}, Surname: {{.surname}}
{{- end}}
```
note: CSV file must be **[RFC4180](https://datatracker.ietf.org/doc/html/rfc4180)** compliant, file must have header line and separator must be **comma ( , )**

### Any SQL to XML
Bafi can be used in combination with very interesting tool **USQL** [https://github.com/xo/usql](https://github.com/xo/usql). USQL allows query almost any SQL like database (MSSQL,MySQL,postgres, ...) and get result in various formats. In this example we use -J for JSON. Output can be further processed by BaFi and templates

```sh
usql.exe mssql://user:password@server/instance/database -c "SELECT * FROM USERS" -J -q | bafi.exe -f json -t "?{{toXML .}}'"
```

### MongoDump to CSV
- command
```sh
bafi.exe -i users.bson -f bson -t myTemplate.tmpl -o output.html
```
- myTemplate.tmpl
```
name,surname
{{- range .}}
"{{.firstname}}","{{.lastname}}"
{{- end}}
```

### Input autoformat to XXX
Input data can be easily fomated to oher formats by functions **toXML,toJSON,toBSON,toYAML**. In this case its not necesarry add template file because it's as easy as 
```sh
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toXML .}}" -o output.xml
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toJSON .}}" -o output.json
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toBSON .}}" -o output.bson
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toYAML .}}" -o output.yml
```

### Multiple input files
Bafi can read multiple input files and merge them into one output file. This will require aditional file with files description.
Description file must be in YAML format as described below and prefixed by question mark **"?"** for examle **bafi.exe -t ?files.yaml**

Example: 

- batch file which gets the data from multiple sources **myFiles.bat**
```sh
curl -s https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml > ecbRates.xml
curl -s https://goweather.herokuapp.com/weather/prague > pragueWeather.json
```
- Files description **myFiles.yaml**
```yaml
- file: ./ecbRates.xml # file path
  format: xml # File format
  label: RATES # Label which will be used in the template {{ .RATES }}
- file: ./pragueWeather.json
  format: json
  label: WEATHER
```
- Template file **myTemplate.tmpl** which will generate simple HTML page with data
```html
<html>
    <body>
    <h3> Weather in Prague </h3>
    <h4> Temperatre: {{.WEATHER.temperature}} </h4>
    <h4> Wind: {{.WEATHER.wind}} </h4>
    <h3> ECB Exchange rates from: {{dateFormat (index .RATES.Envelope.Cube.Cube "-time") "2006-01-02" "02.01.2006" }}</h3>
        <table>
            <tr><th>currency</th><th>rate</th>
            {{- range .RATES.Envelope.Cube.Cube.Cube }} 
            <tr><td>{{index . "-currency" }}</td><td>{{index . "-rate" }}</td>
            {{- end}}
        </table>
    <body>
</html>

<style>
table, th, td { border: 1px solid black; width: 400px; }
</style>
```
- Finally run bafi
```sh
bafi.exe -t myTemplate.tmpl -i ?myFiles.yaml -o output.html
```


