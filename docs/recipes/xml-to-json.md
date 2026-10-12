# Convert XML to JSON

Turn an XML file or API response into JSON, either as a straight conversion or reshaped into exactly
the JSON structure you need.

## Sample input

<!-- test:file orders.xml -->
```xml
<?xml version="1.0" encoding="UTF-8"?>
<orders>
  <order id="1001" status="paid">
    <customer>Alice Smith</customer>
    <total currency="EUR">120.50</total>
  </order>
  <order id="1002" status="open">
    <customer>Bob "Bobby" Brown</customer>
    <total currency="EUR">80.00</total>
  </order>
</orders>
```

## One-line conversion

<!-- test:cmd -->
```sh
bafi -i orders.xml -t '?{{toJSON .}}'
```

<!-- test:output -->
```json
{
  "orders": {
    "order": [
      {
        "-id": "1001",
        "-status": "paid",
        "customer": "Alice Smith",
        "total": {
          "#text": "120.50",
          "-currency": "EUR"
        }
      },
      {
        "-id": "1002",
        "-status": "open",
        "customer": "Bob \"Bobby\" Brown",
        "total": {
          "#text": "80.00",
          "-currency": "EUR"
        }
      }
    ]
  }
}
```

How XML maps to the data the template sees:

- **Attributes** become keys prefixed with `-`: `id="1001"` is `-id`.
- **Text of an element that also has attributes** is stored under `#text`: `<total currency="EUR">120.50</total>`
  becomes `{"-currency": "EUR", "#text": "120.50"}`.
- **Repeated elements** become a list; a single element is *not* a list (see [below](#one-element-or-many)).
- All values are text; XML has no numbers.

## Reshape into your own JSON

Pick the fields you need and give them your own names. `toJSON` on a single value produces a correctly
quoted and escaped JSON string, so names containing quotes stay valid JSON.

<!-- test:file orders-json.tmpl -->
```
[
{{- range $i, $o := .orders.order}}{{if $i}},{{end}}
  {"id": {{index $o "-id"}}, "customer": {{toJSON $o.customer}}, "total": {{index $o.total "#text"}}, "paid": {{eq (index $o "-status") "paid"}}}
{{- end}}
]
```

<!-- test:cmd -->
```sh
bafi -i orders.xml -t orders-json.tmpl
```

<!-- test:output -->
```json
[
  {"id": 1001, "customer": "Alice Smith", "total": 120.50, "paid": true},
  {"id": 1002, "customer": "Bob \"Bobby\" Brown", "total": 80.00, "paid": false}
]
```

## One element or many

If the XML contains a single `<order>`, `.orders.order` is that order itself, not a list, and `range`
iterates over its fields instead. Wrap it with `mustArray`, which turns a single value into a one-item
list and leaves lists alone:

<!-- test:file single.xml -->
```xml
<orders>
  <order id="1001"><customer>Alice Smith</customer></order>
</orders>
```

<!-- test:cmd -->
```sh
bafi -i single.xml -t '?{{range mustArray .orders.order}}{{.customer}}{{end}}'
```

<!-- test:output -->
```
Alice Smith
```

## From an API

```sh
curl -fs https://example.com/api/orders.xml | bafi -f xml -t orders-json.tmpl -o orders.json
```

`-f xml` is needed because stdin has no file extension. `curl -f` makes curl fail on HTTP errors
instead of piping an error page into bafi.

Related: [JSON to XML](json-to-xml.md) · [XML to CSV](xml-to-csv.md) · [SOAP response to JSON](soap-to-json.md)
