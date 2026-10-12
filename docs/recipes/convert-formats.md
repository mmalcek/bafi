# Convert between JSON, YAML, TOON and BSON

Generic format conversion needs no template file: an inline template with one conversion function is
enough. The input format comes from the file extension or from `-f`.

## Sample input

<!-- test:file config.yaml -->
```yaml
service: billing
replicas: 3
ports:
  - 8080
  - 8443
owner:
  team: payments
  email: payments@example.com
```

## YAML to JSON

<!-- test:cmd -->
```sh
bafi -i config.yaml -t '?{{toJSON .}}'
```

<!-- test:output -->
```json
{
  "owner": {
    "email": "payments@example.com",
    "team": "payments"
  },
  "ports": [
    8080,
    8443
  ],
  "replicas": 3,
  "service": "billing"
}
```

## JSON to YAML

<!-- test:file config.json -->
```json
{"service": "billing", "replicas": 3, "ports": [8080, 8443]}
```

<!-- test:cmd -->
```sh
bafi -i config.json -t '?{{toYAML .}}'
```

<!-- test:output -->
```yaml
ports:
    - 8080
    - 8443
replicas: 3
service: billing
```

## To TOON (for LLM prompts)

[TOON](https://github.com/toon-format/toon) (Token-Oriented Object Notation) carries the same data as
JSON with far fewer tokens, which makes prompts to language models cheaper.

<!-- test:cmd -->
```sh
bafi -i config.yaml -t '?{{toTOON .}}'
```

<!-- test:output -->
```
owner:
  email: payments@example.com
  team: payments
ports[2]: 8080,8443
replicas: 3
service: billing
```

## To BSON (MongoDB)

`toBSON` writes binary BSON, so send it to a file rather than the terminal:

<!-- test:cmd quiet -->
```sh
bafi -i config.json -t '?{{toBSON .}}' -o config.bson
```

…and read it back like any other input:

<!-- test:cmd -->
```sh
bafi -i config.bson -t '?{{.service}} runs {{.replicas}} replicas'
```

<!-- test:output -->
```
billing runs 3 replicas
```

## Notes

- Keys are sorted alphabetically in JSON and YAML output; JSON and YAML maps have no order.
- `toJSON` indents with two spaces. To control the layout or the value types, write the JSON in the
  template as in [CSV to JSON](csv-to-json.md#numbers-and-booleans).
- In PowerShell use double quotes around inline templates: `.\bafi.exe -i config.yaml -t "?{{toJSON .}}"`.

Related: [XML to JSON](xml-to-json.md) · [JSON to XML](json-to-xml.md)
