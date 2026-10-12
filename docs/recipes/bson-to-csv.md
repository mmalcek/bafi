# Convert a MongoDB dump (BSON) to CSV

Read `.bson` files written by `mongodump` (or a single BSON document) and turn them into CSV, JSON or
any other text, without a running MongoDB.

```sh
mongodump --db shop --collection users --out dump
bafi -i dump/shop/users.bson -t users-csv.tmpl -o users.csv
```

A `mongodump` file holds one document per record; bafi reads it as a list, so the template iterates
with `range .`. A file with a single document is read as that document.

## Try it without MongoDB

Make a small BSON file from JSON with `toBSON`, then read it back:

<!-- test:file user.json -->
```json
{"name": "Alice Smith", "email": "alice@example.com", "roles": ["admin", "billing"]}
```

<!-- test:cmd quiet -->
```sh
bafi -i user.json -t '?{{toBSON .}}' -o user.bson
```

<!-- test:file user-csv.tmpl -->
```
name,email,roles
"{{.name}}",{{.email}},"{{range $i, $r := .roles}}{{if $i}};{{end}}{{$r}}{{end}}"
```

<!-- test:cmd -->
```sh
bafi -i user.bson -t user-csv.tmpl
```

<!-- test:output -->
```
name,email,roles
"Alice Smith",alice@example.com,"admin;billing"
```

For a dump with many users, wrap the data line in `{{range .}}...{{end}}`.

## Inspect a dump

When you do not know the structure of a collection, print the first record:

```sh
bafi -i dump/shop/users.bson -t '?{{toJSON (index . 0)}}'
```

MongoDB's own types (ObjectId, dates) are printed in their Go form; convert them in the template or with
`toString`.

Related: [Convert between JSON, YAML, TOON and BSON](convert-formats.md) · [JSON to CSV](json-to-csv.md)
