// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package files

import (
	. "golang.org/x/tools/gopls/internal/test/integration"
	"golang.org/x/tools/gopls/internal/util/bug"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	bug.PanicOnBugs = true
	os.Exit(Main(m))
}

func TestDidCreateFiles(t *testing.T) {
	const files = `
-- go.mod --
module mod.com

go 1.12
-- fruits/newfile.go --
-- fruits/apple.go --
package apple

fun apple() int {
	return 0
}

-- fruits/testfile.go --
// this is a comment

/*
 this is a multiline comment
*/

import "fmt"

func test() {}

-- fruits/testfile2.go --
package

-- fruits/testfile3.go --
pac
-- 123f_r.u~its-123/testfile.go --
package

-- .invalid-dir@-name/testfile.go --
package
`

	for _, tc := range []struct {
		name     string
		filename string
	}{
		{
			name:     "package completion at valid position",
			filename: "fruits/testfile.go",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			Run(t, files, func(t *testing.T, env *Env) {
				f := "fruits/newfile.go"
				env.OpenFile(tc.filename)
				env.DidCreateFiles(string(env.Editor.DocumentURI(f)))
				t.Log(env.FileContent(f))
			})
		})
	}
}
