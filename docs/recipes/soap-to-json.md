# Turn a SOAP response into JSON

Call a SOAP service and hand a clean JSON document to a REST client, a script or a web page.

```sh
curl -fs -H 'Content-Type: text/xml' --data-binary @get-order.xml https://erp.example.com/OrderService \
  | bafi -f xml -t order-json.tmpl
```

## The SOAP response

<!-- test:file response.xml -->
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:m="http://example.com/orders">
  <soap:Body>
    <m:GetOrderResponse>
      <m:Order id="1001">
        <m:Customer>Alice Smith</m:Customer>
        <m:Total currency="EUR">120.50</m:Total>
      </m:Order>
    </m:GetOrderResponse>
  </soap:Body>
</soap:Envelope>
```

## What bafi sees

Namespace prefixes are dropped (`soap:Body` is `Body`); namespace declarations show up as attributes
(`-soap`, `-m`). Print the parsed structure whenever you are unsure of a path:

<!-- test:cmd -->
```sh
bafi -i response.xml -t '?{{toJSON .Envelope.Body}}'
```

<!-- test:output -->
```json
{
  "GetOrderResponse": {
    "Order": {
      "-id": "1001",
      "Customer": "Alice Smith",
      "Total": {
        "#text": "120.50",
        "-currency": "EUR"
      }
    }
  }
}
```

## Template

<!-- test:file order-json.tmpl -->
```
{{- with .Envelope.Body.GetOrderResponse.Order -}}
{"id": {{index . "-id"}}, "customer": {{toJSON .Customer}}, "total": {{index .Total "#text"}}, "currency": {{toJSON (index .Total "-currency")}}}
{{- end}}
```

<!-- test:cmd -->
```sh
cat response.xml | bafi -f xml -t order-json.tmpl
```

<!-- test:output -->
```json
{"id": 1001, "customer": "Alice Smith", "total": 120.50, "currency": "EUR"}
```

`with` sets the dot to the order, so the template does not repeat the long path. If the service can
return a SOAP fault instead, test for it with `{{if .Envelope.Body.Fault}}...{{end}}`.

Related: [REST response to SOAP request](rest-to-soap.md) · [XML to JSON](xml-to-json.md)
