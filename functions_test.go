package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/shopspring/decimal"
)

func runt(tpl, expect string) error {
	return runtv(tpl, expect, map[string]string{})
}

func runtv(tpl, expect string, vars interface{}) error {
	t := template.Must(template.New("test").Funcs(templateFunctions()).Parse(tpl))
	var b bytes.Buffer
	err := t.Execute(&b, vars)
	if err != nil {
		return err
	}
	if expect != b.String() {
		return fmt.Errorf("Expected '%s', got '%s'", expect, b.String())
	}
	return nil
}

func TestTemplateFunctions(t *testing.T) {
	if err := runt(`{{ add "1" 2 }}`, "3"); err != nil {
		t.Errorf("templateError: %v", err.Error())
	}
}

func TestAdd(t *testing.T) {
	result := add("6", 4)
	if result != 10 {
		t.Errorf("result: %v", result)
	}
}
func TestAdd1(t *testing.T) {
	result := add1("6")
	if result != 7 {
		t.Errorf("result: %v", result)
	}
}

func TestSub(t *testing.T) {
	result := sub("6", 2)
	if result != 4 {
		t.Errorf("result: %v", result)
	}
}

func TestDiv(t *testing.T) {
	result := div("6", 2)
	if result != 3 {
		t.Errorf("result: %v", result)
	}
}

func TestMod(t *testing.T) {
	result := mod("6", 5)
	if result != 1 {
		t.Errorf("result: %v", result)
	}
}
func TestMul(t *testing.T) {
	result := mul("6", 4)
	if result != 24 {
		t.Errorf("result: %v", result)
	}
}

func TestAddf(t *testing.T) {
	result := addf("6.14", 4.12)
	if result != 10.26 {
		t.Errorf("result: %v", result)
	}
}

func TestAdd1f(t *testing.T) {
	result := add1f("6.14")
	if result != 7.14 {
		t.Errorf("result: %v", result)
	}
}

func TestSubf(t *testing.T) {
	result := subf("6.12487", 2.347511)
	if result != 3.777359 {
		t.Errorf("result: %v", result)
	}
}

func TestDivf(t *testing.T) {
	result := divf("6.12487", 2.347511)
	if result != 2.6090910756115733 {
		t.Errorf("result: %v", result)
	}
}

func TestMulf(t *testing.T) {
	result := mulf("6.12487", 2.347511)
	if result != 14.37819969857 {
		t.Errorf("result: %v", result)
	}
}

func TestRandInt(t *testing.T) {
	result := randInt(50, 55)
	if result < 50 || result > 55 {
		t.Errorf("result: %v", result)
	}
}

func TestRound(t *testing.T) {
	if round("6.32487", 2) != 6.32 {
		t.Errorf("result: %v", round("6.32487", 2))
	}
	if round("6.35", 1, 0.6) != 6.3 {
		t.Errorf("result: %v", round("6.35", 1, 0.6))
	}
	if round("6.35", 1, 0.4) != 6.4 {
		t.Errorf("result: %v", round("6.35", 1, 0.4))
	}
}

func TestMax(t *testing.T) {
	if max("6", 4, "12", "5") != 12 {
		t.Errorf("result: %v", max("6", 4, "12", "5"))
	}
}

func TestMin(t *testing.T) {
	if min("6", 4, "12", "5") != 4 {
		t.Errorf("result: %v", min("6", 4, "12", "5"))
	}
}

func TestMaxf(t *testing.T) {
	if maxf("6.32", 4.15, "12.3128", "5") != 12.3128 {
		t.Errorf("result: %v", maxf("6.32", 4.15, "12.3128", "5"))
	}
}

func TestMinf(t *testing.T) {
	if minf("6.32", 4.15, "12.3128", "5") != 4.15 {
		t.Errorf("result: %v", minf("6.32", 4.15, "12.3128", "5"))
	}
}

func TestDateFormat(t *testing.T) {
	if dateFormat("15.03.2021", "02.01.2006", "01022006") != "03152021" {
		t.Errorf("result: %s", dateFormat("15.03.2021", "02.01.2006", "01022006"))
	}
	if dateFormat("Hello", "World", "01022006") != "Hello" {
		t.Errorf("result: %s", dateFormat("Hello", "World", "01022006"))
	}
}

func TestNow(t *testing.T) {
	if now("2006-01-02 15:04") != time.Now().Format("2006-01-02 15:04") {
		t.Errorf("result: %s", now("2006-01-02 15:04"))
	}
}

func TestBase64encode(t *testing.T) {
	if base64encode("Hello World!") != "SGVsbG8gV29ybGQh" {
		t.Errorf("result: %v", base64encode("Hello World!"))
	}
}

func TestBase64decode(t *testing.T) {
	if base64decode("SGVsbG8gV29ybGQh") != "Hello World!" {
		t.Errorf("result: %v", base64decode("SGVsbG8gV29ybGQh"))
	}
	if base64decode("Hello") != "illegal base64 data at input byte 4" {
		t.Errorf("result: %v", base64decode("Hello"))
	}
}

func TestBase32encode(t *testing.T) {
	if base32encode("Hello World!") != "JBSWY3DPEBLW64TMMQQQ====" {
		t.Errorf("result: %v", base32encode("Hello World!"))
	}
}

