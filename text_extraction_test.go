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
	"os"
	"testing"

	notion "github.com/conduitio-labs/notionapi"
	"github.com/matryer/is"
)

func TestExtractText(t *testing.T) {
	testCases := []struct {
		name  string
		parse func([]byte) (notion.Block, error)
		input string
		want  string
	}{
		{
			name: "Paragraph blocks",
			parse: func(bytes []byte) (notion.Block, error) {
				var b notion.ParagraphBlock
				err := json.Unmarshal(bytes, &b)
				return b, err
			},
			// Text in HTML: A paragraph with a link to <a href="https://conduit.io/">Conduit’s website</a>.
			input: "./test/paragraph-block.json",
			want:  "A paragraph with a link to Conduit’s website.",
		},
		{
			name: "Numbered item block",
			parse: func(bytes []byte) (notion.Block, error) {
				var b notion.NumberedListItemBlock
				err := json.Unmarshal(bytes, &b)
				return b, err
			},
			input: "./test/numbered-list-item-block.json",
			want:  "Numbered item 2",
		},
		{
			name: "Bookmark block",
			parse: func(bytes []byte) (notion.Block, error) {
				var b notion.BookmarkBlock
				err := json.Unmarshal(bytes, &b)
				return b, err
			},
			input: "./test/bookmark-block.json",
			want:  "https://meroxa.com Meroxa’s web-site",
		},
		{
			name: "Equation block",
			parse: func(bytes []byte) (notion.Block, error) {
				var b notion.EquationBlock
				err := json.Unmarshal(bytes, &b)
				return b, err
			},
			input: "./test/equation-block.json",
			want:  "|x| = 1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			bytes, err := os.ReadFile(tc.input)
			is.NoErr(err)

			block, err := tc.parse(bytes)
			is.NoErr(err)

			got, err := extractText(block)
			is.NoErr(err)
			is.Equal(tc.want, got)
		})
	}
}
