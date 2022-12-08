package main

import (
	sdk "github.com/conduitio/conduit-connector-sdk"

	notion "github.com/conduitio-labs/conduit-connector-notion"
)

func main() {
	sdk.Serve(
		notion.Specification,
		notion.NewSource,
		nil,
	)
}
