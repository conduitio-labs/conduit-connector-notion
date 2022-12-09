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

package notion_test

import (
	"context"
	"strings"
	"testing"

	notion "github.com/conduitio-labs/conduit-connector-notion"
)

func TestConfigureSource_FailsWhenConfigEmpty(t *testing.T) {
	con := notion.Source{}
	err := con.Configure(context.Background(), make(map[string]string))
	if err == nil {
		t.Error("expected error for missing config params")
	}

	if strings.HasPrefix(err.Error(), "config is invalid:") {
		t.Errorf("expected error to be about missing config, got %v", err)
	}
}

func TestTeardownSource_NoOpen(t *testing.T) {
	con := notion.NewSource()
	err := con.Teardown(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
