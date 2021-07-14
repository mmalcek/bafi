// Inspired by https://github.com/Masterminds/sprig
package main

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
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

func templateFunctions() template.FuncMap {
	return template.FuncMap{
		"lua":     luaF,
		"add":     add,
		"add1":    func(i interface{}) int64 { return toInt64(i) + 1 },
		"sub":     func(a, b interface{}) int64 { return toInt64(a) - toInt64(b) },
		"div":     func(a, b interface{}) int64 { return toInt64(a) / toInt64(b) },
		"mod":     func(a, b interface{}) int64 { return toInt64(a) % toInt64(b) },
		"mul":     mul,
		"randInt": func(min, max int) int { return rand.Intn(max-min) + min },
		"add1f": func(i interface{}) float64 {
			return execDecimalOp(i, []interface{}{1}, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Add(d2) })
		},
		"addf": func(i ...interface{}) float64 {
			a := interface{}(float64(0))
			return execDecimalOp(a, i, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Add(d2) })
		},
		"subf": func(a interface{}, v ...interface{}) float64 {
			return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Sub(d2) })
		},
		"divf": func(a interface{}, v ...interface{}) float64 {
			return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Div(d2) })
		},
		"mulf": func(a interface{}, v ...interface{}) float64 {
			return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Mul(d2) })
		},
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
		"uuid":       func() string { return uuid.New().String() },
		"regexMatch": regexMatch,
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"trim":       strings.TrimSpace,
		"trimAll":    func(a, b string) string { return strings.Trim(b, a) },
		"trimSuffix": func(a, b string) string { return strings.TrimSuffix(b, a) },
		"trimPrefix": func(a, b string) string { return strings.TrimPrefix(b, a) },
		"atoi":       func(a string) int { i, _ := strconv.Atoi(a); return i },
		"int64":      toInt64,
		"int":        toInt,
		"float64":    toFloat64,
		"toJSON":     toJSON,
		"toBSON":     toBSON,
		"toYAML":     toYAML,
		"toXML":      toXML,
		"mapJSON":    mapJSON,
	}
}

func mapJSON(input string) map[string]interface{} {
	var mapData map[string]interface{}
	if err := json.Unmarshal([]byte(input), &mapData); err != nil {
		log.Fatalf("errorMapJSON: %s\n", err.Error())
	}
	return mapData
}

func toJSON(data map[string]interface{}) string {
	if out, err := json.Marshal(data); err != nil {
		return fmt.Sprintf("err: %s", err.Error())
	} else {
		return string(out)
	}
}

func toBSON(data map[string]interface{}) string {
	if out, err := bson.Marshal(data); err != nil {
		return fmt.Sprintf("err: %s", err.Error())
	} else {
		return string(out)
	}
}

func toYAML(data map[string]interface{}) string {
	if out, err := yaml.Marshal(data); err != nil {
		return fmt.Sprintf("err: %s", err.Error())
	} else {
		return string(out)
	}
}

func toXML(data map[string]interface{}) string {
	if out, err := mxj.AnyXml(data); err != nil {
		return fmt.Sprintf("err: %s", err.Error())
	} else {
		return string(out)
	}
}

func luaF(i ...interface{}) string {
	if !luaReady {
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
	} else {
		return "luaError: getResult"
	}
}

func dateFormat(date string, inputFormat string, outputFormat string) string {
	timeParsed, err := time.Parse(inputFormat, date)
	if err != nil {
		return date
	}
	return timeParsed.Format(outputFormat)
}

func now(format string) string {
	return time.Now().Format(format)
}

func add(i ...interface{}) int64 {
	var a int64 = 0
	for _, b := range i {
		a += toInt64(b)
	}
	return a
}

func mul(a interface{}, v ...interface{}) int64 {
	val := toInt64(a)
	for _, b := range v {
		val = val * toInt64(b)
	}
	return val
}

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

func maxf(a interface{}, i ...interface{}) float64 {
	aa := toFloat64(a)
	for _, b := range i {
		bb := toFloat64(b)
		aa = math.Max(aa, bb)
	}
	return aa
}

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

func minf(a interface{}, i ...interface{}) float64 {
	aa := toFloat64(a)
	for _, b := range i {
		bb := toFloat64(b)
		aa = math.Min(aa, bb)
	}
	return aa
}

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

func base64encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func base64decode(v string) string {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func base32encode(v string) string {
	return base32.StdEncoding.EncodeToString([]byte(v))
}

func base32decode(v string) string {
	data, err := base32.StdEncoding.DecodeString(v)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func regexMatch(regex string, s string) bool {
	match, _ := regexp.MatchString(regex, s)
	return match
}

// toFloat64 converts 64-bit floats
func toFloat64(v interface{}) float64 {
	return cast.ToFloat64(v)
}

func toInt(v interface{}) int {
	return cast.ToInt(v)
}

// toInt64 converts integer types to 64-bit integers
func toInt64(v interface{}) int64 {
	return cast.ToInt64(v)
}

func execDecimalOp(a interface{}, b []interface{}, f func(d1, d2 decimal.Decimal) decimal.Decimal) float64 {
	prt := decimal.NewFromFloat(toFloat64(a))
	for _, x := range b {
		dx := decimal.NewFromFloat(toFloat64(x))
		prt = f(prt, dx)
	}
	rslt, _ := prt.Float64()
	return rslt
}
