# Convert JSON to CSV

Turn JSON, typically a list of objects from an API, into a CSV file that opens cleanly in Excel or
LibreOffice.

## Sample input

<!-- test:file users.json -->
```json
{
  "users": [
    {"id": 1, "name": "Alice Smith", "email": "alice@example.com"},
    {"id": 2, "name": "Brown, Bob", "email": "bob@example.com"},
    {"id": 3, "name": "=HYPERLINK(\"http://evil.example\")", "email": "x@example.com"}
  ]
}
```

## Template

The header is a plain line; `range` writes one line per user. `{{-` trims the line break before it,
so no empty lines appear.

<!-- test:file users-csv.tmpl -->
```
id,name,email
{{- range .users}}
{{.id}},"{{replaceAll "\"" "\"\"" (csvSafe .name)}}",{{.email}}
{{- end}}
```

<!-- test:cmd -->
```sh
bafi -i users.json -t users-csv.tmpl
```

<!-- test:output -->
```
id,name,email
1,"Alice Smith",alice@example.com
2,"Brown, Bob",bob@example.com
3,"'=HYPERLINK(""http://evil.example"")",x@example.com
```

What the template takes care of:

- **Commas inside values**: the name is wrapped in quotes, and quotes inside it are doubled
  (`replaceAll "\"" "\"\""`), as CSV requires.
- **Formula injection**: `csvSafe` prefixes values starting with `=`, `+`, `-`, `@` with `'`, so a
  spreadsheet shows them as text instead of executing them. Plain numbers such as `-5` are left alone.
  Use it whenever the data comes from outside.

## Write to a file

<!-- test:cmd quiet -->
```sh
bafi -i users.json -t users-csv.tmpl -o users.csv
```

From an API: `curl -fs https://example.com/api/users | bafi -f json -t users-csv.tmpl -o users.csv`

Related: [CSV to JSON](csv-to-json.md) · [XML to CSV](xml-to-csv.md)
