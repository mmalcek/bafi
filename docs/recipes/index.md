# Recipes

Each recipe solves one task end to end: sample input, the command, the template and the exact output.
The sample commands are run by bafi's test suite, so their output is exactly what you get. Lines that
need outside services (`curl`, `mongodump`, OpenAI) are shown but not run.

## Convert between formats

- [Convert XML to JSON](xml-to-json.md)
- [Convert JSON to XML](json-to-xml.md)
- [Convert CSV to JSON](csv-to-json.md)
- [Convert JSON to CSV](json-to-csv.md)
- [Convert XML to CSV](xml-to-csv.md)
- [Convert between JSON, YAML, TOON and BSON](convert-formats.md)

## Integrate services

- [Turn a REST (JSON) response into a SOAP request](rest-to-soap.md)
- [Turn a SOAP response into JSON](soap-to-json.md)
- [Query any SQL database into XML or JSON](sql-database.md)
- [Convert a MongoDB dump (BSON) to CSV](bson-to-csv.md)

## Reports and generated files

- [Generate an HTML report from JSON](json-to-html.md)
- [Generate SQL INSERT statements from CSV](csv-to-sql.md)
- [Merge several files into one output](merge-files.md)

## Finance

- [Convert an MT940 bank statement to CSV](mt940-to-csv.md)

## Going further

- [Custom logic with Lua](lua.md)
- [Ask OpenAI about your data](openai.md)

Missing a recipe? Ask in [GitHub Discussions](https://github.com/mmalcek/bafi/discussions).
