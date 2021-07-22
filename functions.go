// Inspired by https://github.com/Masterminds/sprig
package main

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/clbanning/mxj/v2"
	"github.com/go-yaml/yaml"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/spf13/cast"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
)

// templateFunctions extends template functions
func templateFunctions() template.FuncMap {
	return template.FuncMap{
		"add":        add,
		"add1":       add1,
		"sub":        sub,
		"div":        div,
		"mod":        mod,
		"mul":        mul,
		"addf":       addf,
		"add1f":      add1f,
		"subf":       subf,
		"divf":       divf,
		"mulf":       mulf,
		"randInt":    randInt,
		"round":      round,
		"max":        max,
		"min":        min,
		"maxf":       maxf,
		"minf":       minf,
		"dateFormat": dateFormat,
		"now":        now,
		"b64enc":     base64encode,
		"b64dec":     base64decode,
		"b32enc":     base32encode,
		"b32dec":     base32decode,
		"uuid":       newUUID,
		"regexMatch": regexMatch,
		"upper":      upper,
		"lower":      lower,
		"trim":       trim,
		"trimAll":    trimAll,
		"trimSuffix": trimSuffix,
		"trimPrefix": trimPrefix,
		"atoi":       atoi,
		"int":        toInt,
		"int64":      toInt64,
		"float64":    toFloat64,
		"toJSON":     toJSON,
		"toBSON":     toBSON,
		"toYAML":     toYAML,
		"toXML":      toXML,
		"mapJSON":    mapJSON,
		"lua":        luaF,
	}
}

// add count
func add(i ...interface{}) int64 {
	var a int64 = 0
	for _, b := range i {
		a += toInt64(b)
	}
	return a
}

// add1 input+1
func add1(i interface{}) int64 { return toInt64(i) + 1 }

// sub substitute
func sub(a, b interface{}) int64 { return toInt64(a) - toInt64(b) }

// div divide
func div(a, b interface{}) int64 { return toInt64(a) / toInt64(b) }

// mod modulo
func mod(a, b interface{}) int64 { return toInt64(a) % toInt64(b) }

// mul multiply
func mul(a interface{}, v ...interface{}) int64 {
	val := toInt64(a)
	for _, b := range v {
		val = val * toInt64(b)
	}
	return val
}

// addf count float
func addf(i ...interface{}) float64 {
	a := interface{}(float64(0))
	return execDecimalOp(a, i, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Add(d2) })
}

// add1f inputFloat+1
func add1f(i interface{}) float64 {
	return execDecimalOp(i, []interface{}{1}, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Add(d2) })
}

// subf substitute float
func subf(a interface{}, v ...interface{}) float64 {
	return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Sub(d2) })
}

// divide float
func divf(a interface{}, v ...interface{}) float64 {
	return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Div(d2) })
}

// mulf multiply float
func mulf(a interface{}, v ...interface{}) float64 {
	return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Mul(d2) })
}

// randInt returns random integer in defined range {{randInt min max}} e.g. {{randInt 1 10}}
func randInt(min, max int) int { return rand.Intn(max-min) + min }

// round float {{round .val 2}} -> 2 decimals or {{round .val 1 0.4}} 0.4 round point
func round(a interface{}, p int, rOpt ...float64) float64 {
	roundOn := .5
	if len(rOpt) > 0 {
		roundOn = rOpt[0]
	}
	val := toFloat64(a)
	places := toFloat64(p)

	var round float64
	pow := math.Pow(10, places)
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	return round / pow
}

// max return highest from numbers {{max .v1 .v2 .v3}}
func max(a interface{}, i ...interface{}) int64 {
	aa := toInt64(a)
	for _, b := range i {
		bb := toInt64(b)
		if bb > aa {
			aa = bb
		}
	}
	return aa
}

// min return lowest from numbers {{min .v1 .v2 .v3}}
func min(a interface{}, i ...interface{}) int64 {
	aa := toInt64(a)
	for _, b := range i {
		bb := toInt64(b)
		if bb < aa {
			aa = bb
		}
	}
	return aa
}

// maxf return highest from float numbers {{maxf .v1 .v2 .v3}}
func maxf(a interface{}, i ...interface{}) float64 {
	aa := toFloat64(a)
	for _, b := range i {
		bb := toFloat64(b)
		aa = math.Max(aa, bb)
	}
	return aa
}

// minf return lowest from float numbers {{minf .v1 .v2 .v3}}
func minf(a interface{}, i ...interface{}) float64 {
	aa := toFloat64(a)
	for _, b := range i {
		bb := toFloat64(b)
		aa = math.Min(aa, bb)
	}
	return aa
}

