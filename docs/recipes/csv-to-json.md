# Convert CSV to JSON

Turn a CSV file (an export from Excel, a database or another tool) into JSON. Each row becomes an
object whose keys are the column names from the header line.

## Sample input

<!-- test:file users.csv -->
```
id,name,email,active
1,Alice Smith,alice@example.com,true
2,"Brown, Bob",bob@example.com,false
```

Values that contain the delimiter are quoted (`"Brown, Bob"`); that is standard CSV
([RFC 4180](https://datatracker.ietf.org/doc/html/rfc4180)).

## One-line conversion

<!-- test:cmd -->
```sh
bafi -i users.csv -t '?{{toJSON .}}'
```

<!-- test:output -->
```json
[
  {
    "active": "true",
    "email": "alice@example.com",
    "id": "1",
    "name": "Alice Smith"
  },
  {
    "active": "false",
    "email": "bob@example.com",
    "id": "2",
    "name": "Brown, Bob"
  }
]
```

Every CSV value is text, so the generic conversion produces strings only.

## Numbers and booleans

Write the JSON yourself to choose the types. `toJSON` quotes and escapes text values:

<!-- test:file users-json.tmpl -->
```
[
{{- range $i, $u := .}}{{if $i}},{{end}}
  {"id": {{$u.id}}, "name": {{toJSON $u.name}}, "active": {{$u.active}}}
{{- end}}
]
```

<!-- test:cmd -->
```sh
bafi -i users.csv -t users-json.tmpl
```

<!-- test:output -->
```json
[
  {"id": 1, "name": "Alice Smith", "active": true},
  {"id": 2, "name": "Brown, Bob", "active": false}
]
```

## Semicolons, tabs and other delimiters

Many European exports use `;`. Set the delimiter with `-d`: any single character, or a hex byte such
as `0x09` for TAB.

<!-- test:file export.csv -->
```
id;name
1;Alice Smith
```

<!-- test:cmd -->
```sh
bafi -i export.csv -d ';' -t '?{{toJSON .}}'
```

<!-- test:output -->
```json
[
  {
    "id": "1",
    "name": "Alice Smith"
  }
]
```

If you forget `-d`, bafi warns that the whole header ended up in a single column.

Related: [JSON to CSV](json-to-csv.md) · [SQL INSERT statements from CSV](csv-to-sql.md)
