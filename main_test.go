package main

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/clbanning/mxj/v2"
)

const bsonDump = `JgAAAAdfaWQAYQAVOXB0lMEH42tuAm5hbWUABgAAAEhlbGxvAAAmAAAAB19pZABhCmAZAkO5o4cVWbsCbmFtZQAGAAAAV29ybGQAAA==`

func TestProcessTemplate(t *testing.T) {
	inputFile := ""
	inputFormat := ""
	inputDelimiter := ","
	outputFile := ""
	textTemplate := `?{{define content}}`
	getVersion := false
	getHelp := false
	chatGPTkey := ""
	chatGPTmodel := ""
	chatGPTquery := ""

	params := tParams{
		inputFile:      &inputFile,
		inputFormat:    &inputFormat,
		inputDelimiter: &inputDelimiter,
		outputFile:     &outputFile,
		textTemplate:   &textTemplate,
		getVersion:     &getVersion,
		getHelp:        &getHelp,
		chatGPTkey:     &chatGPTkey,
		chatGPTmodel:   &chatGPTmodel,
		chatGPTquery:   &chatGPTquery,
	}
	err := processTemplate(params)
	if !strings.Contains(err.Error(), "stdin: Error-noPipe") {
		t.Errorf("result: %v", err.Error())
	}
	inputFile = "testdata.xml"
	textTemplate = "hello.tmpl"
	err = processTemplate(params)
	if !strings.Contains(err.Error(), `readFile: open hello.tmpl:`) {
		t.Errorf("result: %v", err.Error())
	}
	inputFile = "testdata.xml"
	textTemplate = "?{{define content}}"
	err = processTemplate(params)
	if !strings.Contains(err.Error(), `unexpected "content" in define`) {
		t.Errorf("result: %v", err.Error())
	}
	inputFormat = "json"
	err = processTemplate(params)
	if !strings.Contains(err.Error(), `mapJSON: invalid character`) {
		t.Errorf("result: %v", err.Error())
	}
	textTemplate = `?Output template test: description = {{index .TOP_LEVEL "-description"}} {{print "\r\n"}}`
	inputFormat = ""
	err = processTemplate(params)
	if err != nil {
		t.Errorf("result: %v", err.Error())
	}

	textTemplate = `?Output template test: description = {{index .filesTest.TOP_LEVEL "-description"}} {{print "\r\n"}}`
	inputFile = "?filesTest.yaml"
	err = processTemplate(params)
	if err != nil {
		t.Errorf("result: %v", err.Error())
	}

	inputFile = "?filesTest.yamlx"
	err = processTemplate(params)
	if !strings.Contains(err.Error(), "readFileList: open filesTest.yamlx") {
		t.Errorf("result: %v", err.Error())
	}

	getVersion = true
	err = processTemplate(params)
	if err != nil {
		t.Errorf("result: %v", err.Error())
	}
	textTemplate = ""
	getVersion = false
	err = processTemplate(params)
	if err != nil {
		t.Errorf("result: %v", err.Error())
	}
}

func TestGetInputData(t *testing.T) {
	inputFile := "testdata.xml"
	data, _, _ := getInputData(&inputFile)
	if !strings.Contains(string(data), `<?xml version="1.0" encoding="utf-8"?>`) {
		t.Errorf("result: %v", string(data))
	}
	inputFile = "Hello.xml"
	_, _, err := getInputData(&inputFile)
	if !strings.Contains(err.Error(), `readFile: open Hello.xml:`) {
		t.Errorf("result: %v", err.Error())
	}
	inputFile = ""
	_, _, err = getInputData(&inputFile)
	if !strings.Contains(err.Error(), `stdin: Error-noPipe`) {
		t.Errorf("result: %v", err.Error())
	}
}

