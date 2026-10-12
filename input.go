package main

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/clbanning/mxj/v2"
	"github.com/mmalcek/mt940"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v3"
)

// inputFormats lists the accepted -f values (case-insensitive), in the order shown to users.
var inputFormats = []string{"json", "bson", "yaml", "csv", "xml", "mt940"}

// inputSpec describes one input document: where it came from (for error messages) and how to parse it.
type inputSpec struct {
	Source    string // "stdin", a file path, or "files.yaml entry 2 (rates.xml)"
	Format    string // one of inputFormats, any case
	Delimiter string // -d value: CSV delimiter or MT940 message separator
}

// detectFormat guesses the input format from the file extension; "" when the extension is unknown.
func detectFormat(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "json"
	case ".bson":
		return "bson"
	case ".yaml", ".yml":
		return "yaml"
	case ".csv":
		return "csv"
	case ".sta":
		return "mt940"
	case ".xml", ".cdf", ".cdf3":
		return "xml"
	default:
		return ""
	}
}

// knownFormat reports whether format (any case) is a supported -f value.
func knownFormat(format string) bool {
	for _, f := range inputFormats {
		if strings.EqualFold(f, format) {
			return true
		}
	}
	return false
}

// sniffFormat guesses a format from the first non-space byte. Used only to word a hint;
// bafi never switches format on its own.
func sniffFormat(data []byte) string {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return ""
	}
	switch trimmed[0] {
	case '{', '[':
		return "json"
	case '<':
		return "xml"
	}
	return ""
}

// mapInputData parses data according to spec. warn (may be nil) receives non-fatal findings such as a
// probably wrong CSV delimiter. Errors are *bafiError values pointing at the offending line.
func mapInputData(data []byte, spec inputSpec, warn func(string)) (interface{}, error) {
	switch strings.ToLower(spec.Format) {
	case "json":
		return parseJSON(data, spec.Source)
	case "bson":
		return parseBSON(data, spec.Source)
	case "yaml":
		return parseYAML(data, spec.Source)
	case "csv":
		return parseCSV(data, spec.Delimiter, spec.Source, warn)
	case "xml":
		mapData, err := mxj.NewMapXml(data)
		if err != nil {
			return nil, xmlError(data, spec.Source, err)
		}
		return mapData, nil
	case "mt940":
		var result interface{}
		var err error
		if spec.Delimiter == "" {
			result, err = mt940.Parse(data)
		} else {
			delimiter := strings.ReplaceAll(spec.Delimiter, `\r`, "\r")
			delimiter = strings.ReplaceAll(delimiter, `\n`, "\n")
			result, err = mt940.ParseMultimessage(data, delimiter)
		}
		if err != nil {
			if len(bytes.TrimSpace(data)) == 0 {
				return nil, emptyInputError(spec.Source, "MT940")
			}
			detail := err.Error()
			if explained, ok := mt940Errors[detail]; ok {
				detail = explained + " (" + detail + ")"
			}
			return nil, &bafiError{
				Stage:   stageInput,
				Summary: "cannot parse MT940 input " + sourcePhrase(spec.Source),
				Detail:  detail,
				Hint: `an MT940 message starts with the {1:...}{2:...}{4: header, uses CRLF line endings and ends with -}; ` +
					`for files with several messages (e.g. Multicash) set the separator, e.g. -d "-}\r\n"`,
				Anchor: "invalid-input",
				Err:    err,
			}
		}
		return result, nil
	default:
		if spec.Format == "" {
			return nil, missingFormatError(spec.Source, data)
		}
		return nil, unknownFormatError(spec.Format)
	}
}

// mt940Errors explains the parser's internal error codes.
var mt940Errors = map[string]string{
	"headerStartNotFound": "no {1: header block found",
	"headerEndNotFound":   "the {4: block that starts the message text was not found",
	"headerNotFound":      "the message header is empty",
	"messageEndNotFound":  `no "-}" message end found - are the line endings CRLF?`,
	"fieldsNotFound":      "no :NN: fields found in the message",
}

// parseJSON accepts an object or an array of objects at the top level, exactly as before 1.5.0:
// objects become map[string]interface{}, arrays []map[string]interface{}.
func parseJSON(data []byte, source string) (interface{}, error) {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) > 0 && trimmed[0] == '[' {
		mapDataArray := make([]map[string]interface{}, 0)
		if err := json.Unmarshal(data, &mapDataArray); err != nil {
			return nil, jsonError(data, source, err)
		}
		return mapDataArray, nil
	}
	var mapData map[string]interface{}
	if err := json.Unmarshal(data, &mapData); err != nil {
		return nil, jsonError(data, source, err)
	}
	return mapData, nil
}

