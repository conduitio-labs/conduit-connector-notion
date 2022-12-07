package notion

import (
	sdk "github.com/conduitio/conduit-connector-sdk"
)

// Specification returns the connector's specification.
func Specification() sdk.Specification {
	return sdk.Specification{
		Name:         "notion",
		Summary:      "A Conduit connector for Notion.",
		Description:  "A Conduit connector for Notion.",
		Version:      "v0.1.0",
		Author:       "Meroxa, Inc.",
		SourceParams: map[string]sdk.Parameter{},
	}
}