func TestMapInputData(t *testing.T) {
	inputFile := ""
	inputFormat := ""
	inputDelimiter := ","
	outputFile := ""
	textTemplate := `?{{define content}}`
	getVersion := false
	params := tParams{
		inputFile:      &inputFile,
		inputFormat:    &inputFormat,
		inputDelimiter: &inputDelimiter,
		outputFile:     &outputFile,
		textTemplate:   &textTemplate,
		getVersion:     &getVersion,
	}
	// Test map json
	input := []byte(`{"name": "John","age": 30}`)
	inputFormat = "json"
	result, _ := mapInputData(input, params)
	if result.(map[string]interface{})["name"] != "John" || result.(map[string]interface{})["age"] != float64(30) {
		t.Errorf("resultJSON: %v", result)
	}
	input = []byte(`[{"name": "John","age": 30}, {"name": "Hanz","age": 28}]`)
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != "John" {
		t.Errorf("resultJSONarray: %v", result.([]map[string]interface{})[0]["name"])
	}
	input = []byte(`[{"name": "John","age": 30}, {"name": Hanz","age": 28}]`)
	_, err := mapInputData(input, params)
	if !strings.Contains(err.Error(), "invalid character 'H'") {
		t.Errorf("resultJSONerr: %v", err.Error())
	}
	input = []byte(`{"name" John","age": 30}`)
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "invalid character 'J'") {
		t.Errorf("resultJSONerr: %v", err.Error())
	}
	// Test map bson
	input = []byte{14, 0, 0, 0, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0}
	inputFormat = "bson"
	result, _ = mapInputData(input, params)
	if result.(map[string]interface{})["h"] != "w" {
		t.Errorf("resultBSON: %v", result)
	}
	input, err = base64.StdEncoding.DecodeString(bsonDump)
	if err != nil {
		t.Errorf("base64BSONdump: %v", err.Error())
	}
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != `Hello` {
		t.Errorf("resultBSONdump: %v", result.([]map[string]interface{})[0]["name"])
	}
	input = []byte{14, 0, 0, 1, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0}
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "EOF") {
		t.Errorf("resultBSONerr: %v", err.Error())
	}
	// Test map yaml
	input = []byte(`name: John`)
	inputFormat = "yaml"
	result, _ = mapInputData(input, params)
	if result.(map[string]interface{})["name"] != "John" {
		t.Errorf("resultYAML: %v", result)
	}
	input = []byte("- name: John\r\n- name: Peter")
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != "John" {
		t.Errorf("resultYAMLarray: %v", result)
	}
	input = []byte("- name John\r\n- name: Peter")
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "cannot unmarshal !!str `name John`") {
		t.Errorf("resultYAMLarrayErr: %v", err.Error())
	}
	input = []byte(`name John`)
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "cannot unmarshal !!str `name John`") {
		t.Errorf("resultYAMLerr: %v", err.Error())
	}
	// Test map csv
	input = []byte("name,surname\r\nHello,World")
	inputFormat = "csv"
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != "Hello" {
		t.Errorf("result: %v", result.([]map[string]interface{})[0]["name"])
	}
	input = []byte("name,surname\r\nHello,World")
	inputDelimiter = "0x2C"
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != "Hello" {
		t.Errorf("result: %v", result.([]map[string]interface{})[0]["name"])
	}
	input = []byte("name,surname\r\nHello,World,!!!")
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "wrong number of fields") {
		t.Errorf("result: %v", err.Error())
	}
	input = []byte("")
	_, err = mapInputData(input, params)
	if err == nil || !strings.Contains(err.Error(), "CSV has no rows") {
		t.Errorf("result: %v", err)
	}

	// Test map xml
	input = []byte(`<name>John</name>`)
	inputFormat = "xml"
	result, _ = mapInputData(input, params)
	if result.(mxj.Map)["name"] != "John" {
		t.Errorf("result: %v", result)
	}
	input = []byte(`<name>John<name>`)
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "xml.Decoder.Token() - XML syntax error on line 1") {
		t.Errorf("result: %v", err.Error())
	}
}

func TestReadTemplate(t *testing.T) {
	// Test inline template
	result, err := readTemplate("?{{toXML .}}")
	if string(result) != "{{toXML .}}" {
		t.Errorf("result: %v", string(result))
	}
	if err != nil {
		t.Errorf("err: %v", err)
	}
	// Test template file
	result, err = readTemplate("template.tmpl")
	if !strings.Contains(string(result), "CSV formatted data:") {
		t.Errorf("result: %v", string(result))
	}
	if err != nil {
		t.Errorf("err: %v", err)
	}
	// Test nonExisting file
	_, err = readTemplate("hello.tmpl")
	if !strings.Contains(err.Error(), "readFile: open hello.tmpl:") {
		t.Errorf("err: %v", err)
	}
}

func TestWriteOutputData(t *testing.T) {
	testData := make(map[string]interface{})
	testData["Hello"] = "World"
	outputFile := ""
	templateFile := []byte(`{{define content}}`)
	err := writeOutputData(testData, &outputFile, templateFile)
	if !strings.Contains(err.Error(), `new:1: unexpected "content"`) {
		t.Errorf("result: %v", err.Error())
	}
	templateFile = []byte(`Output test: Hello {{.Hello}} {{print "\r\n"}}`)
	if err := writeOutputData(testData, &outputFile, templateFile); err != nil {
		t.Errorf("result: %v", err.Error())
	}
	outputFile = "output.txt"
	if err := writeOutputData(testData, &outputFile, templateFile); err != nil {
		t.Errorf("result: %v", err.Error())
	}
	testData["Hello"] = make(chan int, 1)
	err = writeOutputData(testData, &outputFile, templateFile)
	if !strings.Contains(err.Error(), "can't print {{.Hello}} of type chan int") {
		t.Errorf("result: %v", err.Error())
	}
	outputFile = "out*he\\ll//o/./txt"
	err = writeOutputData(testData, &outputFile, templateFile)
	if !strings.Contains(err.Error(), "createOutputFile:") {
		t.Errorf("result: %v", err.Error())
	}
}

func TestCleanBOM(t *testing.T) {
	input := "\xef\xbb\xbf" + "Hello"
	result := string(cleanBOM([]byte(input)))
	if result != "Hello" {
		t.Errorf("result: %v", result)
	}
	input = "Hello"
	result = string(cleanBOM([]byte(input)))
	if result != "Hello" {
		t.Errorf("result: %v", result)
	}
}
