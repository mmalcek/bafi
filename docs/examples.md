# Examples

## Command line

note: in Powershell you must use <span style="color:red; font-weight: bold;">.\\</span>bafi.exe e.g.

```powershell
.\bafi.exe -i input.csv -t "?{{toXML .}}"
curl.exe -s someurl.com/api/xxx | .\bafi.exe -f json -t "?{{toXML .}}"
```

### Basic

Get data from _testdata.xml_ -> process using _template.tmpl_ -> save output as _output.txt_

```sh
bafi.exe -i testdata.xml -t template.tmpl -o output.txt
```

### Inline template

Get data from _testdata.xml_ -> process using inline template -> save output as _output.json_

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
bafi.exe -i users.csv -t myTemplate.tmpl -o output.txt
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

note: CSV file must be **[RFC4180](https://datatracker.ietf.org/doc/html/rfc4180)** compliant, file must have header line and separator must be **comma ( , )**. Or you can use command line argument -d ( e.g. **-d ';'** or **-d 0x09** ) to define separator(delimiter).

### mt940 to CSV

- mt940 returns simple struct (Header,Fields,[]Transactions) of strings and additional parsing needs to be done in template. This allows full flexibility on data processing
- Identifiers are prefixed by **"F\_"** (e.g. **:20:** = **.Fields.F_20**)
- if parameter -d (delimiter e.g. -d "-\}\r\n" or "\r\n$") is defined for files with multiple messages (e.g. - Multicash), app returns array of mt940 messages.
- Note: This is actually good place to use integrated [LUA interpreter](/bafi/#lua-custom-functions) where you can create your own set of custom functions to parse data and easily reuse them in templates.

- command

```sh
bafi.exe -i message.sta -t myTemplate.tmpl -o output.csv
```

- myTemplate.tmpl

```
Reference, balance, VS
{{- $F20 := .Fields.F_20 }}{{ $F60F := .Fields.F_60F }}
{{range .Transactions }}
{{- $vsS := add (indexOf .F_86 "?21") 3 }} {{- $vsE := add $vsS 17 -}}
{{- $F20}}, {{$F60F}}, {{slice .F_86 $vsS $vsE}}
{{ end }}
```

### Any SQL to XML

Bafi can be used in combination with very interesting tool **USQL** [https://github.com/xo/usql](https://github.com/xo/usql). USQL allows query almost any SQL like database (MSSQL,MySQL,postgres, ...) and get result in various formats. In this example we use -J for JSON. Output can be further processed by BaFi and templates

```sh
usql.exe mssql://user:password@server/instance/database -c "SELECT * FROM USERS" -J -q | bafi.exe -f json -t "?{{toXML .}}"
```

### MongoDump to CSV

- command

```sh
bafi.exe -i users.bson -t myTemplate.tmpl -o output.html
```

- myTemplate.tmpl

```
name,surname
{{- range .}}
"{{.firstname}}","{{.lastname}}"
{{- end}}
```

### Dashes in key names

If key name contains dashes ( - ) bafi will fail with error "bad character U+002D '-'" for example:

```
{{.my-key.subkey}}
```

This is known limitation of go templates which can be solved by workaround

```
{{index . "my-key" "subkey"}}
```

### Input autoformat to XXX

Input data can be easily fomated to oher formats by functions **toXML,toJSON,toBSON,toYAML**. In this case its not necesarry add template file because it's as easy as

```sh
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toXML .}}" -o output.xml
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toJSON .}}" -o output.json
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toBSON .}}" -o output.bson
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toYAML .}}" -o output.yml
```

### ChatGPT query

```sh
curl -s https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml | ./bafi -f xml -gk myChatGPTToken -gq "What's the current CZK rate?"
curl -s https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml | ./bafi -f xml -gk myChatGPTToken -gq "format rates to html" -gm gpt4
curl -s "https://api.open-meteo.com/v1/forecast?latitude=52.52&longitude=13.41&hourly=temperature_2m&past_days=7&forecast_days=0" | ./bafi -f json -gk "myChatGPTToken" -gm gpt4o-mini -gq "Create forecast for next 2 days based on provided data"
./bafi -i invoice.json -gk myChatGPTToken -gq "create XML UBL format invoice" -o invoice.xml

```

### Multiple input files

Bafi can read multiple input files and merge them into one output file. This will require aditional file with files description.
Description file must be in YAML format as described below and prefixed by question mark **"?"** for examle **bafi.exe -i ?files.yaml**

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
