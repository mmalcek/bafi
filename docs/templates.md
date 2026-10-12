# Template guide

bafi templates are Go [text/template](https://pkg.go.dev/text/template) templates: plain text with
actions in `{{ }}`. Everything outside the braces is copied to the output as it is. The examples below
use this input:

<!-- test:file users.json -->
```json
{
  "company": "Example Ltd",
  "users": [
    {"name": "Alice", "age": 34, "roles": ["admin", "dev"], "my-id": "A1"},
    {"name": "Bob", "age": 19, "roles": ["dev"], "my-id": "B2"}
  ]
}
```

In VS Code the [gotemplate-syntax](https://marketplace.visualstudio.com/items?itemName=casualjim.gotemplate)
extension adds highlighting.

## Accessing data

The dot `.` is the current data; at the start it is the whole input. Follow keys with more dots:

<!-- test:cmd -->
```sh
bafi -i users.json -t '?{{.company}} has {{len .users}} users'
```

<!-- test:output -->
```
Example Ltd has 2 users
```

- `{{index .users 0}}` - an item of a list (counting from 0).
- `{{index . "my-id"}}` - keys containing `-`, spaces or other special characters need `index`.
- `{{toJSON .users}}` - print any part of the data, which is the quickest way to find a path.
- `$` is always the root, even inside `range` or `with`: `{{$.company}}`.

### XML specifics

XML becomes data like this:

- attributes are keys with a `-` prefix: `<order id="7">` → `{{index .order "-id"}}`;
- the text of an element that also has attributes is under `#text`: `{{index .total "#text"}}`;
- namespace prefixes are dropped: `<soap:Body>` → `.Envelope.Body`;
- an element that occurs once is a single value, several occurrences are a list. Use `mustArray` before
  `range` when it can be either: `{{range mustArray .orders.order}}`;
- all values are text; convert with `toInt`, `toFloat64` or the `...f` math functions.

See [XML to JSON](recipes/xml-to-json.md) for a complete example.

## Loops and conditions

<!-- test:file list.tmpl -->
```
{{range .users -}}
{{.name}} ({{index . "my-id"}}): {{if ge .age 21.0}}adult{{else}}junior{{end}}, roles: {{range $i, $r := .roles}}{{if $i}}, {{end}}{{$r}}{{end}}
{{end -}}
```

<!-- test:cmd -->
```sh
bafi -i users.json -t list.tmpl
```

<!-- test:output -->
```
Alice (A1): adult, roles: admin, dev
Bob (B2): junior, roles: dev
```

- `{{range .list}}...{{end}}` repeats for every item; inside, `.` is the item.
  `{{range $i, $item := .list}}` also gives the position.
- `{{if condition}}...{{else if other}}...{{else}}...{{end}}`.
- `{{with .value}}...{{end}}` runs only when the value exists and sets `.` to it.
- Comparisons: `eq`, `ne`, `lt`, `le`, `gt`, `ge`; combine with `and`, `or`, `not`. Compare values of the
  same kind: JSON numbers are `float64`, so compare with `21.0` or convert (`{{if ge (toInt .age) 21}}`).

## Whitespace control

A dash inside the braces removes whitespace and line breaks on that side: `{{-` trims before, `-}}`
trims after. In the loop above, `{{range .users -}}` drops the line break after the `range` line and
`{{end -}}` the one after `end`, so each user produces exactly one line.

```
{{- .name -}}
```

## Variables

```
{{$total := 0.0}}
{{range .items}}{{$total = addf $total .price}}{{end}}
Total: {{$total}}
```

`:=` creates a variable, `=` changes it. Variables live until the `{{end}}` of the block they were
created in.

## Functions

Functions are written before their arguments ("prefix" or Polish notation):

<img src="img/calc.jpg" height="160px" alt="calculator">

| Math | Template |
|---|---|
| 1 + 2 | `{{add 1 2}}` |
| (1 + 2) * 3 | `{{mul (add 1 2) 3}}` |
| 6 / 2 * (1 + 2) | `{{div 6 (mul 2 (add 1 2))}}` → 1, `{{mul (div 6 2) (add 1 2)}}` → 9 |

The order of operations is always explicit, inside out, so there is no doubt about precedence. Values
are converted automatically where possible:

<!-- test:file items.json -->
```json
{"items": ["item-0", "item-1", "item-2"]}
```

<!-- test:cmd -->
```sh
bafi -i items.json -t '?{{range .items}}{{print "item-" (add1 (trimPrefix . "item-"))}} {{end}}'
```

<!-- test:output -->
```
item-1 item-2 item-3 
```

The pipe `|` passes a value as the last argument of the next function: `{{.name | upper}}` equals
`{{upper .name}}`.

All functions with examples: [Function reference](functions.md). Your own functions: [Lua](recipes/lua.md).

## Comments

```
{{/* a comment */}}
{{- /* a comment that also trims the whitespace around it */ -}}
```

## Reusing parts of a template

```
{{define "user"}}{{.name}} <{{.email}}>{{end}}
{{range .users}}{{template "user" .}}
{{end}}
```

## Escaping for the output format

bafi writes values exactly as they are; it does not know whether the output is HTML, XML, CSV, SQL or a
shell script. Escape where the format needs it:

| Output | Use |
|---|---|
| JSON | `toJSON` for text values, or `toJSON` on the whole structure |
| XML, HTML | `html` on every value, or `toXML` |
| CSV | quotes around values, `replaceAll "\"" "\"\""` inside, `csvSafe` against formulas |
| SQL | `replaceAll "'" "''"` inside single-quoted values |

See [Security notes](security.md#no-output-escaping).
