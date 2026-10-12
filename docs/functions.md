# Function reference

Templates call functions in prefix notation: the function name comes first, then its arguments.
`{{add 1 2}}` adds 1 and 2; nested calls use parentheses: `{{upper (trim .name)}}`. Values from JSON,
YAML, XML and CSV are converted automatically where it makes sense, so `{{add "1" 2}}` works too.

Every example on this page is executed by the test suite; the result column is what bafi prints.
Rows marked `*` change on every run.

## Math functions

<!-- test:table -->
| Function | Example | Result |
|---|---|---|
| **add** - sum of integers | `{{add 1 2 "3"}}` | `6` |
| **add1** - plus one | `{{add1 41}}` | `42` |
| **sub** - subtract | `{{sub 10 4}}` | `6` |
| **mul** - product of integers | `{{mul 2 3 4}}` | `24` |
| **div** - integer division | `{{div 7 2}}` | `3` |
| **mod** - remainder | `{{mod 7 2}}` | `1` |
| **addf** - exact decimal sum | `{{addf 0.1 0.2}}` | `0.3` |
| **add1f** - plus one, decimal | `{{add1f 1.5}}` | `2.5` |
| **subf** - decimal subtract | `{{subf 1.5 0.2 0.3}}` | `1` |
| **mulf** - decimal product | `{{mulf 1.1 3}}` | `3.3` |
| **divf** - decimal division | `{{divf 10 4}}` | `2.5` |
| **round** - to N decimals | `{{round 3.14159 2}}` | `3.14` |
| **round** - custom rounding point | `{{round 6.35 1 0.4}}` | `6.4` |
| **max** - largest integer | `{{max 3 9 "5"}}` | `9` |
| **min** - smallest integer | `{{min 3 9 "5"}}` | `3` |
| **maxf** - largest number | `{{maxf 1.5 2.25}}` | `2.25` |
| **minf** - smallest number | `{{minf 1.5 2.25}}` | `1.5` |
| **randInt** - random integer, both bounds included | `{{randInt 1 6}}` | * |

