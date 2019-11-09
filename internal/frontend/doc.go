// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frontend

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"strings"

	"golang.org/x/discovery/internal"
	"golang.org/x/discovery/internal/log"
	"golang.org/x/discovery/internal/stdlib"
)

// DocumentationDetails contains data for the doc template.
type DocumentationDetails struct {
	GOOS          string
	GOARCH        string
	Documentation template.HTML
}

// doDocumentationHack controls whether to use a regexp replacement to append
// ?tab=doc to urls linking to package identifiers within the documentation.
//
// This is a temporary measure so that we don't have to re-process all the
// documentation in order to make this trivial change.
var doDocumentationHack = os.Getenv("GO_DISCOVERY_DOCUMENTATION_HACK") == "TRUE"

// fetchDocumentationDetails fetches data for the package specified by path and version
// from the database and returns a DocumentationDetails.
func fetchDocumentationDetails(ctx context.Context, ds DataSource, pkg *internal.VersionedPackage) (*DocumentationDetails, error) {
	docBytes := pkg.DocumentationHTML
	if doDocumentationHack {
		docBytes = hackUpDocumentation(docBytes)
	}
	return &DocumentationDetails{
		GOOS:          pkg.GOOS,
		GOARCH:        pkg.GOARCH,
		Documentation: template.HTML(docBytes),
	}, nil
}

// packageLinkRegexp matches cross-package identifier links that have been
// generated by the dochtml package. At the time this hack was added, these
// links are all constructed to have either the form
//   <a href="/pkg/[path]">[name]</a>
// or the form
//   <a href="/pkg/[path]#identifier">[name]</a>
//
// The packageLinkRegexp mutates these links as follows:
//   - remove the now unnecessary '/pkg' path prefix
//   - add an explicit ?tab=doc after the path.
var packageLinkRegexp = regexp.MustCompile(`(<a href="/)pkg/([^?#"]+)((?:#[^"]*)?">.*?</a>)`)

func hackUpDocumentation(docBytes []byte) []byte {
	return packageLinkRegexp.ReplaceAll(docBytes, []byte(`$1$2?tab=doc$3`))
}

// fileSource returns the original filepath in the module zip where the given
// filePath can be found. For std, the corresponding URL in
// go.google.source.com/go is returned.
func fileSource(modulePath, version, filePath string) string {
	if modulePath != stdlib.ModulePath {
		return fmt.Sprintf("%s@%s/%s", modulePath, version, filePath)
	}

	root := strings.TrimPrefix(stdlib.GoRepoURL, "https://")
	tag, err := stdlib.TagForVersion(version)
	if err != nil {
		// This should never happen unless there is a bug in
		// stdlib.TagForVersion. In which case, fallback to the default
		// zipFilePath.
		log.Errorf("fileSource: %v", err)
		return fmt.Sprintf("%s/+/refs/heads/master/%s", root, filePath)
	}
	return fmt.Sprintf("%s/+/refs/tags/%s/%s", root, tag, filePath)
}
