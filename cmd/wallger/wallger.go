// Copyright (c) The Cryptorium Authors.
// Licensed under the MIT License.

package main

import (
	"context"
	"os"
	"reflect"

	"github.com/alecthomas/kong"
	cli_p "github.com/cryptoriums/packages/cli"
	"github.com/cryptoriums/packages/logging"
	"github.com/cryptoriums/wallger/pkg/cli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-kit/log"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

func main() {
	l := logging.NewLogger()
	c := context.Background()

	parser := kong.Must(
		&cli.CLIInstance,
		kong.BindTo(c, (*context.Context)(nil)), kong.BindTo(l, (*log.Logger)(nil)),
		kong.Name("wallger"),
		kong.UsageOnError(),
		kong.TypeMapper(reflect.TypeOf((*common.Address)(nil)).Elem(), cli_p.AddressDecoder()),
	)

	// Run kongplete.Complete to handle completion requests
	kongplete.Complete(parser, kongplete.WithPredictor("wallger", complete.PredictAnything))

	ctx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	ctx.FatalIfErrorf(ctx.Run(*ctx))
}
