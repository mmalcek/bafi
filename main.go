package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/alpkeskin/gotoon"
	"github.com/clbanning/mxj/v2"
	"github.com/mmalcek/mt940"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v3"
)

//go:embed version.txt
var versionFile string

// version is the release version, trimmed from version.txt (single source of truth, also read by Taskfile.yml).
var version = strings.TrimSpace(versionFile)

type tParams struct {
	inputFile      *string
	outputFile     *string
	textTemplate   *string
	inputFormat    *string
	inputDelimiter *string
	getVersion     *bool
	getHelp        *bool
	chatGPTkey     *string
	chatGPTmodel   *string
	chatGPTquery   *string
}

// openAIKeyEnv is the environment variable consulted for the OpenAI API key when -gk is not given.
// Preferred over -gk: command-line arguments are visible in process listings and shell history.
const openAIKeyEnv = "OPENAI_API_KEY"

func main() {
	params := tParams{
		inputFile: flag.String("i", "", `input file 
 -if not defined read from stdin (pipe mode)
 -if prefixed with "?" app will expect yaml file with multiple files description. `),
		outputFile: flag.String("o", "", `output file, 
 -if not defined write to stdout (pipe mode)`),
		textTemplate: flag.String("t", "", `template, file or inline. 
 -Inline template should start with ? e.g. -t "?{{.MyValue}}" `),
		inputFormat:    flag.String("f", "", "input format: json, bson, yaml, csv, mt940, xml(default)"),
		inputDelimiter: flag.String("d", "", "input delimiter: CSV only, default is comma -d ';' or -d 0x09"),
		getVersion:     flag.Bool("v", false, "show version (Project page: https://github.com/mmalcek/bafi)"),
		getHelp:        flag.Bool("h", false, "show help"),
		chatGPTkey:     flag.String("gk", "", "OpenAI API key (prefer the "+openAIKeyEnv+" environment variable; -gk overrides it)"),
		chatGPTmodel:   flag.String("gm", "gpt35", "OpenAI GPT-3 model (gpt35, gpt4)"),
		chatGPTquery:   flag.String("gq", "", "OpenAI query"),
	}
	flag.Parse()

	if err := processTemplate(params); err != nil {
		log.Fatal(err.Error())
	}
	if luaData != nil {
		luaData.Close()
	}
}

func processTemplate(params tParams) error {
	if *params.getVersion {
		fmt.Printf("Version: %s\r\nProject page: https://github.com/mmalcek/bafi\r\n", version)
		return nil
	}
	if *params.getHelp {
		fmt.Println("Usage: bafi -i input.json -t template.tmpl -o output.txt")
		flag.PrintDefaults()
		return nil
	}
	// OpenAI mode is selected by either OpenAI flag; the key itself may also come from the environment.
	openAIMode := *params.chatGPTkey != "" || *params.chatGPTquery != ""
	if *params.textTemplate == "" && !openAIMode {
		fmt.Println("template file must be defined: -t template.tmpl")
		return nil
	}
	data, files, err := getInputData(params.inputFile)
	if err != nil {
		return err
	}
	// Try identify file format by extension. Input parameter -f has priority
	if *params.inputFormat == "" {
		*params.inputFormat = detectFormat(*params.inputFile)
	}

	// If list of file map them one by one else map incoming []byte to mapData
	var mapData interface{}
	if data == nil && files != nil {
		filesStruct := make(map[string]interface{})
		for i, file := range files {
			entry, err := parseFileListEntry(i, file)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(entry.file)
			if err != nil {
				return err
			}
			*params.inputFormat = entry.format
			if filesStruct[entry.label], err = mapInputData(data, params); err != nil {
				return err
			}
		}
		mapData = &filesStruct
	} else {
		if mapData, err = mapInputData(data, params); err != nil {
			return err
		}
	}

	if openAIMode {
		if *params.chatGPTquery == "" {
			fmt.Println("OpenAI query must be defined: -gq \"What is the weather like?\"")
			return nil
		}
		apiKey := *params.chatGPTkey
		if apiKey == "" {
			apiKey = os.Getenv(openAIKeyEnv)
		}
		if apiKey == "" {
			return fmt.Errorf("OpenAI API key must be defined: -gk <key> or environment variable %s", openAIKeyEnv)
		}
		response, err := chatGPTprocess(mapData, params, apiKey)
		if err != nil {
			return err
		}
		if len(response.Choices) == 0 {
			return fmt.Errorf("OpenAI: empty response (no choices returned)")
		}
		if *params.outputFile == "" {
			fmt.Println(response.Choices[0].Message.Content)
		} else {
			output, err := os.Create(*params.outputFile)
			if err != nil {
				return fmt.Errorf("createOutputFile: %s", err.Error())
			}
			defer output.Close()
			output.WriteString(response.Choices[0].Message.Content)
		}
		return nil
	}

	templateFile, err := readTemplate(*params.textTemplate)
	if err != nil {
		return err
	}
	if err := writeOutputData(mapData, params.outputFile, templateFile); err != nil {
		return err
	}
	return nil
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

// fileListEntry is one validated record of the "-i ?files.yaml" file list.
type fileListEntry struct {
	file   string
	format string
	label  string
}

// parseFileListEntry validates one record of the file list. "file" and "label" are required strings;
// "format" is optional and falls back to the file extension (an unknown extension is reported later by
// mapInputData, same as for a single input file). A malformed record yields an error naming the record
// index and key instead of a panic.
func parseFileListEntry(index int, entry map[string]interface{}) (fileListEntry, error) {
	getString := func(key string, required bool) (string, error) {
		raw, ok := entry[key]
		if !ok || raw == nil {
			if required {
				return "", fmt.Errorf("fileList[%d]: missing %q", index, key)
			}
			return "", nil
		}
		value, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("fileList[%d]: %q must be a string (got %T)", index, key, raw)
		}
		if required && value == "" {
			return "", fmt.Errorf("fileList[%d]: %q must not be empty", index, key)
		}
		return value, nil
	}
	var result fileListEntry
	var err error
	if result.file, err = getString("file", true); err != nil {
		return result, err
	}
	if result.label, err = getString("label", true); err != nil {
		return result, err
	}
	if result.format, err = getString("format", false); err != nil {
		return result, err
	}
	if result.format == "" {
		result.format = detectFormat(result.file)
	}
	return result, nil
}

