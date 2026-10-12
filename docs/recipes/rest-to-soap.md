# Turn a REST (JSON) response into a SOAP request

Bridge a REST service and a SOAP service in one pipe: fetch JSON, render the SOAP envelope the other
side expects, and post it.

```sh
curl -fs https://shop.example.com/api/orders/1001 \
  | bafi -f json -t create-invoice.tmpl \
  | curl -fs -H 'Content-Type: text/xml; charset=utf-8' -H 'SOAPAction: "CreateInvoice"' \
         --data-binary @- https://erp.example.com/InvoiceService
```

## The JSON coming from REST

<!-- test:file order.json -->
```json
{
  "id": 1001,
  "customer": {"name": "Alice & Co.", "vat": "CZ12345678"},
  "items": [
    {"sku": "A-1", "qty": 2, "price": 10.5},
    {"sku": "B-7", "qty": 1, "price": 99}
  ]
}
```

## The SOAP template

Every text value goes through `html`, which escapes `&`, `<`, `>` and quotes, so the envelope stays
valid XML whatever the data contains. `mulf` multiplies decimals exactly (no `20.999999` surprises).

<!-- test:file create-invoice.tmpl -->
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:inv="http://erp.example.com/invoice">
  <soap:Body>
    <inv:CreateInvoice>
      <inv:OrderId>{{.id}}</inv:OrderId>
      <inv:Customer vat="{{html .customer.vat}}">{{html .customer.name}}</inv:Customer>
      {{- range .items}}
      <inv:Line sku="{{html .sku}}" quantity="{{.qty}}" amount="{{mulf .qty .price}}"/>
      {{- end}}
    </inv:CreateInvoice>
  </soap:Body>
</soap:Envelope>
```

## Try it locally

<!-- test:cmd -->
```sh
cat order.json | bafi -f json -t create-invoice.tmpl
```

<!-- test:output -->
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:inv="http://erp.example.com/invoice">
  <soap:Body>
    <inv:CreateInvoice>
      <inv:OrderId>1001</inv:OrderId>
      <inv:Customer vat="CZ12345678">Alice &amp; Co.</inv:Customer>
      <inv:Line sku="A-1" quantity="2" amount="21"/>
      <inv:Line sku="B-7" quantity="1" amount="99"/>
    </inv:CreateInvoice>
  </soap:Body>
</soap:Envelope>
```

Check the request before sending it: run the first two commands of the pipe and read the output, or
save it with `-o request.xml`.

Related: [SOAP response to JSON](soap-to-json.md) · [JSON to XML](json-to-xml.md)
