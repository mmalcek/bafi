package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/clbanning/mxj/v2"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v2"
)

const version = "1.0.5"

var (
	luaData *lua.LState
)

type tParams struct {
	inputFile    *string
	outputFile   *string
	textTemplate *string
	inputFormat  *string
	getVersion   *bool
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
		inputFile:    flag.String("i", "", "input file, if not defined read from stdin (pipe mode)"),
		outputFile:   flag.String("o", "", "output file, if not defined write to stdout (pipe mode)"),
		textTemplate: flag.String("t", "", "template, file or inline. Inline should start with ? e.g. -t \"?{{.MyValue}}\" "),
		inputFormat:  flag.String("f", "", "input format, json, bson, yaml, csv, xml(default)"),
		getVersion:   flag.Bool("v", false, "print version and exit"),
	}
	flag.Parse()

	if err := processTemplate(params); err != nil {
		log.Fatal(err.Error())
	}
}

func processTemplate(params tParams) error {
	if *params.getVersion {
		fmt.Printf("Version: %s\r\n", version)
		return nil
	}
	if *params.textTemplate == "" {
		fmt.Println("template file must be defined: -t template.tmpl")
		return nil
	}
	data, err := getInputData(params.inputFile)
	if err != nil {
		return err
	}
	mapData, err := mapInputData(data, params.inputFormat)
	if err != nil {
		return err
	}
	templateFile, err := readTemplate(*params.textTemplate)
	if err != nil {
		return err
	}
	if err := writeOutputData(mapData, params.outputFile, templateFile); err != nil {
		return err
	}
	if luaData != nil {
		luaData.Close()
	}
	return nil
}

// getInputData get the data from stdin/pipe or from file
func getInputData(inputFile *string) ([]byte, error) {
	var data []byte
	var err error
	if *inputFile == "" {
		fi, err := os.Stdin.Stat()
		if err != nil {
			return nil, fmt.Errorf("getStdin: %s", err.Error())
		}
		if fi.Mode()&os.ModeNamedPipe == 0 {
			return nil, fmt.Errorf("stdin: Error-noPipe")
		}
		if data, err = ioutil.ReadAll(os.Stdin); err != nil {
			return nil, fmt.Errorf("readStdin: %s", err.Error())
		}
	} else {
		if data, err = ioutil.ReadFile(*inputFile); err != nil {
			return nil, fmt.Errorf("readFile: %s", err.Error())
		}
	}
	return cleanBOM(data), nil
}

// mapInputData map input data to map[string]interface{}
func mapInputData(data []byte, inputFormat *string) (map[string]interface{}, error) {
	var err error
	var mapData map[string]interface{}
	switch strings.ToLower(*inputFormat) {
	case "json":
		if err := json.Unmarshal(data, &mapData); err != nil {
			if strings.Contains(err.Error(), "cannot unmarshal array") {
				x := make([]map[string]interface{}, 0)
				if err := json.Unmarshal(data, &x); err != nil {
					return nil, fmt.Errorf("jsonArray: %s", err.Error())
				}
				mapData = make(map[string]interface{})
				for i := range x {
					mapData[strconv.Itoa(i)] = x[i]
				}
				return mapData, nil
			}
			return nil, fmt.Errorf("mapJSON: %s", err.Error())
		}
	case "bson":
		if err := bson.Unmarshal(data, &mapData); err != nil {
			// If error try parse as mongoDump
			if strings.Contains(err.Error(), "invalid document length") {
				var rawData bson.Raw
				mapData = make(map[string]interface{})
				i := 0
				for len(data) > 0 {
					var x map[string]interface{}
					if err := bson.Unmarshal(data, &rawData); err != nil {
						return nil, fmt.Errorf("mapBSONArray1: %s", err.Error())
					}
					if err := bson.Unmarshal(rawData, &x); err != nil {
						return nil, fmt.Errorf("mapBSONArray2: %s", err.Error())
					}
					mapData[strconv.Itoa(i)] = x
					data = data[len(rawData):]
					i++
				}
				return mapData, nil
			}
			return nil, fmt.Errorf("mapBSON: %s", err.Error())
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &mapData); err != nil {
			return nil, fmt.Errorf("mapYAML: %s", err.Error())
		}
	case "csv":
		r := csv.NewReader(strings.NewReader(string(data)))
		r.Comma = ','
		lines, err := r.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("mapCSV: %s", err.Error())
		}
		mapData = make(map[string]interface{})
		headers := make([]string, len(lines[0]))
		for i, header := range lines[0] {
			headers[i] = header
		}
		for i, line := range lines[1:] {
			x := make(map[string]interface{})
			for j, value := range line {
				x[headers[j]] = value
			}
			mapData[strconv.Itoa(i)] = x
		}
	default:
		mapData, err = mxj.NewMapXml(data)
		if err != nil {
			return nil, fmt.Errorf("mapXML: %s", err.Error())
		}
	}
	return mapData, nil
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
func writeOutputData(mapData map[string]interface{}, outputFile *string, templateFile []byte) error {
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
