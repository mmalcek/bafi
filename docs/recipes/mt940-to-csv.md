# Convert an MT940 bank statement to CSV

Turn a SWIFT MT940 statement (`.sta`) into CSV for accounting software or a spreadsheet.

## What the parsed statement looks like

bafi splits the message into fields named after their tags, prefixed with `F_`:

- `.Header` - the header blocks `{1:...}{2:...}`
- `.Fields` - statement-level fields: `:20:` is `.Fields.F_20`, `:60F:` is `.Fields.F_60F`, ...
- `.Transactions` - one entry per `:61:` line, with its `:86:` details: `.F_61`, `.F_86`

The values are the raw field texts; split them in the template (or in [Lua](lua.md)).

## Sample statement

<!-- test:file statement.sta crlf -->
```
{1:F01BANKCZPPAXXX0000000000}{2:I940BANKCZPPXXXXN}{4:
:20:STMT-2024-001
:25:CZ6508000000192000145399
:28C:1/1
:60F:C240301EUR1000,00
:61:2403040304C250,00NTRFNONREF
:86:Invoice 2024-17 paid by Alice Smith
:61:2403050305D99,90NTRFNONREF
:86:Office supplies
:62F:C240305EUR1150,10
-}
```

## Template

`slice` cuts fixed positions out of `:61:` (date `YYMMDD`, then `C`/`D`), and `dateFormat` turns the
date into `DD.MM.YYYY`.

<!-- test:file statement-csv.tmpl -->
```
Statement,Date,Type,Details
{{- $ref := .Fields.F_20}}
{{- range .Transactions}}
{{$ref}},{{dateFormat (slice .F_61 0 6) "060102" "02.01.2006"}},{{if eq (slice .F_61 10 11) "C"}}credit{{else}}debit{{end}},"{{.F_86}}"
{{- end}}
```

<!-- test:cmd -->
```sh
bafi -i statement.sta -t statement-csv.tmpl
```

<!-- test:output -->
```
Statement,Date,Type,Details
STMT-2024-001,04.03.2024,credit,"Invoice 2024-17 paid by Alice Smith"
STMT-2024-001,05.03.2024,debit,"Office supplies"
```

`.sta` files are recognised by their extension; for other names add `-f mt940`.

## Several statements in one file

Exports such as Multicash put many messages in one file. Pass the separator between messages with
`-d`; bafi then returns a list of statements:

```sh
bafi -i multicash.sta -d "-}\r\n" -t '?{{range .}}{{.Fields.F_20}} {{end}}'
```

`\r` and `\n` in `-d` are turned into real line breaks.

Related: [Custom logic with Lua](lua.md) · [Template guide](../templates.md)
