// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package files

import (
	. "golang.org/x/tools/gopls/internal/test/integration"
	"golang.org/x/tools/gopls/internal/util/bug"
	"os"
	"testing"
	"time"
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
-- fruits/apple.go --
package apple

fun apple() int {
	return 0
}
-- cmd/main.go --
package main
-- test/test_test.go --
package test

-- test/newfile.go --
-- fruits/newfile.go --
-- fruits/newfile_test.go --
-- cmd/newfile.go --
-- empty_folder/newfile.go --
`

	for _, tc := range []struct {
		name    string
		newfile string
		want    string
	}{
		{
			name:    "new go file under a folder with a go test file",
			newfile: "test/newfile.go",
			want:    "package test\n",
		},
		{
			name:    "new file under a folder with a go file",
			newfile: "fruits/newfile.go",
			want:    "package apple\n",
		},
		{
			name:    "new go file under a folder with a go file",
			newfile: "fruits/newfile.go",
			want:    "package apple\n",
		},
		{
			name:    "new go test file under a folder with a go file",
			newfile: "fruits/newfile_test.go",
			want:    "package apple\n",
		},
		{
			name:    "new go file under main package",
			newfile: "cmd/newfile.go",
			// TODO(yuchen) we want main,
			//  so blindly pick up the first element from suggested package name won't take effect.
			want: "package cmd\n",
		},
		{
			name:    "new go file under an empty folder",
			newfile: "empty_folder/newfile.go",
			want:    "package emptyfolder\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			Run(t, files, func(t *testing.T, env *Env) {
				env.DidCreateFiles(string(env.Editor.DocumentURI(tc.newfile)))
				time.Sleep(time.Second) // todo(yuchen): sleep a second makes the test passes.
				got := env.FileContent(tc.newfile)
				if tc.want != got {
					t.Fatalf("want '%s' but got '%s'", tc.want, got)
				}
			})
		})
	}
}
