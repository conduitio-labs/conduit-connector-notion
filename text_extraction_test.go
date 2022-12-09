// Copyright © 2022 Meroxa, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notion

import (
	"encoding/json"
	"testing"

	notion "github.com/jomei/notionapi"
	"github.com/matryer/is"
)

func TestExtractText_Paragraph(t *testing.T) {
	is := is.New(t)

	blockString := `{
	  "object": "block",
	  "id": "64b74001-739d-4e29-a6e9-d81f79bb0877",
	  "type": "paragraph",
	  "created_time": "2022-12-09T16:52:00Z",
	  "last_edited_time": "2022-12-09T16:55:00Z",
	  "created_by": {
		"object": "user",
		"id": "9f0964c0-d4d5-4943-abf4-773ee8f86dbc"
	  },
	  "last_edited_by": {
		"object": "user",
		"id": "9f0964c0-d4d5-4943-abf4-773ee8f86dbc"
	  },
	  "paragraph": {
		"rich_text": [
		  {
			"type": "text",
			"text": {
			  "content": "A paragraph with a link to "
			},
			"annotations": {
			  "bold": false,
			  "italic": false,
			  "strikethrough": false,
			  "underline": false,
			  "code": false,
			  "color": "default"
			},
			"plain_text": "A paragraph with a link to "
		  },
		  {
			"type": "text",
			"text": {
			  "content": "Conduit’s website",
			  "link": {
				"url": "https://conduit.io"
			  }
			},
			"annotations": {
			  "bold": false,
			  "italic": false,
			  "strikethrough": false,
			  "underline": false,
			  "code": false,
			  "color": "default"
			},
			"plain_text": "Conduit’s website",
			"href": "https://conduit.io"
		  },
		  {
			"type": "text",
			"text": {
			  "content": "."
			},
			"annotations": {
			  "bold": false,
			  "italic": false,
			  "strikethrough": false,
			  "underline": false,
			  "code": false,
			  "color": "default"
			},
			"plain_text": "."
		  }
		],
		"color": "default"
	  }
	}`
	var b notion.ParagraphBlock
	err := json.Unmarshal([]byte(blockString), &b)
	is.NoErr(err)

	text, err := extractText(b)
	is.NoErr()
}
