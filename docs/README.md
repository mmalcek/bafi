# BaFi

**Universal JSON, BSON, YAML, CSV, XML, mt940 translator to ANY format using templates**

**Github repository**

- [https://github.com/mmalcek/bafi](https://github.com/mmalcek/bafi)

**Releases (Windows, MAC, Linux)**

- [https://github.com/mmalcek/bafi/releases](https://github.com/mmalcek/bafi/releases)

## Key features

- Various input formats **(json, bson, yaml, csv, xml, mt940)**
- Flexible output formatting using text templates
- Output can be anything: HTML page, SQL Query, Shell script, CSV file, ...
- Support for [Lua](https://www.lua.org/pil/contents.html) custom functions which allows very flexible data manipulation
- stdin/stdout support which allows get data from source -> translate -> delivery to destination. This allows easily translate data between different web services like **REST to SOAP, SOAP to REST, REST to CSV, ...**
- Merge multiple input files in various formats into single output file formated using template
- Support chatGPT queries to analyze or format data (experimental)

<img src="img/scheme.svg" style="border: 0;" height="150px" />

[![Go](https://github.com/mmalcek/bafi/actions/workflows/go.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/go.yml)
[![CodeQL](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mmalcek/bafi)](https://goreportcard.com/report/github.com/mmalcek/bafi)
[![License](https://img.shields.io/github/license/mmalcek/bafi)](https://github.com/mmalcek/bafi/blob/main/LICENSE)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#text-processing)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/mmalcek/bafi?label=latest%20release)](https://github.com/mmalcek/bafi/releases/latest)

<!--![GitHub All Releases Downloads](https://img.shields.io/github/downloads/mmalcek/bafi/total)-->

**If you like this app you can buy me a coffe ;)**

<a href='https://ko-fi.com/mmalcek' target='_blank'>
	<img height='40' style='border:0px;height:50px;' src='https://az743702.vo.msecnd.net/cdn/kofi3.png?v=0' border='0' alt='Buy Me a Coffee at ko-fi.com' />
</a>

## How does it work?

Application automaticaly parse input data into object which can be simply accessed in tamplate using dot notation where first dot represent root of object **{{ . }}**.

For example JSON document **myUser.json**

```json
{
  "user": {
    "name": "John Doe",
    "age": 25,
    "address": {
      "street": "Main Street",
      "city": "New York",
      "state": "NY"
    },
    "favourite_colors": ["red", "green", "blue"]
  }
}
```

- Get user name:

```sh
bafi.exe -i myUser.json -t '?{{.user.name}}'
```

- Use function to change all letters to uppercase:

```sh
bafi.exe -i myUser.json -t '?{{upper .user.name}}'
```

- Use IF statement to compare user age to 20:

```sh
bafi.exe -i myUser.json -t '?User is {{if gt (toInt .user.age) 20}}old{{else}}young{{end}}.'
```

- List favourite colors:

```sh
bafi.exe -i myUser.json -t '?{{range .user.favourite_colors}}{{.}},{{end}}'
```

- Format data using template file **myTemplate.tmpl** and save output to **myUser.txt**:

```sh
bafi.exe -i myUser.json -t myTemplate.tmpl -o myUser.txt
```

```
{{- /* Content of myTemplate.tmpl file */ -}}
User: {{.user.name}}
Age: {{.user.age}}
Address: {{.user.address.street}}, {{.user.address.city}} - {{.user.address.state}}
{{- /* Create list of colors and remove comma at the end */ -}}
{{- $colors := ""}}{{range .user.favourite_colors}}{{$colors = print $colors . ", "}}{{end}}
{{- $colors = print (trimSuffix $colors ", " )}}
Favourite colors: {{$colors}}
```

note: in Powershell you must use <span style="color:red; font-weight: bold;">.\\</span>bafi.exe e.g.

```powershell
.\bafi.exe -i input.csv -t "?{{toXML .}}"
curl.exe -s someurl.com/api/xxx | .\bafi.exe -f json -t "?{{toXML .}}"
```

More examples [here](examples/#template)

<script src="js/wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly
  .instantiateStreaming(fetch('js/bafi.wasm'), go.importObject)
  .then((result) => { go.run(result.instance)});
function getBAFI() {
    let input1 = document.getElementById("input1").value;
    let input2 = document.getElementById("input2").value;
    let format = document.getElementById("format").value;
    var element = document.getElementById("bafiData");
    element.innerHTML = bafi(input1,input2,format); 
}
</script>

## Online demo (WASM)

**Just try it here :)**
<textarea type="text" id="input1" rows="13" cols="40">
{
"user": {
"name": "John Doe",
"age": 25,
"address": {
"street": "Main Street",
"city": "New York",
"state": "NY"
},
"favourite_colors": ["red", "green", "blue"]
}
}
</textarea>

<textarea type="text" id="input2" rows="13" cols="40">
Hello {{upper .user.name}},

you are {{.user.age}} years old
and live in {{.user.address.city}}, {{.user.address.state}}.
Your favourite colors are:
{{range .user.favourite_colors}} {{.}}
{{end}}
</textarea>
<br />
<select name="format" id="format">

<option value="json">JSON</option>
<option value="xml">XML</option>
<option value="yaml">YAML</option>
<option value="csv">CSV</option>
</select>
<button type="button" onClick="getBAFI()">Create OUTPUT</button>

<pre style="max-width: 600px; min-height:200px"  id="bafiData"></pre>

## Command line arguments

- **-i input.xml** Input file name.
  - If not defined app tries read stdin
  - If prefixed with "?" (**-i ?files.yaml**) app will expect yaml file with multiple files description. See [example](examples/#multiple-input-files)
- **-o output.txt** Output file name.
  - If not defined result is send to stdout
- **-t template.tmpl** Template file. Alternatively you can use _inline_ template
  - inline template must start with **?** e.g. -t **"?{{.someValue}}"**
- **-f json** Input format.
  - Supported formats: **json, bson, yaml, csv, xml, mt940**
  - If not defined (for file input) app tries detect input format automatically by file extension
- **-d ','** Data delimiter
  - format CSV:
    - Can be defined as string e.g. -d ',' or as [hex](https://www.asciitable.com/asciifull.gif) value prefixed by **0x** e.g. 'TAB' can be defined as -f 0x09. Default delimiter is comma (**,**)
  - format mt940:
    - For Multiple messages in one file (e.g. Multicash). Can be defined as string e.g. -d "-\}\r\n" or "\r\n$" . If delimiter is set BaFi will return array of mt940 messages
- **-v** Show current verion
- **-?** list available command line arguments
- **-gk myChatGPTToken** - ChatGPT token
- **-gq "What's the current CZK rate?"** - ChatGPT query
- **-gm gpt35** - ChatGPT model. Currently supportsed options "gpt35"(default), "gpt4"

```sh
bafi.exe -i testdata.xml -t template.tmpl -o output.txt
```

More examples [here](examples/#command-line)

## Templates

Bafi uses [text/template](https://pkg.go.dev/text/template). Here is a quick summary how to use. Examples are based on _testdata.xml_ included in project

note: in **vscode** you can use [gotemplate-syntax](https://marketplace.visualstudio.com/items?itemName=casualjim.gotemplate) for syntax highlighting

### Comments

```
{{/* a comment */}}
{{- /* a comment with white space trimmed from preceding and following text */ -}}
```

### Trim new line

New line before or after text can be trimmed by adding dash

```
{{- .TOP_LEVEL}}, {{.TOP_LEVEL -}}
```

### Accessing data

Data are accessible by _pipline_ which is represented by dot

- Simplest template

```
{{.}}
```

- Get data form inner node

```
{{.TOP_LEVEL}}
```

- Get data from XML tag. XML tags are autoprefixed by dash and accessible as index

```
{{index .TOP_LEVEL "-description"}}
```

- Convert TOP_LEVEL node to JSON

```
{{toJSON .TOP_LEVEL}}
```

### Variables

You can store selected data to [template variable](https://pkg.go.dev/text/template#hdr-Variables)

```
{{$myVar := .TOP_LEVEL}}
```

### Actions

Template allows to use [actions](https://pkg.go.dev/text/template#hdr-Actions), for example

Iterate over lines

```
{{range .TOP_LEVEL.DATA_LINE}}{{.val1}}{{end}}
```

If statement

```
{{if gt (int $val1) (int $val2)}}Value1{{else}}Value2{{end}} is greater
```

### Functions

In go templates all operations are done by functions where function name is followed by operands

For example:

count val1+val2

```
{{add $val1 $val2}}
```

count (val1+val2)/val3

```
{{div (add $val1 $val2) $val3}}
```

This is called [Polish notation](https://en.wikipedia.org/wiki/Polish_notation) or "Prefix notation" also used in another languages like [Lisp](<https://en.wikipedia.org/wiki/Lisp_(programming_language)>)

The key benefit of using this notation is that order of operations is clear. For example **6/2\*(1+2)** - even diferent calculators may have different opinion on order of operations in this case. With Polish notation order of operations is strictly defined (from inside to outside) **div 6 (mul 2 (add 1 2))** . This brings benefits with increasing number of operations especially in templates where math and non-math operations can be mixed together.

 <img src="img/calc.jpg" height="160px">

For example we have json array of items numbered from **0**

```json
{ "items": ["item-0", "item-1", "item-2", "item-3"] }
```

We need change items numbering to start with **1**. To achieve this we have to do series of operations: 1. trim prefix "item-" -> 2. convert to int -> 3. add 1 -> 4. convert to string -> 5. append "item-" for all items in range. This can be done in one line

```
{{ range .items }}{{ print "item-" (toString (add1 (toInt (trimPrefix . "item-")))) }} {{ end }}
```

or alternatively (slightly shorter) print formatted string - examples [here](https://zetcode.com/golang/string-format/), documentation [here](https://golang.org/pkg/fmt/)

```
{{ range .items }}{{ printf "item-%d " (add1 (toInt (trimPrefix . "item-"))) }}{{ end }}
```

but BaFi also tries automaticaly cast variables so the shortest option is

```
{{range .items}}{{print "item-" (add1 (trimPrefix . "item-"))}} {{end}}
```

Expected result: **item-1 item-2 item-3 item-4**

There are 3 categories of functions

#### Native functions

text/template integrates [native functions](https://pkg.go.dev/text/template#hdr-Functions) to work with data

#### Additional functions

Asside of integated functions bafi contains additional common functions

##### Math functions

- **add** - {{add .Value1 .Value2}}
- **add1** - {{add1 .Value1}} = Value1+1
- **sub** - substract
- **div** - divide
- **mod** - modulo
- **mul** - multiply
- **randInt** - return random integer {{randInt .Min .Max}}
- **add1f** - "...f" functions parse float but provide **decimal** operations using [shopspring decimal](https://github.com/shopspring/decimal)
- **addf**
- **subf**
- **divf**
- **mulf**
- **round** - {{round .Value1 2}} - will round to 2 decimals
- **max** - {{round .Value1 .Value2 .Value3 ...}} get Max value from range
- **min** - get Min value from range
- **maxf**
- **minf**

##### Date functions

- **dateFormat** - {{dateFormat .Value "oldFormat" "newFormat"}} - [GO time format](https://programming.guide/go/format-parse-string-time-date-example.html)
  - {{dateFormat "2021-08-26T22:14:00" "2006-01-02T15:04:05" "02.01.2006-15:04"}}
- **dateFormatTZ** - {{dateFormatTZ .Value "oldFormat" "newFormat" "timeZone"}}
  - This fuction is similar to dateFormat but applies timezone offset - [Timezones](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)
  - {{dateFormatTZ "2021-08-26T03:35:00.000+04:00" "2006-01-02T15:04:05.000-07:00" "02.01.2006-15:04" "Europe/Prague"}}
- **dateToInt** - {{dateToInt .Value "dateFormat"}} - convert date to integer (unixtime, int64), usefull for comparing dates
- **intToDate** - {{intToDate .Value "dateFormat"}} - convert integer (unixtime, int64) to date, usefull for comparing dates
- **now** - {{now "02.01.2006"}} - GO format date (see notes below)

##### String functions

- **addSubstring** - {{addSubstring $myString, "XX", $position}} add substring to $position in string (if $position is 1,2,3 = Adding from right, if -1,-2,-3 = Adding from left)
- **atoi** - {{atoi "042"}} - string to int. Result will be 42. atoi must be used especially for convert strings with leading zeroes
- **b64enc** - encode to base64
- **b64dec** - decode from base64
- **b32enc** - oncode to base32
- **b32dec** - decode from base32
- **contains** - check if string contains substring e.g. {{contains "aaxbb" "xb"}}
- **indexOf** - {{indexOf "aaxbb" "xb"}} - returns indexOf first char of substring in string
- **isArray** - {{isArray .Value1}} - check if value is array
- **isBool** - {{isBool .Value1}} - check if value is bool
- **isInt** - {{isInt .Value1}} - check if value is int
- **isFloat64** - {{isFloat64 .Value1}} - check if value is float64
- **isString** - {{isString .Value1}} - check if value is string
- **isMap** - {{isMap .Value1}} - check if value is map
- **regexMatch** - {{regexMatch pattern .Value1}} more about go [regex](https://gobyexample.com/regular-expressions)
- **replaceAll** - {{replaceAll "oldValue" "newValue" .Value}} - replace all occurences of "oldValue" with "newValue" e.g. {{replaceAll "x" "Z" "aaxbb"}} -> "aaZbb"
- **replaceAllRegex** - {{replaceAllRegex "regex" "newValue" .Value}} - replace all occurences of "regex" with "newValue" e.g. {{replaceAllRegex "[a-d]", "Z" "aaxbb"}} -> "ZZxZZ"
- **lower** - to lowercase
- **trim** - remove leading and trailing whitespace
- **trimPrefix** - {{trimPrefix "!Hello World!" "!"}} - returns "Hello World!"
- **trimSuffix** - {{trimSuffix "!Hello World!" "!"}} - returns "!HelloWorld"
- **mapJSON** - convert stringified JSON to map so it can be used as object or translated to other formats (e.g. "toXML"). Check template.tmpl for example
- **mustArray** - {{mustArray .Value1}} - convert to array. Useful with XML where single node is not treated as array
- **toBool** - {{toBool "true"}} - string to bool
- **toDecimal** - {{toDecimal "3.14159"}} - cast to decimal (if error return 0)
- **toDecimalString** - {{toDecimalString "3.14159"}} - cast to decimal string (if error return "error message")
- **toFloat64** - {{float64 "3.14159"}} - cast to float64
- **toInt** - {{int true}} - cast to int. Result will be 1. If you need convert string with leading zeroes use "atoi"
- **toInt64** - {{int64 "42"}} - cast to int64. Result will be 42. If you need convert string with leading zeroes use "atoi"
- **toString** - {{toString 42}} - int to string
- **toJSON** - convert input object to JSON
- **toBSON** - convert input object to BSON
- **toYAML** - convert input object to YAML
- **toXML** - convert input object to XML
- **trimAll** - {{trimAll "!Hello World!" "!"}} - returns "Hello World"
- **upper** - to uppercase
- **uuid** - generate UUID

#### Lua custom functions

You can write your own custom lua functions defined in ./lua/functions.lua file

Call Lua function in template ("sum" - Lua function name)

```
{{lua "sum" .val1 .val2}}
```

- Input is always passed as stringified JSON and should be decoded (json.decode(incomingData))
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

Check [examples](examples/) and **template.tmpl** and **testdata.xml** for advanced examples