// getInputData get the data from stdin/pipe or from file or forward list of multiple input files
func getInputData(input *string) (data []byte, files []map[string]interface{}, errorMsg error) {
	var err error
	inputFile := *input
	switch {
	case inputFile == "":
		fi, err := os.Stdin.Stat()
		if err != nil {
			return nil, nil, fmt.Errorf("getStdin: %s", err.Error())
		}
		if fi.Mode()&os.ModeNamedPipe == 0 {
			return nil, nil, fmt.Errorf("stdin: Error-noPipe")
		}
		if data, err = io.ReadAll(os.Stdin); err != nil {
			return nil, nil, fmt.Errorf("readStdin: %s", err.Error())
		}
	case inputFile[:1] == "?":
		files = make([]map[string]interface{}, 0)
		configFile, err := os.ReadFile(inputFile[1:])
		if err != nil {
			return nil, nil, fmt.Errorf("readFileList: %s", err.Error())
		}
		if err := yaml.Unmarshal(configFile, &files); err != nil {
			return nil, nil, fmt.Errorf("yaml.UnmarshalFileList: %s", err.Error())
		}
		return nil, files, nil
	default:
		if data, err = os.ReadFile(inputFile); err != nil {
			return nil, nil, fmt.Errorf("readFile: %s", err.Error())
		}
	}
	return cleanBOM(data), nil, nil
}

