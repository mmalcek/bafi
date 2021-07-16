[![Go](https://github.com/mmalcek/bafi/actions/workflows/go.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mmalcek/bafi)](https://goreportcard.com/report/github.com/mmalcek/bafi)
[![CodeQL](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml)

# Universal JSON, BSON, YAML, XML translator to **ANY** format using templates

<img src="./docs/img/scheme.svg" style="border: 0;" height="150px" />

## Key features
- Various input formats **(json, bson, yaml, xml)**
- Flexible output formatting using text templates
- Support for [LUA](https://www.lua.org/pil/contents.html) custom functions which allows very flexible data manipulation
- stdin/stdout support which allows get data from source -> translate -> delivery to destination. This allows easily translate data between different web services like **REST to SOAP, SOAP to REST, REST to CSV, ...**



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