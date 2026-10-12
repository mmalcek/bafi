# Convert JSON to XML

Turn JSON, for example a REST API response, into XML: generic XML in one line, or the exact XML layout
another system expects.

## Sample input

<!-- test:file users.json -->
```json
{
  "users": [
    {"id": 1, "name": "Alice Smith", "email": "alice@example.com"},
    {"id": 2, "name": "Tom & Jerry", "email": "tj@example.com"}
  ]
}
```

## One-line conversion

<!-- test:cmd -->
```sh
bafi -i users.json -t '?{{toXML .}}'
```

<!-- test:output -->
```xml
<doc>
  <users>
    <email>alice@example.com</email>
    <id>1</id>
    <name>Alice Smith</name>
  </users>
  <users>
    <email>tj@example.com</email>
    <id>2</id>
    <name>Tom &amp; Jerry</name>
  </users>
</doc>
```

`toXML` wraps everything in `<doc>`; a top-level JSON array becomes a list of `<record>` elements.
Special characters such as `&` are escaped for you.

## Your own XML layout

When the receiving system expects specific element names, write them in the template. Use the
built-in `html` function on every value: it escapes `&`, `<`, `>` and quotes, which is exactly what XML
text and attributes need.

<!-- test:file users-xml.tmpl -->
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Customers>
{{- range .users}}
  <Customer id="{{.id}}">
    <Name>{{html .name}}</Name>
    <Email>{{html .email}}</Email>
  </Customer>
{{- end}}
</Customers>
```

<!-- test:cmd -->
```sh
bafi -i users.json -t users-xml.tmpl
```

<!-- test:output -->
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Customers>
  <Customer id="1">
    <Name>Alice Smith</Name>
    <Email>alice@example.com</Email>
  </Customer>
  <Customer id="2">
    <Name>Tom &amp; Jerry</Name>
    <Email>tj@example.com</Email>
  </Customer>
</Customers>
```

JSON numbers are printed without decimals when they are whole (`1`, not `1.0`).

## From an API

```sh
curl -fs https://example.com/api/users | bafi -f json -t users-xml.tmpl -o customers.xml
```

Related: [XML to JSON](xml-to-json.md) · [REST response to SOAP request](rest-to-soap.md)
