package server

import (
	"context"
	"fmt"
	"go/token"

	"golang.org/x/tools/gopls/internal/golang"
	"golang.org/x/tools/gopls/internal/golang/completion"
	"golang.org/x/tools/gopls/internal/protocol"
	"golang.org/x/tools/internal/event"
)

// DidCreateFiles should have the following behaviors:
//
// first we need to select a package name for the new created file,
// and the package name is determined by 2 factors, the files/folder around and whether it's a test file.
// Factor 1:
//   - the file is created under a folder without any .go file
//     => pick up the folder name after converting all alpha bytes to lowercase.
//   - the file is created under a folder with some .go file
//     => xx.go only: use package name inside xx.go
//     => xx.go and xx_test.go: use package name inside xx.go
//     => xx_test.go only: use package name after trim _test suffix if any inside xx_test.go
//
// Factor 2:
//   - the file doesn't have _test.go suffix
//     => use the package name in factor 1 directly.
//   - the file has _test.go suffix
//     => append a _test suffix after the package name got from factor1
func (s *server) DidCreateFiles(ctx context.Context, params *protocol.CreateFilesParams) error {
	event.Log(ctx, fmt.Sprintf("+ ycx start to handle didCreateFiles event"))
	ctx, done := event.Start(ctx, "lsp.Server.didCreateFiles")
	defer done()

	for _, fileCreate := range params.Files {
		event.Log(ctx, fmt.Sprintf("++ %s", fileCreate.URI))
		uri := protocol.DocumentURI(fileCreate.URI)
		fh, snapshot, release, err := s.fileOf(ctx, uri)
		if err != nil {
			event.Error(ctx, "fail to call fileOf", err)
			continue
		}
		defer release()
		// todo(yuchen): check whether it's a valid case that when gopls
		// receives DidCreateFiles request, the file is already modified.

		_, pgf, err := golang.NarrowestPackageForFile(ctx, snapshot, fh.URI())
		if err != nil || pgf.File.Package == token.NoPos {
			// If we can't parse this file or find position for the package
			// keyword, it may be missing a package declaration. Try offering
			// suggestions for the package declaration.
			// Note that this would be the case even if the keyword 'package' is
			// present but no package name exists.
			items, _, innerErr := completion.PackageClauseCompletions(ctx, snapshot, fh, protocol.Position{})
			if innerErr != nil {
				event.Error(ctx, "fail to get package completion: ", innerErr)
				continue
			}

			if len(items) != 0 {
				edit := protocol.DocumentChangeEdit(fh, []protocol.TextEdit{
					{
						Range:   protocol.Range{},
						NewText: items[0].InsertText + "\n",
					},
				})
				result, err := s.client.ApplyEdit(ctx, &protocol.ApplyWorkspaceEditParams{
					Edit: *protocol.NewWorkspaceEdit(edit),
				})
				if err != nil {
					event.Log(ctx, err.Error())
					continue
				}
				if !result.Applied {
					event.Log(ctx, result.FailureReason)
					continue
				}
			}
		}
	}

	return nil
}
