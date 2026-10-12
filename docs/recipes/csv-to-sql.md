# Generate SQL INSERT statements from CSV

Load a CSV export into a database without an import wizard: generate `INSERT` statements and run them
with your usual SQL client.

## Sample input

<!-- test:file products.csv -->
```
sku,name,price
A-1,Coffee beans,12.90
B-7,O'Reilly mug,9.50
```

## Template

Text values go in single quotes, so a quote inside a value must be doubled: `O'Reilly` becomes
`'O''Reilly'`. `replaceAll "'" "''"` does exactly that.

<!-- test:file insert.tmpl -->
```sql
INSERT INTO products (sku, name, price) VALUES
{{- range $i, $p := .}}{{if $i}},{{end}}
  ('{{replaceAll "'" "''" $p.sku}}', '{{replaceAll "'" "''" $p.name}}', {{toFloat64 $p.price}})
{{- end}};
```

<!-- test:cmd -->
```sh
bafi -i products.csv -t insert.tmpl
```

<!-- test:output -->
```sql
INSERT INTO products (sku, name, price) VALUES
  ('A-1', 'Coffee beans', 12.9),
  ('B-7', 'O''Reilly mug', 9.5);
```

`toFloat64` makes sure the price is a number; a malformed value becomes `0` instead of breaking the
SQL. Pipe the result straight into the client, for example
`bafi -i products.csv -t insert.tmpl | psql shop` or `| sqlite3 shop.db`.

**Note:** escaping quotes is enough for trusted exports. For data from untrusted sources prefer your
database's bulk loader (`COPY`, `LOAD DATA`, `bcp`) with the CSV itself.

Related: [CSV to JSON](csv-to-json.md) · [Query a SQL database into XML or JSON](sql-database.md)
