package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/clbanning/mxj/v2"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/yaml.v2"
)

const version = "1.0.3"

var (
	luaData  *lua.LState
	luaReady = false
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
	if _, err := os.Stat("./lua/functions.lua"); !os.IsNotExist(err) {
		luaData = lua.NewState()
		err := luaData.DoFile("./lua/functions.lua")
		if err != nil {
			log.Fatal("loadLuaFunctions", err.Error())
		}
		luaReady = true
	}
}

func main() {
	inputFile := flag.String("i", "", "input file, if not defined read from stdin (pipe mode)")
	outputFile := flag.String("o", "", "output file, if not defined stdout is used")
	textTemplate := flag.String("t", "", "template, file or inline. Inline should start with ? e.g. -t \"?{{.MyValue}}\" ")
	inputFormat := flag.String("f", "", "input format (json,xml), default xml")
	getVersion := flag.Bool("v", false, "template")
	flag.Parse()

	if *getVersion {
		fmt.Printf("Version: %s\r\n", version)
		os.Exit(0)
	}

	if *textTemplate == "" {
		log.Fatal("template file must be defined: -t template.tmpl")
	}

	if err := processTemplate(inputFile, inputFormat, textTemplate, outputFile); err != nil {
		log.Fatal(err.Error())
	}

	if luaReady {
		luaData.Close()
	}
}

func processTemplate(inputFile *string, inputFormat *string, textTemplate *string, outputFile *string) error {
	data, err := getInputData(inputFile)
	if err != nil {
		return err
	}
	mapData, err := mapInputData(data, inputFormat)
	if err != nil {
		return err
	}
	templateFile, err := readTemplate(*textTemplate)
	if err != nil {
		return err
	}
	if err := writeOutputData(mapData, outputFile, templateFile); err != nil {
		return err
	}
	return nil
}

func writeOutputData(mapData map[string]interface{}, outputFile *string, templateFile []byte) error {
	var err error
	template, err := template.New("new").Funcs(templateFunctions()).Parse(string(templateFile))
	if err != nil {
		return fmt.Errorf("parseTemplate: %s", err.Error())
	}
	if *outputFile == "" {
		output := new(bytes.Buffer)
		err = template.Execute(output, mapData)
		if err != nil {
			return fmt.Errorf("writeStdout: %s", err.Error())
		}
		fmt.Print(output)
	} else {
		output, err := os.Create(*outputFile)
		if err != nil {
			return fmt.Errorf("createOutputFile: %s", err.Error())
		}
		defer output.Close()
		err = template.Execute(output, mapData)
		if err != nil {
			return fmt.Errorf("writeOutputFile: %s", err.Error())
		}
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
			return nil, fmt.Errorf("mapJSON: %s", err.Error())
		}
	case "bson":
		if err := bson.Unmarshal(data, &mapData); err != nil {
			return nil, fmt.Errorf("mapBSON: %s", err.Error())
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &mapData); err != nil {
			return nil, fmt.Errorf("mapYAML: %s", err.Error())
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
