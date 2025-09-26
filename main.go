// Package main は、gopose コマンドラインツールのエントリーポイントです。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/harakeishi/gopose/cmd"
)

var version = "dev"

func main() {
	ctx := context.Background()

	if err := cmd.Execute(ctx, version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
