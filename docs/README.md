[![Go](https://github.com/mmalcek/bafi/actions/workflows/go.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/go.yml)
[![CodeQL](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/)
# Universal JSON, BSON, YAML, XML converter with templates

## Key features
- Various input formats **(json, bson, yaml, xml)**
- Flexible output formatting using text templates
- Support for [LUA](https://www.lua.org/pil/contents.html) custom functions which allows very flexible data manipulation
- stdin/stdout support which allows get data from source -> translate -> delivery to destination. This allows easily translate data between different web services like REST to SOAP

<img src="img/scheme.svg" style="border: 0;" height="150px" />

## Command line arguments
- "-i input.xml" Input file name. If not defined app tries read stdin
- "-o output.txt" Output file name. If not defined result is send to stdout
- "-t template.tmpl" Template file. Alternatively you can use *inline* template 
    - inline template must start with **?** e.g. -t **"?{{.someValue}}"**
- "-f json" Alternative input format. 
    - Supported formats: **json, bson, yaml, xml (default)**
- "-v" - Show current verion
- "-?" - list available command line arguments

#### Examples
- Basic
```sh
bafi.exe -i testdata.xml -o output.csv -t template.tmpl
```
- JSON input
```sh
bafi.exe -i testdata.json -f json -o output.csv -t template.tmpl
```
- STDIN
```sh
curl.exe -s "someURL" | bafi.exe -f json -t myTemplate.tmpl -o out.txt 
```
More info about curl [here](https://curl.se/) but you can of course use any tool with stdout

- Inline template
```sh
bafi.exe -i testdata.xml -o output.json -t "?{{toJSON .}}"
```
- Working example with inline template
```
curl.exe -s "https://api.predic8.de/shop/products/" | bafi.exe -f json -t "?{{range .products}}\"{{.name}}\",{{.product_url}}{{print \"\n\"}}{{end}}" 
```

## Template formating
- Basic iterate over lines
```
{{range .TOP_LEVEL.DATA_LINE}}{{.val1}}{{end}}
```
- Get XML tag (-VALUE = tag name)
```
{{index .Linked_Text "-VALUE"}}
```
- Use functions (count val1 + val2)
```
{{add .val1 .val2}} 
```
- If statement 
```
{{if gt (int $val1) (int $val2)}}Value1{{else}}Value2{{end}} is greater
```
Check template.tmpl and testdata.xml for more advanced example

(more detailed info on https://golang.org/pkg/text/template/ )

## Lua custom functions
Aside of builtin functions you can write your own custom lua functions defined in ./lua/functions.lua file

- Input is always passed as json array of strings
- Output must be passed as string
- lua table array starts with 1
- Lua [documentation](http://www.lua.org/manual/5.1/)

Minimal functions.lua example
```lua
json = require './lua/json'

function sum(incomingData) 
    dataTable = json.decode(incomingData)
    return tostring(tonumber(dataTable[1]) + tonumber(dataTable[2]))
end
```

### Call Lua function in template 
```
{{lua "sum" .val1 .val2}}
```
- "sum" - Lua function name

### Call built-in function
```
{{add .val1 .val2}}
```

## Built-in functions
- add -> {{add .Value1 .Value2}}
- add1
- sub
- div
- mod
- mul
- randInt
- add1f - "...f" functions parse float but provide **decimal** operation using https://github.com/shopspring/decimal
- addf
- subf
- divf
- mulf
- round
- max
- min
- maxf
- minf
- dateFormat -> {{dateFormat .Value "oldFormat" "newFormat"}}
- now - {{now "02.01.2006"}} - GO format date (see notes below)
- b64enc
- b64dec
- b32enc
- b32dec
- uuid
- regexMatch
- upper
- lower
- trim
- trimAll
- trimSuffix
- trimPrefix
- atoi
- int64
- int
- float64
- toJSON - convert input object to JSON
- toBSON - convert input object to BSON
- toYAML - convert input object to YAML
- toXML - convert input object to XML
- mapJSON - convert stringified JSON to map so it can be used as object or translated to other formats (e.g. "toXML"). Check template.tmpl for example 

### dateFormat
dateFormat can parse date and time using [GO time format](https://programming.guide/go/format-parse-string-time-date-example.html)

## Relase builds (download)
- Releases (windows, linux, mac) can be found [here](https://github.com/mmalcek/bafi/releases)