// mapInputData map input data to map[string]interface{}
func mapInputData(data []byte, params tParams) (interface{}, error) {
	switch strings.ToLower(*params.inputFormat) {
	case "json":
		var mapData map[string]interface{}
		if err := json.Unmarshal(data, &mapData); err != nil {
			if strings.Contains(err.Error(), "cannot unmarshal array") {
				mapDataArray := make([]map[string]interface{}, 0)
				if err := json.Unmarshal(data, &mapDataArray); err != nil {
					return nil, fmt.Errorf("jsonArray: %s", err.Error())
				}
				return mapDataArray, nil
			}
			return nil, fmt.Errorf("mapJSON: %s", err.Error())
		}
		return mapData, nil
	case "bson":
		var mapData map[string]interface{}
		if err := bson.Unmarshal(data, &mapData); err != nil {
			// If error try parse as mongoDump
			if strings.Contains(err.Error(), "invalid document length") {
				var rawData bson.Raw
				mapDataArray := make([]map[string]interface{}, 0)
				i := 0
				for len(data) > 0 {
					var x map[string]interface{}
					if err := bson.Unmarshal(data, &rawData); err != nil {
						return nil, fmt.Errorf("mapBSONArray1: %s", err.Error())
					}
					if err := bson.Unmarshal(rawData, &x); err != nil {
						return nil, fmt.Errorf("mapBSONArray2: %s", err.Error())
					}
					mapDataArray = append(mapDataArray, x)
					data = data[len(rawData):]
					i++
				}
				return mapDataArray, nil
			}
			return nil, fmt.Errorf("mapBSON: %s", err.Error())
		}
		return mapData, nil
	case "yaml":
		var mapData map[string]interface{}
		if err := yaml.Unmarshal(data, &mapData); err != nil {
			if strings.Contains(err.Error(), "cannot unmarshal !!") {
				mapDataArray := make([]map[string]interface{}, 0)
				if err := yaml.Unmarshal(data, &mapDataArray); err != nil {
					return nil, fmt.Errorf("yamlArray: %s", err.Error())
				}
				return mapDataArray, nil
			}
			return nil, fmt.Errorf("mapYAML: %s", err.Error())
		}
		return mapData, nil
	case "csv":
		return parseCSV(data, *params.inputDelimiter)
	case "xml":
		mapData, err := mxj.NewMapXml(data)
		if err != nil {
			return nil, fmt.Errorf("mapXML: %s", err.Error())
		}
		return mapData, nil
	case "mt940":
		if *params.inputDelimiter == "" {
			return mt940.Parse(data)
		} else {
			*params.inputDelimiter = strings.Replace(*params.inputDelimiter, `\r`, "\r", -1)
			*params.inputDelimiter = strings.Replace(*params.inputDelimiter, `\n`, "\n", -1)
			return mt940.ParseMultimessage(data, *params.inputDelimiter)
		}
	default:
		return nil, fmt.Errorf("unknown input format: use parameter -f to define input format e.g. -f json (accepted values are json, bson, yaml, csv, mt940, xml)")
	}
}

func parseCSV(data []byte, delimiter string) ([]map[string]interface{}, error) {
	var mapData []map[string]interface{}
	r := csv.NewReader(bytes.NewReader(data))
	comma, err := prepareDelimiter(delimiter)
	if err != nil {
		return nil, fmt.Errorf("mapCSV: %s", err.Error())
	}
	r.Comma = comma
	lines, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("mapCSV: %s", err.Error())
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("mapCSV: CSV has no rows")
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

// readTemplate get template from file or from input
func readTemplate(textTemplate string) ([]byte, error) {
	var templateFile []byte
	var err error
	if textTemplate == "" {
		return nil, fmt.Errorf("readTemplate: template must be defined: -t template.tmpl")
	}
	if textTemplate[:1] == "?" {
		templateFile = []byte(textTemplate[1:])
	} else {
		templateFile, err = os.ReadFile(textTemplate)
		if err != nil {
			return nil, fmt.Errorf("readFile: %s", err.Error())
		}
	}
	return templateFile, nil
}

// writeOutputData process template and write output
func writeOutputData(mapData interface{}, outputFile *string, templateFile []byte) error {
	var err error
	template, err := template.New("new").Funcs(templateFunctions()).Parse(string(templateFile))
	if err != nil {
		return fmt.Errorf("parseTemplate: %s", err.Error())
	}
	if *outputFile == "" {
		output := new(bytes.Buffer)
		if err = template.Execute(output, mapData); err != nil {
			return fmt.Errorf("writeStdout: %s", err.Error())
		}
		fmt.Print(output)
	} else {
		output, err := os.Create(*outputFile)
		if err != nil {
			return fmt.Errorf("createOutputFile: %s", err.Error())
		}
		defer output.Close()
		if err = template.Execute(output, mapData); err != nil {
			return fmt.Errorf("writeOutputFile: %s", err.Error())
		}
	}
	return nil
}

// chatGPTprocess sends the query and the whole parsed input (TOON encoded) to OpenAI.
// apiKey is already resolved by the caller (-gk flag or OPENAI_API_KEY environment variable).
func chatGPTprocess(mapData interface{}, params tParams, apiKey string) (response openai.ChatCompletionResponse, err error) {
	jsonData, err := gotoon.Encode(mapData)
	if err != nil {
		return response, fmt.Errorf("jsonMarshal: %s", err.Error())
	}
	model := openai.GPT3Dot5Turbo
	switch *params.chatGPTmodel {
	case "gpt35":
		model = openai.GPT3Dot5Turbo
	case "gpt4":
		model = openai.GPT4
	case "gpt4o":
		model = openai.GPT4o
	case "gpt4o-mini":
		model = openai.GPT4oMini
	case "gpt5":
		model = openai.GPT5
	case "gpt5mini":
		model = openai.GPT5Mini
	case "gpt5nano":
		model = openai.GPT5Nano
	default:
		model = openai.GPT3Dot5Turbo
	}

	client := openai.NewClient(apiKey)
	return client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: *params.chatGPTquery + "\n" + string(jsonData),
				},
			},
		},
	)
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
