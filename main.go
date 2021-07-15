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

const version = "1.0.1"

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

	var data []byte
	var err error
	if *inputFile == "" {
		if err = checkStdin(); err != nil {
			log.Fatal(err.Error())
		}
		if data, err = ioutil.ReadAll(os.Stdin); err != nil {
			log.Fatal("readStdin: ", err.Error())
		}
	} else {
		data, err = ioutil.ReadFile(*inputFile)
		if err != nil {
			log.Fatal("readFile: ", err.Error())
		}
	}
	data = cleanBOM(data) // Remove UTF-8 Byte Order Mark if present
	var mapData map[string]interface{}
	switch strings.ToLower(*inputFormat) {
	case "json":
		if err := json.Unmarshal(data, &mapData); err != nil {
			log.Fatal("mapJSON: ", err.Error())
		}
	case "bson":
		if err := bson.Unmarshal(data, &mapData); err != nil {
			log.Fatal("mapBSON: ", err.Error())
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &mapData); err != nil {
			log.Fatal("mapYAML: ", err.Error())
		}
	default:
		mapData, err = mxj.NewMapXml(data)
		if err != nil {
			log.Fatal("mapXML: ", err.Error())
		}
	}
	textTemplateVar := *textTemplate
	var templateFile []byte
	if textTemplateVar[:1] == "?" {
		templateFile = []byte(textTemplateVar[1:])
	} else {
		templateFile, err = ioutil.ReadFile(*textTemplate)
		if err != nil {
			log.Fatal("readFile: ", err.Error())
		}
	}
	template, err := template.New("new").Funcs(templateFunctions()).Parse(string(templateFile))
	if err != nil {
		log.Fatal("parseTemplate: ", err.Error())
	}
	if *outputFile == "" {
		output := new(bytes.Buffer)
		err = template.Execute(output, mapData)
		if err != nil {
			log.Fatal("writeStdout: ", err.Error())
		}
		fmt.Print(output)
	} else {
		output, err := os.Create(*outputFile)
		if err != nil {
			log.Fatal("createOutputFile: ", err.Error())
		}
		defer output.Close()
		err = template.Execute(output, mapData)
		if err != nil {
			log.Fatal("writeOutputFile: ", err.Error())
		}
	}
	if luaReady {
		luaData.Close()
	}
}

func cleanBOM(b []byte) []byte {
	if len(b) >= 3 &&
		b[0] == 0xef &&
		b[1] == 0xbb &&
		b[2] == 0xbf {
		return b[3:]
	}
	return b
}

func checkStdin() error {
	if fi, err := os.Stdin.Stat(); err != nil {
		return fmt.Errorf("getStdin: %s", err.Error())
	} else {
		if fi.Mode()&os.ModeNamedPipe == 0 {
			return fmt.Errorf("stdin: Error-noPipe")
		}
	}
	return nil
}