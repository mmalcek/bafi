# Convert XML to CSV

Flatten repeating XML records into CSV rows, including values stored in attributes.

## Sample input

This is `testdata.xml`, shipped with every bafi release:

<!-- test:file data.xml -->
```xml
<?xml version="1.0" encoding="utf-8"?>
<TOP_LEVEL description="employeeData">
  <DATA_LINE>
    <Employee ID="0021"></Employee>
    <Trans_Date>2021-05-21</Trans_Date>
    <val1>5</val1>
    <val2>12</val2>
    <Linked_Text VALUE="Magic Bus Company"></Linked_Text>
  </DATA_LINE>
  <DATA_LINE>
    <Employee ID="0023"></Employee>
    <Trans_Date>2021-05-23</Trans_Date>
    <val1>43</val1>
    <val2>9</val2>
    <Linked_Text VALUE="Baltimore Grand"></Linked_Text>
  </DATA_LINE>
</TOP_LEVEL>
```

## Template

<!-- test:file employees-csv.tmpl -->
```
Employee,Date,val1,val2,SUM,LinkedText
{{- range mustArray .TOP_LEVEL.DATA_LINE}}
{{index .Employee "-ID"}},{{dateFormat .Trans_Date "2006-01-02" "02.01.2006"}},{{.val1}},{{.val2}},{{add .val1 .val2}},"{{index .Linked_Text "-VALUE"}}"
{{- end}}
```

<!-- test:cmd -->
```sh
bafi -i data.xml -t employees-csv.tmpl
```

<!-- test:output -->
```
Employee,Date,val1,val2,SUM,LinkedText
0021,21.05.2021,5,12,17,"Magic Bus Company"
0023,23.05.2021,43,9,52,"Baltimore Grand"
```

- `index .Employee "-ID"` reads the `ID` attribute; attributes are prefixed with `-`, and keys with a
  `-` need `index` instead of a dot.
- `mustArray` keeps the template working when the file contains only one `DATA_LINE`.
- `dateFormat` converts dates using Go's [reference layout](../functions.md#date-functions)
  (`2006-01-02` means YYYY-MM-DD).
- `add` converts the text values to numbers before adding.

Related: [XML to JSON](xml-to-json.md) · [JSON to CSV](json-to-csv.md)