// dateFormat convert date format {{dateFormat "string", "inputPattern", "outputPattern"}} e.g. {{dateFormat "15.03.2021", "02.01.2006", "01022006"}}
func dateFormat(date string, inputFormat string, outputFormat string) string {
	timeParsed, err := time.Parse(inputFormat, date)
	if err != nil {
		return date
	}
	return timeParsed.Format(outputFormat)
}

// now return current date/time in specified format
func now(format string) string {
	return time.Now().Format(format)
}

// base64encode encode to base64
func base64encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

// base64decode decode from base64
func base64decode(v string) string {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// base32encode encode to base32
func base32encode(v string) string {
	return base32.StdEncoding.EncodeToString([]byte(v))
}

// base32decode decode from base32
func base32decode(v string) string {
	data, err := base32.StdEncoding.DecodeString(v)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// newUUID returns UUID
func newUUID() string { return uuid.New().String() }

// regexMatch check regex e.g. {{regexMatch "a.b", "aaxbb"}}
func regexMatch(regex string, s string) bool {
	match, _ := regexp.MatchString(regex, s)
	return match
}

// upper string to uppercase
func upper(s string) string {
	return strings.ToUpper(s)
}

// lower string to lowercase
func lower(s string) string {
	return strings.ToLower(s)
}

func trim(s string) string {
	return strings.TrimSpace(s)
}

// trimAll remove leading and trailing whitespace
func trimAll(a, b string) string { return strings.Trim(a, b) }

// {{trimPrefix "!Hello World!" "!"}} - returns "Hello World!"
func trimPrefix(a, b string) string { return strings.TrimPrefix(a, b) }

// trimSuffix - {{trimSuffix "!Hello World!" "!"}} - returns "!HelloWorld"
func trimSuffix(a, b string) string { return strings.TrimSuffix(a, b) }

// - atoi {{atoi "42"}} - string to int
func atoi(a string) int { i, _ := strconv.Atoi(a); return i }

// toInt convert to int
func toInt(v interface{}) int {
	return cast.ToInt(v)
}

// toInt64 converts integer types to 64-bit integers
func toInt64(v interface{}) int64 {
	return cast.ToInt64(v)
}

// toFloat64 converts 64-bit floats
func toFloat64(v interface{}) float64 {
	return cast.ToFloat64(v)
}

// toJSON convert to JSON
func toJSON(data map[string]interface{}) string {
	out, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("err: %s", err.Error())
	}
	return string(out)
}

// toBSON convert to BSON
func toBSON(data map[string]interface{}) string {
	out, err := bson.Marshal(data)
	if err != nil {
		return fmt.Sprintf("err: %s", err.Error())
	}
	return string(out)
}

// toYAML convert to YAML
func toYAML(data map[string]interface{}) string {
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Sprintf("err: %s", err.Error())
	}
	return string(out)
}

// toXML convert to XML
func toXML(data map[string]interface{}) string {
	out, err := mxj.AnyXml(data)
	if err != nil {
		return fmt.Sprintf("err: %s", err.Error())
	}
	return string(out)
}

// mapJSON string JSON to map[string]interface{} so it can be used in pipline -> template
func mapJSON(input string) map[string]interface{} {
	var mapData map[string]interface{}
	if err := json.Unmarshal([]byte(input), &mapData); err != nil {
		testData := make(map[string]interface{})
		testData["error"] = err.Error()
		return testData
	}
	return mapData
}

// luaF Call LUA function {{lua "functionName" input1 input2 input3 ...}
// 1. Functions must be placed in ./lua/functions, 2. Inputs are passed as stringified json 3. Output of lua function must be string
func luaF(i ...interface{}) string {
	if luaData == nil {
		return "error: ./lua/functions.lua file missing)"
	}
	strData, err := json.Marshal(i[1:])
	if err != nil {
		return fmt.Sprintf("luaInputError: %s\r\n", err.Error())
	}
	if err := luaData.CallByParam(
		lua.P{Fn: luaData.GetGlobal(i[0].(string)), NRet: 1, Protect: true}, lua.LString(string(strData))); err != nil {
		return fmt.Sprintf("luaError: %s\r\n", err.Error())
	}
	if str, ok := luaData.Get(-1).(lua.LString); ok {
		luaData.Pop(1)
		return str.String()
	}
	return "luaError: getResult"
}

// execDecimalOp convert float to decimal
func execDecimalOp(a interface{}, b []interface{}, f func(d1, d2 decimal.Decimal) decimal.Decimal) float64 {
	prt := decimal.NewFromFloat(toFloat64(a))
	for _, x := range b {
		dx := decimal.NewFromFloat(toFloat64(x))
		prt = f(prt, dx)
	}
	rslt, _ := prt.Float64()
	return rslt
}
