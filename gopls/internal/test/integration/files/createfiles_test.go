// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package files

import (
	"os"
	"testing"
	"time"

	. "golang.org/x/tools/gopls/internal/test/integration"
	"golang.org/x/tools/gopls/internal/util/bug"
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
`

	for _, tc := range []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "package completion at valid position",
			filename: "fruits/testfile.go",
			want:     "package apple\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			Run(t, files, func(t *testing.T, env *Env) {
				f := "fruits/newfile.go"
				env.OpenFile(f)
				env.DidCreateFiles(string(env.Editor.DocumentURI(f)))
				time.Sleep(time.Second)
				got := env.FileContent(f)
				if tc.want != got {
					t.Fatalf("want %s but got %s", tc.want, got)
				}
			})
		})
	}
}
