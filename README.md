[![Go](https://github.com/mmalcek/bafi/actions/workflows/go.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/go.yml)
[![CodeQL](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mmalcek/bafi)](https://goreportcard.com/report/github.com/mmalcek/bafi)
[![License](https://img.shields.io/github/license/mmalcek/bafi)](https://github.com/mmalcek/bafi/blob/main/LICENSE)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#text-processing) 
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/mmalcek/bafi?label=latest%20release)](https://github.com/mmalcek/bafi/releases/latest)

# Universal JSON, BSON, YAML, CSV, XML, mt940 translator to ANY format using templates

<img src="./docs/img/scheme.svg" style="border: 0;" height="150px" />

## Key features
- Various input formats **(json, bson, yaml, csv, xml, mt940)**
- Flexible output formatting using text templates
- Support for [Lua](https://www.lua.org/pil/contents.html) custom functions which allows very flexible data manipulation
- stdin/stdout support which allows get data from source -> translate -> delivery to destination. This allows easily translate data between different web services like **REST to SOAP, SOAP to REST, REST to CSV, ...**
- Merge multiple input files in various formats into single output file formated using template



## Documentation [https://mmalcek.github.io/bafi/](https://mmalcek.github.io/bafi/)

## Releases (Windows, MAC, Linux) [https://github.com/mmalcek/bafi/releases](https://github.com/mmalcek/bafi/releases)

usage: 
```
bafi.exe -i testdata.xml -t template.tmpl -o output.txt
```
or 
```
curl.exe -s https://api.predic8.de/shop/customers/ | bafi.exe -f json -t "?{{toXML .}}"
```

More examples and description in [documentation](https://mmalcek.github.io/bafi/)

**If you like this app you can buy me a coffe ;)**

<a href='https://ko-fi.com/mmalcek' target='_blank'>
	<img height='30' style='border:0px;height:40px;' src='https://az743702.vo.msecnd.net/cdn/kofi3.png?v=0' border='0' alt='Buy Me a Coffee at ko-fi.com' />
</a>