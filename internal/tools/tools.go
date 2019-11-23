// +build tools

package tools

import (
	_ "github.com/goreleaser/goreleaser"
	_ "github.com/goreleaser/nfpm"
	_ "golang.org/x/lint/golint"
)
