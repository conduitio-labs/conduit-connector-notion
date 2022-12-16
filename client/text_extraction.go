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

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	notion "github.com/conduitio-labs/notionapi"
	"github.com/tidwall/gjson"
)

type extractor func(notion.Block) (string, error)

var errNoExtractor = errors.New("no extractor")

// getJSONPath returns the value found at the specified path *within*
// the entity wrapped by this block.
// For example, if this is a paragraph block:
//
//	{
//	 "type": "paragraph",
//	 //...other keys excluded
//	 "paragraph": {
//	 //...other keys excluded
//	 }
//	}
//
// then the function will be looking for `path` in `paragraph`.
func getJSONPath(block notion.Block, path string) (gjson.Result, error) {
	bytes, err := json.Marshal(block)
	if err != nil {
		return gjson.Result{}, fmt.Errorf("failed marshalling into JSON: %w", err)
	}
	return gjson.Get(string(bytes), block.GetType().String()+path), nil
}

var titleExtractor = extractor(func(block notion.Block) (string, error) {
	title, err := getJSONPath(block, ".title")
	if err != nil {
		return "", err
	}
	return title.Str, nil
})

var plainTextExtractor = extractor(func(block notion.Block) (string, error) {
	richTexts, err := getJSONPath(block, ".rich_text")
	if err != nil {
		return "", err
	}
	var result string
	for _, rt := range richTexts.Array() {
		result += rt.Get("plain_text").Str
	}
	return result, nil
})

var urlExtractor = extractor(func(block notion.Block) (string, error) {
	url, err := getJSONPath(block, ".url")
	if err != nil {
		return "", err
	}

	elems := []string{url.Str}

	captions, err := getJSONPath(block, ".caption")
	if err != nil {
		return "", err
	}
	for _, rt := range captions.Array() {
		elems = append(elems, rt.Get("plain_text").Str)
	}

	return strings.Join(elems, " "), nil
})

var fileExtractor = extractor(func(block notion.Block) (string, error) {
	notionFileURL, err := getJSONPath(block, ".file.url")
	if err != nil {
		return "", fmt.Errorf("failed getting JSON path %v: %w", ".file.url", err)
	}

	externalURL, err := getJSONPath(block, ".external.url")
	if err != nil {
		return "", err
	}
	return strings.Join([]string{notionFileURL.Str, externalURL.Str}, " "), nil
})

var equationExtractor = extractor(func(block notion.Block) (string, error) {
	expression, err := getJSONPath(block, ".expression")
	if err != nil {
		return "", err
	}
	return expression.Str, nil
})

var extractors = map[string]extractor{
	"child_page":     titleExtractor,
	"child_database": titleExtractor,

	"equation": equationExtractor,

	"file":  fileExtractor,
	"image": fileExtractor,
	"video": fileExtractor,
	"pdf":   fileExtractor,

	"paragraph":          plainTextExtractor,
	"heading_1":          plainTextExtractor,
	"heading_2":          plainTextExtractor,
	"heading_3":          plainTextExtractor,
	"callout":            plainTextExtractor,
	"quite":              plainTextExtractor,
	"bulleted_list_item": plainTextExtractor,
	"numbered_list_item": plainTextExtractor,
	"to_do":              plainTextExtractor,
	"toggle":             plainTextExtractor,
	"code":               plainTextExtractor,
	"template":           plainTextExtractor,

	"embed":        urlExtractor,
	"bookmark":     urlExtractor,
	"link_preview": urlExtractor,
}

func extractText(b notion.Block) (string, error) {
	e, ok := extractors[b.GetType().String()]
	if !ok {
		return "", fmt.Errorf("block type %v: %w", b.GetType().String(), errNoExtractor)
	}
	return e(b)
}
