package main

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/clbanning/mxj/v2"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v3"
)

const version = "1.0.15"

var (
	luaData *lua.LState
)

type tParams struct {
	inputFile      *string
	outputFile     *string
	textTemplate   *string
	inputFormat    *string
	inputDelimiter *string
	getVersion     *bool
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	if _, err := os.Stat("./lua/functions.lua"); !os.IsNotExist(err) {
		luaData = lua.NewState()
		if err := luaData.DoFile("./lua/functions.lua"); err != nil {
			log.Fatal("loadLuaFunctions", err.Error())
		}
	}
}

func main() {
	params := tParams{
		inputFile: flag.String("i", "", `input file 
 -if not defined read from stdin (pipe mode)
 -if prefixed with "?" app will expect yaml file with multiple files description. `),
		outputFile: flag.String("o", "", `output file, 
 -if not defined write to stdout (pipe mode)`),
		textTemplate: flag.String("t", "", `template, file or inline. 
 -Inline template should start with ? e.g. -t "?{{.MyValue}}" `),
		inputFormat:    flag.String("f", "", "input format: json, bson, yaml, csv, xml(default)"),
		inputDelimiter: flag.String("d", "", "input delimiter: CSV only, default is comma"),
		getVersion:     flag.Bool("v", false, "show version (Project page: https://github.com/mmalcek/bafi)"),
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
	if *params.textTemplate == "" {
		fmt.Println("template file must be defined: -t template.tmpl")
		return nil
	}
	data, files, err := getInputData(params.inputFile)
	if err != nil {
		return err
	}
	// Try identify file format by extension. Input parameter -f has priority
	if *params.inputFormat == "" {
		switch filepath.Ext(*params.inputFile) {
		case ".json":
			*params.inputFormat = "json"
		case ".bson":
			*params.inputFormat = "bson"
		case ".yaml", ".yml":
			*params.inputFormat = "yaml"
		case ".csv":
			*params.inputFormat = "csv"
		case ".xml", ".cdf", ".cdf3":
			*params.inputFormat = "xml"
		default:
			*params.inputFormat = ""
		}
	}

	// If list of file map them one by one else map incoming []byte to mapData
	var mapData interface{}
	if data == nil && files != nil {
		filesStruct := make(map[string]interface{})
		for _, file := range files {
			data, err := ioutil.ReadFile(file["file"].(string))
			if err != nil {
				return err
			}
			*params.inputFormat = file["format"].(string)
			if filesStruct[file["label"].(string)], err = mapInputData(data, params); err != nil {
				return err
			}
		}
		mapData = &filesStruct
	} else {
		if mapData, err = mapInputData(data, params); err != nil {
			return err
		}
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
		if data, err = ioutil.ReadAll(os.Stdin); err != nil {
			return nil, nil, fmt.Errorf("readStdin: %s", err.Error())
		}
	case inputFile[:1] == "?":
		files = make([]map[string]interface{}, 0)
		configFile, err := ioutil.ReadFile(inputFile[1:])
		if err != nil {
			return nil, nil, fmt.Errorf("readFileList: %s", err.Error())
		}
		if err := yaml.Unmarshal(configFile, &files); err != nil {
			return nil, nil, fmt.Errorf("yaml.UnmarshalFileList: %s", err.Error())
		}
		return nil, files, nil
	default:
		if data, err = ioutil.ReadFile(inputFile); err != nil {
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
		var mapData []map[string]interface{}
		r := csv.NewReader(strings.NewReader(string(data)))
		r.Comma = prepareDelimiter(*params.inputDelimiter)
		lines, err := r.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("mapCSV: %s", err.Error())
		}
		mapData = make([]map[string]interface{}, len(lines[1:]))
		headers := make([]string, len(lines[0]))
		for i, header := range lines[0] {
			headers[i] = header
		}
		for i, line := range lines[1:] {
			x := make(map[string]interface{})
			for j, value := range line {
				x[headers[j]] = value
			}
			mapData[i] = x
		}
		return mapData, nil
	case "xml":
		mapData, err := mxj.NewMapXml(data)
		if err != nil {
			return nil, fmt.Errorf("mapXML: %s", err.Error())
		}
		return mapData, nil
	default:
		return nil, fmt.Errorf("unknown input format: use parameter -f to define input format e.g. -f json (accepted values are json, bson, yaml, csv, xml)")
	}
}

// Delimiter can be defined as string or as HEX value eg. 0x09
func prepareDelimiter(inputString string) rune {
	if inputString != "" {
		if len(inputString) == 4 && inputString[0:2] == "0x" {
			bytes, err := hex.DecodeString(inputString[2:4])
			if err != nil {
				log.Fatal(fmt.Sprintf("error CSV delimiter: %s", err.Error()))
			}
			return rune(string(bytes)[0])
		}
		return rune(inputString[0])
	}
	return rune(',')
}

// readTemplate get template from file or from input
func readTemplate(textTemplate string) ([]byte, error) {
	var templateFile []byte
	var err error
	if textTemplate[:1] == "?" {
		templateFile = []byte(textTemplate[1:])
	} else {
		templateFile, err = ioutil.ReadFile(textTemplate)
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
