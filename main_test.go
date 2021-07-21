package main

import (
	"fmt"
	"testing"
)

func TestCleanBOM(t *testing.T) {
	fmt.Println("TestCleanBOM")
	input := "\xef\xbb\xbf" + "Hello"
	result := string(cleanBOM([]byte(input)))
	if result != "Hello" {
		t.Errorf("result: %v", result)
	}
}
