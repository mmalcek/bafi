# Custom logic with Lua

When template functions are not enough - parsing a complex field, a checksum, a lookup table - write
the logic in [Lua](https://www.lua.org/manual/5.1/) and call it from the template.

## 1. Write the function

bafi loads `./lua/functions.lua` from the **current directory** the first time a template calls `lua`.
Every function receives its arguments as one JSON-encoded array and must return a string. The
`json.lua` helper that ships with bafi decodes the arguments:

<!-- test:file lua/functions.lua -->
```lua
json = require './lua/json'

-- IBAN check: move the first 4 characters to the end, letters to numbers, mod 97 must be 1
function ibanValid(incomingData)
    local iban = string.upper(string.gsub(json.decode(incomingData)[1], "%s", ""))
    local moved = string.sub(iban, 5) .. string.sub(iban, 1, 4)
    local digits = string.gsub(moved, "%a", function(c) return tostring(string.byte(c) - 55) end)
    local remainder = 0
    for i = 1, #digits do
        remainder = (remainder * 10 + tonumber(string.sub(digits, i, i))) % 97
    end
    return tostring(remainder == 1)
end
```

`lua/json.lua` must sit next to it (it is included in every release archive).

## 2. Call it from a template

<!-- test:file accounts.csv -->
```
owner,iban
Alice,CZ65 0800 0000 1920 0014 5399
Bob,CZ65 0800 0000 1920 0014 5398
```

<!-- test:cmd -->
```sh
bafi -i accounts.csv -t '?{{range .}}{{.owner}}: {{lua "ibanValid" .iban}}{{"\n"}}{{end}}'
```

<!-- test:output -->
```
Alice: true
Bob: false
```

Arguments are passed as `{{lua "functionName" arg1 arg2 ...}}`; in Lua, `json.decode(incomingData)[1]`
is the first one (Lua lists start at 1).

## Errors

Problems inside Lua do not stop the template; the function's place in the output shows the message
instead, for example `luaError: ... attempt to call a non-function object` for a misspelled function
name. Search the output for `luaError` when something looks wrong.

## Security

`./lua/functions.lua` is code, and it runs with your permissions. Only use a `lua/` directory you
trust. bafi removes the functions that start programs or delete files (`os.execute`, `io.popen`,
`os.remove`, ...); the rest of the Lua standard library works. More in [Security notes](../security.md).

The shipped example defines `sum` and `mul`: `{{lua "sum" .val1 .val2}}`.

Related: [MT940 bank statement to CSV](mt940-to-csv.md) · [Function reference](../functions.md)
