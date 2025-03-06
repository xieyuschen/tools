package stringscutprefix

import (
	"bytes"
	"strings"
)

// test the pattern 1
func _() {
	var s, pre string
	if strings.HasPrefix(s, pre) { // want "if statement can be modernized using strings.CutPrefix"
		a := strings.TrimPrefix(s, pre)
		_ = a
	}
	if strings.HasPrefix("", "") { // want "if statement can be modernized using strings.CutPrefix"
		a := strings.TrimPrefix("", "")
		_ = a
	}
	if strings.HasPrefix(s, "") { // want "if statement can be modernized using strings.CutPrefix"
		println([]byte(strings.TrimPrefix(s, "")))
	}
	if strings.HasPrefix(s, "") { // want "if statement can be modernized using strings.CutPrefix"
		a, b := "", strings.TrimPrefix(s, "")
		_, _ = a, b
	}
	var bss, bspre []byte
	if bytes.HasPrefix(bss, bspre) { // want "if statement can be modernized using strings.CutPrefix"
		a := bytes.TrimPrefix(bss, bspre)
		_ = a
	}
	if bytes.HasPrefix([]byte(""), []byte("")) { // want "if statement can be modernized using strings.CutPrefix"
		a := bytes.TrimPrefix([]byte(""), []byte(""))
		_ = a
	}
	var a, b string
	if strings.HasPrefix(s, "") { // want "if statement can be modernized using strings.CutPrefix"
		a, b = "", strings.TrimPrefix(s, "")
		_, _ = a, b
	}

	ok := strings.HasPrefix("", "")
	if ok { // noop, currently it doesn't track the result usage of HasPrefix
		a := strings.TrimPrefix("", "")
		_ = a
	}

	if strings.HasPrefix(s, pre) {
		a := strings.TrimPrefix("", "") // noop, as the argument isn't the same
		_ = a
	}

	if strings.HasPrefix("", "") {
		a := strings.TrimPrefix(s, pre) // noop, as the argument isn't the same
		_ = a
	}
}

var value0 string

// test pattern2
func _() {
	var s, pre string
	if after := strings.TrimPrefix(s, pre); after != s { // want "if statement can be modernized using strings.CutPrefix"
		println(after)
	}
	if after := strings.TrimPrefix(s, pre); s != after { // want "if statement can be modernized using strings.CutPrefix"
		println(after)
	}
	if after := strings.TrimPrefix(s, pre); s != after { // want "if statement can be modernized using strings.CutPrefix"
		println(strings.TrimPrefix(s, pre)) // noop here
	}
	if after := strings.TrimPrefix(s, ""); s != after { // want "if statement can be modernized using strings.CutPrefix"
		println(after)
	}
	var predefined string
	if predefined = strings.TrimPrefix(s, pre); s != predefined { // want "if statement can be modernized using strings.CutPrefix"
		println(predefined)
	}
	var value string
	if value = strings.TrimPrefix(s, pre); s != value { // want "if statement can be modernized using strings.CutPrefix"
		println(value)
	}

	if after := strings.TrimPrefix(s, pre); s != pre { // noop
		println(after)
	}
	if after := strings.TrimPrefix(s, pre); after != pre { // noop
		println(after)
	}
}

// the uncovered cases now, but possible to support in the future.
func _() {
	var s, pre string
	if strings.TrimPrefix(s, pre) != s {
		println(strings.TrimPrefix(s, pre))
	}
}
