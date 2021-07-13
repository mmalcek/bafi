package main

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestLuaF(t *testing.T) {
	if luaF("sum", "5", "5") != "10" {
		t.Errorf("luaF unexpected result: %s", luaF("sum", "5", "5"))
	}
}

func TestDateFormat(t *testing.T) {
	if dateFormat("15.03.2021", "02.01.2006", "01022006") != "03152021" {
		t.Errorf("dateFormat unexpected result: %s", dateFormat("15.03.2021", "02.01.2006", "01022006"))
	}
}

func TestAdd(t *testing.T) {
	if add("6", 4) != 10 {
		t.Errorf("add unexpected result: %v", add("6", 4))
	}
}

func TestMul(t *testing.T) {
	if mul("6", 4) != 24 {
		t.Errorf("mul unexpected result: %v", mul("6", 4))
	}
}

func TestMax(t *testing.T) {
	if max("6", 4, "12", "5") != 12 {
		t.Errorf("max unexpected result: %v", max("6", 4, "12", "5"))
	}
}

func TestMaxf(t *testing.T) {
	if maxf("6.32", 4.15, "12.3128", "5") != 12.3128 {
		t.Errorf("maxf unexpected result: %v", maxf("6.32", 4.15, "12.3128", "5"))
	}
}

func TestMin(t *testing.T) {
	if min("6", 4, "12", "5") != 4 {
		t.Errorf("min unexpected result: %v", min("6", 4, "12", "5"))
	}
}

func TestMinf(t *testing.T) {
	if minf("6.32", 4.15, "12.3128", "5") != 4.15 {
		t.Errorf("minf unexpected result: %v", minf("6.32", 4.15, "12.3128", "5"))
	}
}

func TestRound(t *testing.T) {
	if round("6.32487", 2) != 6.32 {
		t.Errorf("round unexpected result: %v", round("6.32487", 2))
	}
}

func TestBase64encode(t *testing.T) {
	if base64encode("Hello World!") != "SGVsbG8gV29ybGQh" {
		t.Errorf("base64encode unexpected result: %v", base64encode("Hello World!"))
	}
}

func TestBase64decode(t *testing.T) {
	if base64decode("SGVsbG8gV29ybGQh") != "Hello World!" {
		t.Errorf("base64decode unexpected result: %v", base64decode("SGVsbG8gV29ybGQh"))
	}
}

func TestBase32encode(t *testing.T) {
	if base32encode("Hello World!") != "JBSWY3DPEBLW64TMMQQQ====" {
		t.Errorf("base32encode unexpected result: %v", base32encode("Hello World!"))
	}
}

func TestBase32decode(t *testing.T) {
	if base32decode("JBSWY3DPEBLW64TMMQQQ====") != "Hello World!" {
		t.Errorf("base32decode unexpected result: %v", base32decode("JBSWY3DPEBLW64TMMQQQ===="))
	}
}

func TestToFloat64(t *testing.T) {
	if toFloat64("3.14159265") != 3.14159265 {
		t.Errorf("toFloat64 unexpected result: %v", toFloat64("3.14159265"))
	}
}

func TestToInt(t *testing.T) {
	if toInt("42") != 42 {
		t.Errorf("toInt unexpected result: %d", toInt("42"))
	}
}

func TestToInt64(t *testing.T) {
	if toInt64("42") != 42 {
		t.Errorf("toInt unexpected result: %d", toInt64("42"))
	}
}

func TestRegexMatch(t *testing.T) {
	if !regexMatch(`a.b`, "aaxbb") {
		t.Errorf("regexMatch unexpected result: %v", regexMatch(`^a.b$`, "aaxbb"))
	}
}

func TestExecDecimalOp(t *testing.T) {
	testMulf := func(a interface{}, v ...interface{}) float64 {
		return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Mul(d2) })
	}
	if testMulf(6.2154, "4.35") != 27.03699 {
		t.Errorf("testMulf unexpected result: %v", testMulf(6.2154, "4.35"))
	}
	testDivf := func(a interface{}, v ...interface{}) float64 {
		return execDecimalOp(a, v, func(d1, d2 decimal.Decimal) decimal.Decimal { return d1.Div(d2) })
	}
	if testDivf(6.2154, "4.35") != 1.4288275862068966 {
		t.Errorf("testDivf unexpected result: %v", testDivf(6.2154, "4.35"))
	}
}
