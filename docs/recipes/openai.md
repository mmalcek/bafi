# Ask OpenAI about your data

Instead of writing a template, describe the result in plain language and let an OpenAI model produce it:
a summary, an answer, or a conversion that is hard to express as a template. Experimental.

```sh
export OPENAI_API_KEY=sk-...
curl -fs https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml \
  | bafi -f xml -gq "What is the current CZK rate? Answer with the number only."
```

- `-gq` is the question; it switches bafi into OpenAI mode, no `-t` needed.
- The key comes from the `OPENAI_API_KEY` environment variable. `-gk` also works but puts the key in
  your shell history and the process list.
- `-gm` picks the model: `gpt35` (default), `gpt4`, `gpt4o`, `gpt4o-mini`, `gpt5`, `gpt5mini`, `gpt5nano`.
- `-o answer.txt` saves the answer to a file.

## What is sent

bafi sends the question **and the whole parsed input**, encoded as [TOON](convert-formats.md#to-toon-for-llm-prompts)
to save tokens. Do not use it with data that must not leave your network. Preview exactly what goes out:

<!-- test:file invoice.json -->
```json
{"invoice": "2024-17", "customer": "Alice Smith", "lines": [{"item": "Coffee", "qty": 2, "price": 12.9}]}
```

<!-- test:cmd -->
```sh
bafi -i invoice.json -t '?{{toTOON .}}'
```

<!-- test:output -->
```
customer: Alice Smith
invoice: 2024-17
lines[1]{item,price,qty}:
  Coffee,12.9,2
```

## Examples

```sh
bafi -i invoice.json -gq "Create a UBL 2.1 XML invoice from this data" -gm gpt4o -o invoice.xml
curl -fs "https://api.open-meteo.com/v1/forecast?latitude=50.08&longitude=14.42&hourly=temperature_2m" \
  | bafi -f json -gm gpt4o-mini -gq "Summarise tomorrow's temperatures in two sentences"
```

Model output is not deterministic; for repeatable conversions write a template instead and keep OpenAI
for exploration.

Related: [Convert between JSON, YAML, TOON and BSON](convert-formats.md) · [Security notes](../security.md)
