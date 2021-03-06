package util

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/logrusorgru/aurora"
)

const (
	expectedValueString = "foo"
	expectedValueInt    = 10
)

func TestPickStringLast(t *testing.T) {
	if v := PickString("", "", expectedValueString); v != expectedValueString {
		t.Fatalf("expected %s, got %s", expectedValueString, v)
	}
}

func TestPickStringNoValue(t *testing.T) {
	if v := PickString(""); v != "" {
		t.Fatalf("expected '%s', got '%s'", "", v)
	}
}

func TestPickStringFirst(t *testing.T) {
	if v := PickString(expectedValueString, "bar"); v != expectedValueString {
		t.Fatalf("expected %s, got %s", expectedValueString, v)
	}
}

func TestPickIntLast(t *testing.T) {
	if v := PickInt(0, 0, expectedValueInt); v != expectedValueInt {
		t.Fatalf("expected %d, got %d", expectedValueInt, v)
	}
}

func TestPickIntNoValue(t *testing.T) {
	if v := PickInt(0); v != 0 {
		t.Fatalf("expected %d, got %d", 0, v)
	}
}

func TestPickIntFirst(t *testing.T) {
	if v := PickInt(expectedValueInt, 5); v != expectedValueInt {
		t.Fatalf("expected %d, got %d", expectedValueInt, v)
	}
}

func TestIndent(t *testing.T) {
	expected := "   foo"
	if a := Indent("foo", "   "); a != expected {
		t.Fatalf("expected '%s', got '%s'", expected, a)
	}
}

func TestIndentWithNewline(t *testing.T) {
	expected := "  foo\n  bar\n"
	if a := Indent("foo\nbar\n", "  "); a != expected {
		t.Fatalf("expected '%s', got '%s'", expected, a)
	}
}

func TestIndentEmpty(t *testing.T) {
	expected := ""
	if a := Indent("", ""); a != expected {
		t.Fatalf("expected '%s', got '%s'", expected, a)
	}
}

func TestIndentEmptyText(t *testing.T) {
	expected := ""
	if a := Indent("", "  "); a != expected {
		t.Fatalf("expected '%s', got '%s'", expected, a)
	}
}

func TestIndentEmptyIndent(t *testing.T) {
	expected := "foo\nbar"
	if a := Indent("foo\nbar", ""); a != expected {
		t.Fatalf("expected '%s', got '%s'", expected, a)
	}
}

func TestJoinSorted(t *testing.T) {
	expected := "baz/doh|foo/bar"

	values := map[string]string{
		"foo": "bar",
		"baz": "doh",
	}

	if a := JoinSorted(values, "/", "|"); a != expected {
		t.Fatalf("expected '%s', got '%s'", expected, a)
	}
}

func TestPickStringSlice(t *testing.T) {
	expected := []string{"foo"}
	if v := PickStringSlice([]string{}, expected); !reflect.DeepEqual(v, expected) {
		t.Fatalf("Expected %v, got %v", expected, v)
	}
}

func TestPickStringMap(t *testing.T) {
	expected := map[string]string{"foo": "bar"}
	if v := PickStringMap(map[string]string{}, expected); !reflect.DeepEqual(v, expected) {
		t.Fatalf("Expected %v, got %v", expected, v)
	}
}

func TestPickStringSliceEmpty(t *testing.T) {
	expected := []string{}
	if v := PickStringSlice([]string{}, expected); !reflect.DeepEqual(v, expected) {
		t.Fatalf("Expected %v, got %v", expected, v)
	}
}

func TestPickStringMapEmpty(t *testing.T) {
	expected := map[string]string{}
	if v := PickStringMap(map[string]string{}, expected); !reflect.DeepEqual(v, expected) {
		t.Fatalf("Expected %v, got %v", expected, v)
	}
}

func TestKeysStringMap(t *testing.T) {
	expected := []string{"baz", "foo"}
	m := map[string]string{
		"foo": "bar",
		"baz": "doh",
	}

	if k := KeysStringMap(m); !reflect.DeepEqual(expected, k) {
		t.Fatalf("Expected %v, got %v", expected, k)
	}
}

func TestColorizeDiff(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input  string
		output string
	}{
		{
			input:  "",
			output: "",
		},
		{
			input:  "\n",
			output: "\n",
		},
		{
			input:  "foo\n",
			output: "foo\n",
		},
		{
			input:  "+foo\n",
			output: aurora.Green("+foo\n").String(),
		},
		{
			input:  "-foo\n",
			output: aurora.Red("-foo\n").String(),
		},
		{
			input:  "foo\nbar",
			output: "foo\nbar",
		},
		{
			input:  "+foo\n-bar\nbaz\n",
			output: aurora.Green("+foo\n").String() + aurora.Red("-bar\n").String() + "baz\n",
		},
	}

	for n, c := range cases {
		c := c

		t.Run(strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()

			if result := ColorizeDiff(c.input); result != c.output {
				t.Errorf("expected %q, got %q", c.output, result)
			}
		})
	}
}
