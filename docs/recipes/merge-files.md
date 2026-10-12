# Merge several files into one output

Combine data from several files, even in different formats, into a single report: exchange rates from
an XML feed plus orders from a JSON export, for example.

## 1. List the input files

A small YAML file names every input and the label it gets in the template:

<!-- test:file files.yaml -->
```yaml
- file: rates.xml     # path, relative to the current directory
  label: RATES        # the data is available as {{ .RATES }}
- file: orders.json
  label: ORDERS       # format omitted: taken from the .json extension
  # format: json      # set it when the extension is missing or unusual
```

Pass it with a `?` in front of the name, in quotes: `-i '?files.yaml'`. Without quotes, zsh (the macOS default
shell) treats `?` as a wildcard and stops with `no matches found`.

## 2. The input files

<!-- test:file rates.xml -->
```xml
<rates date="2024-03-01">
  <rate currency="USD">1.0812</rate>
  <rate currency="CZK">25.304</rate>
</rates>
```

<!-- test:file orders.json -->
```json
[
  {"id": 1, "amount": 100, "currency": "USD"},
  {"id": 2, "amount": 2530, "currency": "CZK"}
]
```

## 3. Template

Labels are the top-level keys. `$` always refers to the root, which is handy inside `range`.

<!-- test:file summary.tmpl -->
```
Orders in EUR (rates of {{index .RATES.rates "-date"}})
{{- range .ORDERS}}
{{- $order := .}}
{{- range $.RATES.rates.rate}}
{{- if eq (index . "-currency") $order.currency}}
#{{$order.id}}: {{$order.amount}} {{$order.currency}} = {{round (divf $order.amount (index . "#text")) 2}} EUR
{{- end}}
{{- end}}
{{- end}}
```

<!-- test:cmd -->
```sh
bafi -i '?files.yaml' -t summary.tmpl
```

<!-- test:output -->
```
Orders in EUR (rates of 2024-03-01)
#1: 100 USD = 92.49 EUR
#2: 2530 CZK = 99.98 EUR
```

In PowerShell and cmd.exe use double quotes: `-i "?files.yaml"`.

Each file is parsed on its own, so a broken file is reported with its own name and line. The `-d`
delimiter applies to every CSV file in the list.

Related: [HTML report from JSON](json-to-html.md) · [Command line](../cli.md#several-input-files)
