package prompt

import (
	"errors"
	"reflect"
	"testing"
)

func evalOne(t *testing.T, expr string, vars Vars) (any, error) {
	t.Helper()
	return evalValue(expr, vars)
}

func TestExprLiterals(t *testing.T) {
	for _, tc := range []struct {
		expr string
		want any
	}{
		{"3", 3.0},
		{"1 + 2 * 3", 7.0},
		{"(1 + 2) * 3", 9.0},
		{"5 / 2", 2.5},
		{"7 - 2 - 1", 4.0},
		{`"a" + "b"`, "ab"},
		{"-5", -5.0},
		{"--5", 5.0},
		{"true && false", false},
		{"true && true", true},
		{"true || false", true},
		{"!true", false},
		{"!false", true},
		{"3 > 2", true},
		{"3 <= 3", true},
		{"2 != 2", false},
		{"2 == 2.0", true},
		{`"abc" < "abd"`, true},
		{"nil == nil", true},
		{"nil != 1", true},
		{"(2 == 2) && !false", true},
	} {
		got, err := evalOne(t, tc.expr, nil)
		if err != nil {
			t.Fatalf("%q: %v", tc.expr, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q = %#v, want %#v", tc.expr, got, tc.want)
		}
	}
}

func TestExprVars(t *testing.T) {
	vars := Vars{
		"name":  "Omer",
		"count": 3.0,
		"stats": map[string]any{"count": 5.0},
		"ok":    true,
	}
	for _, tc := range []struct {
		expr string
		want any
	}{
		{"name", "Omer"},
		{"count", 3.0},
		{"stats.count", 5.0},
		{`name == "Omer"`, true},
		{"ok == true", true},
		{"count + stats.count", 8.0},
		{`name + "!"`, "Omer!"},
	} {
		got, err := evalOne(t, tc.expr, vars)
		if err != nil {
			t.Fatalf("%q: %v", tc.expr, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q = %#v, want %#v", tc.expr, got, tc.want)
		}
	}
}

func TestExprShortCircuit(t *testing.T) {
	vars := Vars{"ok": true}
	if got, err := evalOne(t, "false && missing_var", vars); err != nil || got != false {
		t.Fatalf("false && missing: got=%v err=%v", got, err)
	}
	if got, err := evalOne(t, "true || missing_var", vars); err != nil || got != true {
		t.Fatalf("true || missing: got=%v err=%v", got, err)
	}
	if got, err := evalOne(t, "ok || missing_var", vars); err != nil || got != true {
		t.Fatalf("ok || missing: got=%v err=%v", got, err)
	}
}

func TestExprErrors(t *testing.T) {
	for _, tc := range []struct {
		expr string
		vars Vars
	}{
		{"missing_var", Vars{}},
		{"1 > true", nil},
		{"5 / 0", nil},
		{"1 + true", nil},
		{"!1", nil},
		{"-true", nil},
		{`"a" < 2`, nil},
		{"1.2.3", nil},
		{"(", nil},
		{"1 2", nil},
		{`"unterminated`, nil},
		{"@bad", nil},
		{"", nil},
	} {
		if _, err := evalOne(t, tc.expr, tc.vars); err == nil {
			t.Fatalf("%q: expected error", tc.expr)
		}
	}
}

func TestExprDefined(t *testing.T) {
	vars := Vars{
		"role":    "coder",
		"project": map[string]any{"license": "MIT"},
	}
	for _, tc := range []struct {
		expr string
		want any
	}{
		{`defined("role")`, true},
		{`defined("nope")`, false},
		{`defined("project.license")`, true},
		{`defined("project.nope")`, false},
		{`defined(role)`, true},
		{`defined(nope)`, false},
		{`!defined("nope")`, true},
		{`defined("nope") == false`, true},
		{`defined("role") && role == "coder"`, true},
	} {
		got, err := evalOne(t, tc.expr, vars)
		if err != nil {
			t.Fatalf("%q: %v", tc.expr, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q = %#v, want %#v", tc.expr, got, tc.want)
		}
	}

	if _, err := evalOne(t, `defined()`, vars); err == nil {
		t.Fatal("expected error for defined() without arguments")
	}
	if _, err := evalOne(t, `defined(1)`, vars); err == nil {
		t.Fatal("expected error for defined() with non-string argument")
	}
	if _, err := evalOne(t, `unknown("x")`, vars); err == nil {
		t.Fatal("expected error for unknown function")
	}
}

func TestExprLen(t *testing.T) {
	vars := Vars{
		"tools":    []string{"a", "b"},
		"empty":    []string{},
		"settings": map[string]any{"timeout": 30},
		"name":     "hello",
	}
	for _, tc := range []struct {
		expr string
		want any
	}{
		{`len("tools")`, 2.0},
		{`len(tools)`, 2.0},
		{`len("empty")`, 0.0},
		{`len("settings")`, 1.0},
		{`len("name")`, 5.0},
		{`len("tools") > 0`, true},
		{`len("empty") == 0`, true},
		{`len("tools") > 0 && len("empty") == 0`, true},
	} {
		got, err := evalOne(t, tc.expr, vars)
		if err != nil {
			t.Fatalf("%q: %v", tc.expr, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q = %#v, want %#v", tc.expr, got, tc.want)
		}
	}

	if _, err := evalOne(t, `len()`, vars); err == nil {
		t.Fatal("expected error for len() without arguments")
	}
	if _, err := evalOne(t, `len(42)`, vars); err == nil {
		t.Fatal("expected error for len() of a number")
	}
	if _, err := evalOne(t, `len("missing")`, vars); err == nil {
		t.Fatal("expected error for len() of missing var")
	}
}

func TestExprEmpty(t *testing.T) {
	vars := Vars{
		"emptyString": "",
		"text":        "hi",
		"emptyList":   []string{},
		"list":        []string{"a"},
		"emptyMap":    map[string]any{},
		"nothing":     nil,
	}
	for _, tc := range []struct {
		expr string
		want any
	}{
		{`empty("emptyString")`, true},
		{`empty(emptyString)`, true},
		{`empty("text")`, false},
		{`!empty("text")`, true},
		{`empty("emptyList")`, true},
		{`empty("list")`, false},
		{`empty("emptyMap")`, true},
		{`empty("nothing")`, true},
		{`empty("list") == false`, true},
		{`!empty("text") && !empty("list")`, true},
	} {
		got, err := evalOne(t, tc.expr, vars)
		if err != nil {
			t.Fatalf("%q: %v", tc.expr, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%q = %#v, want %#v", tc.expr, got, tc.want)
		}
	}

	if _, err := evalOne(t, `empty()`, vars); err == nil {
		t.Fatal("expected error for empty() without arguments")
	}
	if _, err := evalOne(t, `empty("missing")`, vars); err == nil {
		t.Fatal("expected error for empty() of missing var")
	}
	if _, err := evalOne(t, `empty(42)`, vars); err == nil {
		t.Fatal("expected error for empty() of a number")
	}
}

func TestExprMissingVarWrapped(t *testing.T) {
	_, err := evalOne(t, "missing_var", Vars{})
	if !errors.Is(err, ErrMissingVar) {
		t.Fatalf("expected ErrMissingVar, got %v", err)
	}
}

func TestEvalBoolRequiresBool(t *testing.T) {
	if _, err := evalBool("1 + 1", nil); err == nil {
		t.Fatal("expected error for non-boolean condition")
	}
	if ok, err := evalBool("2 > 1", nil); err != nil || !ok {
		t.Fatalf("evalBool(2>1) = %v, %v", ok, err)
	}
}
