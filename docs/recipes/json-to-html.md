# Generate an HTML report from JSON

Render JSON into a self-contained HTML page: a table, totals, conditional styling. Open it in a
browser, attach it to an e-mail, or publish it from a scheduled job.

## Sample input

<!-- test:file sales.json -->
```json
{
  "region": "Central Europe",
  "sales": [
    {"rep": "Alice", "amount": 1200.5, "target": 1000},
    {"rep": "Bob <b>", "amount": 800, "target": 1000}
  ]
}
```

## Template

`html` escapes text so that data such as `Bob <b>` cannot break the page. `addf` adds decimals exactly;
`ge` compares numbers.

<!-- test:file report.tmpl -->
```html
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Sales {{html .region}}</title>
<style>td, th { padding: 4px 12px; } .miss { color: #b00; }</style></head>
<body>
<h1>Sales: {{html .region}}</h1>
<table>
  <tr><th>Rep</th><th>Amount</th><th>Target</th></tr>
  {{- $total := 0.0}}
  {{- range .sales}}
  {{- $total = addf $total .amount}}
  <tr{{if not (ge .amount .target)}} class="miss"{{end}}><td>{{html .rep}}</td><td>{{.amount}}</td><td>{{.target}}</td></tr>
  {{- end}}
</table>
<p>Total: {{$total}}</p>
</body>
</html>
```

<!-- test:cmd -->
```sh
bafi -i sales.json -t report.tmpl
```

<!-- test:output -->
```html
<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Sales Central Europe</title>
<style>td, th { padding: 4px 12px; } .miss { color: #b00; }</style></head>
<body>
<h1>Sales: Central Europe</h1>
<table>
  <tr><th>Rep</th><th>Amount</th><th>Target</th></tr>
  <tr><td>Alice</td><td>1200.5</td><td>1000</td></tr>
  <tr class="miss"><td>Bob &lt;b&gt;</td><td>800</td><td>1000</td></tr>
</table>
<p>Total: 2000.5</p>
</body>
</html>
```

Save it with `-o report.html`. On Windows you can open it right away: `.\bafi.exe -i sales.json -t report.tmpl -o report.html; start report.html`.

Related: [Merge several files into one output](merge-files.md) · [Template guide](../templates.md)