// parseBSON reads a single BSON document or a mongodump stream of documents.
func parseBSON(data []byte, source string) (result interface{}, err error) {
	original := data
	wrap := func(cause error) error {
		if len(bytes.TrimSpace(original)) == 0 {
			return emptyInputError(source, "BSON")
		}
		return &bafiError{
			Stage:   stageInput,
			Summary: "cannot parse BSON input " + sourcePhrase(source),
			Detail:  cause.Error(),
			Hint:    "the input must be a BSON document or a mongodump .bson file (not MongoDB extended JSON; use -f json for that)",
			Anchor:  "invalid-input",
			Err:     cause,
		}
	}
	// The mongo-driver decoder panics on some malformed documents (slice bounds out of range) instead of
	// returning an error; report those like any other broken input instead of crashing.
	defer func() {
		if r := recover(); r != nil {
			result, err = nil, wrap(fmt.Errorf("malformed BSON document: %v", r))
		}
	}()
	var mapData map[string]interface{}
	if err := bson.Unmarshal(data, &mapData); err != nil {
		// If error try parse as mongoDump
		if strings.Contains(err.Error(), "invalid document length") {
			var rawData bson.Raw
			mapDataArray := make([]map[string]interface{}, 0)
			for len(data) > 0 {
				var x map[string]interface{}
				if err := bson.Unmarshal(data, &rawData); err != nil {
					return nil, wrap(err)
				}
				if err := bson.Unmarshal(rawData, &x); err != nil {
					return nil, wrap(err)
				}
				mapDataArray = append(mapDataArray, x)
				data = data[len(rawData):]
			}
			return mapDataArray, nil
		}
		return nil, wrap(err)
	}
	return mapData, nil
}

// parseYAML accepts a mapping or a list of mappings at the top level, exactly as before 1.5.0.
func parseYAML(data []byte, source string) (interface{}, error) {
	var mapData map[string]interface{}
	if err := yaml.Unmarshal(data, &mapData); err != nil {
		if strings.Contains(err.Error(), "cannot unmarshal !!") {
			mapDataArray := make([]map[string]interface{}, 0)
			if err := yaml.Unmarshal(data, &mapDataArray); err != nil {
				return nil, yamlError(data, source, err)
			}
			return mapDataArray, nil
		}
		return nil, yamlError(data, source, err)
	}
	return mapData, nil
}

func parseCSV(data []byte, delimiter, source string, warn func(string)) ([]map[string]interface{}, error) {
	var mapData []map[string]interface{}
	r := csv.NewReader(bytes.NewReader(data))
	comma, err := prepareDelimiter(delimiter)
	if err != nil {
		return nil, &bafiError{
			Stage:   stageUsage,
			Summary: fmt.Sprintf("invalid CSV delimiter %q", delimiter),
			Detail:  strings.TrimPrefix(err.Error(), fmt.Sprintf("invalid delimiter %q: ", delimiter)),
			Hint:    "use a single character (-d ';') or a hex byte value (-d 0x09 for TAB)",
			Anchor:  "csv-in-one-column",
			Err:     err,
		}
	}
	r.Comma = comma
	lines, err := r.ReadAll()
	if err != nil {
		return nil, csvError(data, source, comma, err)
	}
	if len(lines) == 0 {
		return nil, emptyInputError(source, "CSV")
	}
	if warn != nil && len(lines[0]) == 1 {
		for _, candidate := range []rune{';', '\t', '|', ','} {
			if candidate != comma && strings.ContainsRune(lines[0][0], candidate) {
				warn(fmt.Sprintf("the CSV header has a single column containing %s - did you mean -d %s?",
					delimiterName(candidate), delimiterFlag(candidate)))
				break
			}
		}
	}
	mapData = make([]map[string]interface{}, len(lines[1:]))
	headers := make([]string, len(lines[0]))
	copy(headers, lines[0])
	for i, line := range lines[1:] {
		x := make(map[string]interface{})
		for j, value := range line {
			x[headers[j]] = value
		}
		mapData[i] = x
	}
	return mapData, nil
}

// delimiterName and delimiterFlag word a delimiter for warnings and hints.
func delimiterName(r rune) string {
	if r == '\t' {
		return "TABs"
	}
	return fmt.Sprintf("'%c'", r)
}

func delimiterFlag(r rune) string {
	if r == '\t' {
		return "0x09"
	}
	return fmt.Sprintf("'%c'", r)
}

// Delimiter can be defined as string or as HEX value eg. 0x09.
// An invalid hex value is returned as an error so the caller can report it like any other input error.
func prepareDelimiter(inputString string) (rune, error) {
	if inputString != "" {
		if len(inputString) == 4 && inputString[0:2] == "0x" {
			bytes, err := hex.DecodeString(inputString[2:4])
			if err != nil {
				return 0, fmt.Errorf("invalid delimiter %q: %s", inputString, err.Error())
			}
			return rune(string(bytes)[0]), nil
		}
		delimiter, _ := utf8.DecodeRuneInString(inputString)
		return delimiter, nil
	}
	return rune(','), nil
}

// cleanBOM remove UTF-8 Byte Order Mark if present
func cleanBOM(b []byte) []byte {
	if len(b) >= 3 &&
		b[0] == 0xef &&
		b[1] == 0xbb &&
		b[2] == 0xbf {
		return b[3:]
	}
	return b
}
