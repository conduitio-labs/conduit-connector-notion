package main

import (
	sdk "github.com/conduitio/conduit-connector-sdk"

	notion "github.com/github.com/conduitio-labs/conduit-connector-notion"
)

func main() {
	sdk.Serve(notion.Connector)
}
