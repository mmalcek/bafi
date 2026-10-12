# Query any SQL database into XML or JSON

bafi does not connect to databases itself. Combine it with [usql](https://github.com/xo/usql), a
universal SQL client for PostgreSQL, MySQL, SQL Server, Oracle, SQLite and more, which prints query
results as JSON (`-J`):

```sh
usql postgres://user:pass@db.example.com/shop -J -q -c "SELECT id, name, email FROM users" \
  | bafi -f json -t '?{{toXML .}}'
```

## What the query result looks like

`usql -J` prints a JSON array with one object per row:

<!-- test:file result.json -->
```json
[
  {"id": 1, "name": "Alice Smith", "email": "alice@example.com"},
  {"id": 2, "name": "Bob Brown", "email": "bob@example.com"}
]
```

## Rows to XML

<!-- test:cmd -->
```sh
cat result.json | bafi -f json -t '?{{toXML .}}'
```

<!-- test:output -->
```xml
<doc>
  <record>
    <email>alice@example.com</email>
    <id>1</id>
    <name>Alice Smith</name>
  </record>
  <record>
    <email>bob@example.com</email>
    <id>2</id>
    <name>Bob Brown</name>
  </record>
</doc>
```

A top-level array becomes one `<record>` per row inside `<doc>`. For your own element names write the
XML in a template, as in [JSON to XML](json-to-xml.md#your-own-xml-layout).

The same works with any client that prints JSON or CSV, for example `psql --csv` (then `-f csv`) or
`sqlite3 -json`.

Related: [JSON to XML](json-to-xml.md) · [SQL INSERT statements from CSV](csv-to-sql.md)