The `...f` functions calculate with exact decimals ([shopspring/decimal](https://github.com/shopspring/decimal)),
so `addf 0.1 0.2` is `0.3`, not `0.30000000000000004`. `randInt` is not cryptographically secure; never
use it for tokens or passwords (use `uuid` for identifiers).

## Date functions

Dates use Go's reference layout: write the date **Mon Jan 2 15:04:05 MST 2006** the way your format
looks. `2006-01-02` is YYYY-MM-DD, `02.01.2006` is DD.MM.YYYY, `15:04` is HH:MM.
[More about the layout](https://pkg.go.dev/time#pkg-constants).

<!-- test:table -->
| Function | Example | Result |
|---|---|---|
| **dateFormat** - change the format | `{{dateFormat "2021-08-26" "2006-01-02" "02.01.2006"}}` | `26.08.2021` |
| **dateFormat** - unparsable input is returned unchanged | `{{dateFormat "n/a" "2006-01-02" "02.01.2006"}}` | `n/a` |
| **dateFormatTZ** - change format and time zone | `{{dateFormatTZ "2021-08-26T03:35:00+04:00" "2006-01-02T15:04:05-07:00" "15:04" "Europe/Prague"}}` | `01:35` |
| **dateToInt** - date to Unix time | `{{dateToInt "2021-03-15" "2006-01-02"}}` | `1615766400` |
| **intToDate** - Unix time to date (local time zone) | `{{intToDate 1615766400 "2006-01-02"}}` | * |
| **now** - current date and time | `{{now "2006-01-02 15:04"}}` | * |

[Time zone names](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) for `dateFormatTZ`.

## String functions

<!-- test:table -->
| Function | Example | Result |
|---|---|---|
| **upper** | `{{upper "Hello"}}` | `HELLO` |
| **lower** | `{{lower "Hello"}}` | `hello` |
| **trim** - remove surrounding whitespace | `{{trim "  a b  "}}` | `a b` |
| **trimAll** - remove characters from both ends | `{{trimAll "!Hello!" "!"}}` | `Hello` |
| **trimPrefix** | `{{trimPrefix "item-42" "item-"}}` | `42` |
| **trimSuffix** | `{{trimSuffix "a, b, " ", "}}` | `a, b` |
| **contains** | `{{contains "invoice-2024" "2024"}}` | `true` |
| **indexOf** - position of a substring (-1 if absent) | `{{indexOf "abcd" "cd"}}` | `2` |
| **replaceAll** - replace text | `{{replaceAll "x" "Z" "aaxbb"}}` | `aaZbb` |
| **replaceAllRegex** - replace a regular expression | `{{replaceAllRegex "[0-9]+" "#" "a1b22"}}` | `a#b#` |
| **regexMatch** - test a regular expression | `{{regexMatch "^[0-9]{4}$" "2024"}}` | `true` |
| **addSubstring** - insert, position counted from the right | `{{addSubstring "Hello!!!" " World" 3}}` | `Hello World!!!` |
| **addSubstring** - negative position counts from the left | `{{addSubstring "Hello!!!" " World" -5}}` | `Hello World!!!` |
| **csvSafe** - defuse spreadsheet formulas | `{{csvSafe "=1+1"}}` | `'=1+1` |
| **csvSafe** - plain numbers stay | `{{csvSafe "-5"}}` | `-5` |
| **b64enc** - Base64 encode | `{{b64enc "Hello"}}` | `SGVsbG8=` |
| **b64dec** - Base64 decode | `{{b64dec "SGVsbG8="}}` | `Hello` |
| **b32enc** - Base32 encode | `{{b32enc "Hi"}}` | `JBUQ====` |
| **b32dec** - Base32 decode | `{{b32dec "JBUQ===="}}` | `Hi` |
| **uuid** - random UUID v4 | `{{uuid}}` | * |

Regular expressions use [Go syntax](https://pkg.go.dev/regexp/syntax). An invalid expression in
`replaceAllRegex` stops the template with an error. `addSubstring` counts positions in bytes; use it on
ASCII text.

## Conversion and type checks

<!-- test:table -->
| Function | Example | Result |
|---|---|---|
| **toString** | `{{toString 42}}` | `42` |
| **toInt** | `{{toInt "42"}}` | `42` |
| **toInt64** | `{{toInt64 "9000000000"}}` | `9000000000` |
| **atoi** - text to integer, keeps leading zeros meaningless | `{{atoi "042"}}` | `42` |
| **toFloat64** | `{{toFloat64 "3.5"}}` | `3.5` |
| **toBool** | `{{toBool "true"}}` | `true` |
| **toDecimal** - exact decimal (0 if invalid) | `{{toDecimal "3.10"}}` | `3.1` |
| **toDecimalString** - exact decimal as text (error text if invalid) | `{{toDecimalString "3.10"}}` | `3.1` |
| **isString** | `{{isString "x"}}` | `true` |
| **isInt** | `{{isInt 1}}` | `true` |
| **isFloat64** - JSON numbers are float64 | `{{isFloat64 1.5}}` | `true` |
| **isBool** | `{{isBool true}}` | `true` |
| **isMap** | `{{isMap (mapJSON "{}")}}` | `true` |
| **isArray** | `{{isArray (mustArray "x")}}` | `true` |
| **mustArray** - single value to a one-item list, lists unchanged | `{{len (mustArray "x")}}` | `1` |

Use `atoi` for text with leading zeros such as `"042"`; `toInt` may read those as octal.

## Data formats

<!-- test:table -->
| Function | Example | Result |
|---|---|---|
| **toJSON** - JSON (escapes text correctly) | `{{toJSON "say \"hi\""}}` | `"say \"hi\""` |
| **toYAML** | `{{toYAML (mapJSON "{\"a\":1}")}}` | `a: 1` |
| **toTOON** - compact format for LLM prompts | `{{toTOON (mapJSON "{\"a\":1}")}}` | `a: 1` |
| **toXML** - XML (see [JSON to XML](recipes/json-to-xml.md)) | `{{toXML (mapJSON "{\"a\":1}")}}` | * |
| **toBSON** - binary BSON (see [BSON recipe](recipes/bson-to-csv.md)) | `{{toBSON (mapJSON "{\"a\":1}")}}` | * |
| **mapJSON** - parse a JSON string into data | `{{(mapJSON "{\"a\":1}").a}}` | `1` |
| **lua** - call a Lua function (see [Lua](recipes/lua.md)) | `{{lua "sum" 1 2}}` | * |

`toJSON`, `toYAML`, `toXML` and `toTOON` work on any value, including the whole input (`{{toJSON .}}`)
or one part of it (`{{toJSON .users}}`).

## Built-in template functions

These come with Go's [text/template](https://pkg.go.dev/text/template#hdr-Functions):

<!-- test:table -->
| Function | Example | Result |
|---|---|---|
| **len** - length of text, list or map | `{{len "abc"}}` | `3` |
| **index** - item of a list or map; keys with `-` or spaces | `{{index (mapJSON "{\"my-key\":7}") "my-key"}}` | `7` |
| **slice** - part of text or a list | `{{slice "abcdef" 1 3}}` | `bc` |
| **printf** - formatted text | `{{printf "%05.2f" 3.14159}}` | `03.14` |
| **print** - join values | `{{print "a" 1}}` | `a1` |
| **eq** / **ne** - equal, not equal | `{{eq 1 1}}` | `true` |
| **lt** / **le** / **gt** / **ge** - compare | `{{gt 5 3}}` | `true` |
| **and** / **or** / **not** | `{{and true (not false)}}` | `true` |
| **html** - escape for HTML and XML | `{{html "<a & b>"}}` | `&lt;a &amp; b&gt;` |
| **urlquery** - escape for a URL | `{{urlquery "a b&c"}}` | `a+b%26c` |
| **js** - escape for JavaScript strings | `{{js "it's"}}` | `it\'s` |

Comparisons need values of the same kind: compare a JSON number with a number (`{{gt .total 100}}`), and
text from CSV or XML after converting it (`{{gt (toInt .total) 100}}`).