func TestBase32decode(t *testing.T) {
	if base32decode("JBSWY3DPEBLW64TMMQQQ====") != "Hello World!" {
		t.Errorf("result: %v", base32decode("JBSWY3DPEBLW64TMMQQQ===="))
	}
	if base32decode("Hello") != "illegal base32 data at input byte 1" {
		t.Errorf("result: %v", base32decode("Hello"))
	}
}

func TestRegexMatch(t *testing.T) {
	if !regexMatch("a.b", "aaxbb") {
		t.Errorf("result: %v", regexMatch(`^a.b$`, "aaxbb"))
	}
}

func TestUUID(t *testing.T) {
	testUUID := newUUID()
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	if !r.MatchString(testUUID) {
		t.Errorf("result: %v", testUUID)
	}
}

func TestUpper(t *testing.T) {
	if upper("Hello") != "HELLO" {
		t.Errorf("result: %v", upper("Hello"))
	}
}

func TestLower(t *testing.T) {
	if lower("World") != "world" {
		t.Errorf("result: %v", lower("World"))
	}
}

func TestTrim(t *testing.T) {
	result := trim("\r\nHello World\r\n")
	if result != "Hello World" {
		t.Errorf("result: %v", result)
	}
}

func TestTrimAll(t *testing.T) {
	result := trimAll("!Hello World!", "!")
	if result != "Hello World" {
		t.Errorf("result: %v", result)
	}
}

func TestPrefix(t *testing.T) {
	result := trimPrefix("!Hello World!", "!")
	if result != "Hello World!" {
		t.Errorf("result: %v", result)
	}
}

func TestSuffix(t *testing.T) {
	result := trimSuffix("!Hello World!", "!")
	if result != "!Hello World" {
		t.Errorf("result: %v", result)
	}
}

func TestAtoi(t *testing.T) {
	result := atoi("42")
	if result != 42 {
		t.Errorf("result: %v", result)
	}
}

func TestToInt(t *testing.T) {
	if toInt("42") != 42 {
		t.Errorf("result: %d", toInt("42"))
	}
}

func TestToInt64(t *testing.T) {
	if toInt64("42") != 42 {
		t.Errorf("result: %d", toInt64("42"))
	}
}

func TestToFloat64(t *testing.T) {
	if toFloat64("3.14159265") != 3.14159265 {
		t.Errorf("result: %v", toFloat64("3.14159265"))
	}
}

func TestLuaF(t *testing.T) {
	if luaF("sum", "5", "5") != "10" {
		t.Errorf("result: %s", luaF("sum", "5", "5"))
	}
	if !strings.Contains(luaF("Unknown", "5", "5"), `attempt to call a non-function object`) {
		t.Errorf("result: %s", luaF("Unknown", "5", "5"))
	}
	testData := make(map[string]interface{})
	testData["Hello"] = make(chan int)
	if !strings.Contains(luaF("sum", testData), "luaInputError: json: unsupported type: chan int") {
		t.Errorf("result: %v", luaF("sum", testData))
	}
}

func TestToJSON(t *testing.T) {
	testData := make(map[string]interface{})
	testData["Hello"] = "World"
	result := toJSON(testData)
	if result != `{"Hello":"World"}` {
		t.Errorf("result: %v", result)
	}
	testData["Hello"] = make(chan int)
	result = toJSON(testData)
	if result != "err: json: unsupported type: chan int" {
		t.Errorf("result: %v", result)
	}
}

func TestToBSON(t *testing.T) {
	testData := make(map[string]interface{})
	testData["h"] = "w"
	result := toBSON(testData)
	if result != string([]byte{14, 0, 0, 0, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0}) {
		t.Errorf("result: %v", []byte(result))
	}
	testData["Hello"] = make(chan int)
	result = toBSON(testData)
	if result != "err: no encoder found for chan int" {
		t.Errorf("result: %v", result)
	}
}

func TestToYAML(t *testing.T) {
	testData := make(map[string]interface{})
	testData["Hello"] = "World"
	result := toYAML(testData)
	if result != `Hello: World
` {
		t.Errorf("result: %v", result)
	}
}

func TestToXML(t *testing.T) {
	testData := make(map[string]interface{})
	testData["Hello"] = "World"
	result := toXML(testData)
	if result != `<doc><Hello>World</Hello></doc>` {
		t.Errorf("result: %v", result)
	}
}

func TestMapJSON(t *testing.T) {
	testData := "{\"Hello\":\"World\"}"
	result := mapJSON(testData)
	if result["Hello"] != "World" {
		t.Errorf("result: %v", result["Hello"])
	}
}

func TestExecDecimalOp(t *testing.T) {
	testMulf := func(a interface{}, v ...interface{}) float64 {
		return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Mul(d2) })
	}
	if testMulf(6.2154, "4.35") != 27.03699 {
		t.Errorf("result: %v", testMulf(6.2154, "4.35"))
	}
	testDivf := func(a interface{}, v ...interface{}) float64 {
		return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Div(d2) })
	}
	if testDivf(6.2154, "4.35") != 1.4288275862068966 {
		t.Errorf("result: %v", testDivf(6.2154, "4.35"))
	}
}

// go test -coverprofile cover.out
// go tool cover -html='cover.out'
