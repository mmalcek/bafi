package main

import (
	"strings"
	"testing"
)

func TestProcessTemplate(t *testing.T) {
	inputFile := ""
	inputFormat := ""
	outputFile := ""
	textTemplate := `?{{define content}}`
	getVersion := false
	params := tParams{
		inputFile:    &inputFile,
		inputFormat:  &inputFormat,
		outputFile:   &outputFile,
		textTemplate: &textTemplate,
		getVersion:   &getVersion,
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
	data, _ := getInputData(&inputFile)
	if !strings.Contains(string(data), `<?xml version="1.0" encoding="utf-8"?>`) {
		t.Errorf("result: %v", string(data))
	}
	inputFile = "Hello.xml"
	_, err := getInputData(&inputFile)
	if !strings.Contains(err.Error(), `readFile: open Hello.xml:`) {
		t.Errorf("result: %v", err.Error())
	}
	inputFile = ""
	_, err = getInputData(&inputFile)
	if !strings.Contains(err.Error(), `stdin: Error-noPipe`) {
		t.Errorf("result: %v", err.Error())
	}
}

func TestMapInputData(t *testing.T) {
	// Test map json
	input := []byte(`{"name": "John","age": 30}`)
	format := "json"
	result, _ := mapInputData(input, &format)
	if result["name"] != "John" || result["age"] != float64(30) {
		t.Errorf("result: %v", result)
	}
	input = []byte(`{"name" John","age": 30}`)
	_, err := mapInputData(input, &format)
	if !strings.Contains(err.Error(), "invalid character 'J'") {
		t.Errorf("result: %v", err.Error())
	}
	// Test map bson
	input = []byte{14, 0, 0, 0, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0}
	format = "bson"
	result, _ = mapInputData(input, &format)
	if result["h"] != "w" {
		t.Errorf("result: %v", result)
	}
	input = []byte{14, 0, 0, 1, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0}
	_, err = mapInputData(input, &format)
	if !strings.Contains(err.Error(), "mapBSON: invalid document length") {
		t.Errorf("result: %v", err.Error())
	}
	// Test map yaml
	input = []byte(`name: John`)
	format = "yaml"
	result, _ = mapInputData(input, &format)
	if result["name"] != "John" {
		t.Errorf("result: %v", result)
	}
	input = []byte(`name John`)
	_, err = mapInputData(input, &format)
	if !strings.Contains(err.Error(), "cannot unmarshal !!str `name John`") {
		t.Errorf("result: %v", err.Error())
	}
	// Test map xml
	input = []byte(`<name>John</name>`)
	format = "xml"
	result, _ = mapInputData(input, &format)
	if result["name"] != "John" {
		t.Errorf("result: %v", result)
	}
	input = []byte(`<name>John<name>`)
	_, err = mapInputData(input, &format)
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

/*
func TestCheckFlags(t *testing.T) {
	getVersion := false
	textTemplate := ""
	if checkFlags(&getVersion, &textTemplate) == "" {
		t.Errorf("result: %v", checkFlags(&getVersion, &textTemplate))
	}
	getVersion = true
	if checkFlags(&getVersion, &textTemplate) == "" {
		t.Errorf("result: %v", checkFlags(&getVersion, &textTemplate))
	}
	getVersion = false
	textTemplate = "template.tmpl"
	if checkFlags(&getVersion, &textTemplate) != "" {
		t.Errorf("result: %v", checkFlags(&getVersion, &textTemplate))
	}
}
*/
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
