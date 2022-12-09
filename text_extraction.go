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
	"fmt"
	"strings"

	notion "github.com/jomei/notionapi"
	"github.com/tidwall/gjson"
)

type extractor func(notion.Block) (string, error)

var titleExtractor = extractor(func(block notion.Block) (string, error) {
	bytes, err := json.Marshal(block)
	if err != nil {
		return "", fmt.Errorf("failed marshalling into JSON: %w", err)
	}
	return gjson.Get(string(bytes), block.GetType().String()+".title").Str, nil
})

var plainTextExtractor = extractor(func(block notion.Block) (string, error) {
	bytes, err := json.Marshal(block)
	if err != nil {
		return "", fmt.Errorf("failed marshalling into JSON: %w", err)
	}
	richTexts := gjson.Get(string(bytes), block.GetType().String()+".rich_text")
	var result string
	for _, rt := range richTexts.Array() {
		result += rt.Get("plain_text").Str
		result += " "
	}
	return result, nil
})

var urlExtractor = extractor(func(block notion.Block) (string, error) {
	bytes, err := json.Marshal(block)
	if err != nil {
		return "", fmt.Errorf("failed marshalling into JSON: %w", err)
	}

	var result string
	url := gjson.Get(string(bytes), block.GetType().String()+".url")
	result += url.Str
	result += " "

	captions := gjson.Get(string(bytes), block.GetType().String()+".caption")
	for _, rt := range captions.Array() {
		result += rt.Get("plain_text").Str
		result += " "
	}
	return result, nil
})

var fileExtractor = extractor(func(block notion.Block) (string, error) {
	bytes, err := json.Marshal(block)
	if err != nil {
		return "", fmt.Errorf("failed marshalling into JSON: %w", err)
	}
	notionFileURL := gjson.Get(string(bytes), block.GetType().String()+".file.url").Str
	externalURL := gjson.Get(string(bytes), block.GetType().String()+".external.url").Str
	return strings.Join([]string{notionFileURL, externalURL}, " "), nil
})

var equationExtractor = extractor(func(block notion.Block) (string, error) {
	bytes, err := json.Marshal(block)
	if err != nil {
		return "", fmt.Errorf("failed marshalling into JSON: %w", err)
	}
	return gjson.Get(string(bytes), block.GetType().String()+".expression").Str, nil
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
