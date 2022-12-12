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
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	Token        = "token"
	PollInterval = "pollInterval"
)

var Required = []string{Token}

var (
	ErrEmptyConfig          = errors.New("missing or empty config")
	ErrRequiredParamMissing = errors.New("required parameter missing")
)

type Config struct {
	// token is the authorization token to be used
	// in requests to the Notion API
	token string
	// pollInterval is the interval between subsequents polls
	// in which we check for changes in Notion.
	// Given that the last_edited_field is used to detect changes
	// in Notion, and that it's precision is in minutes,
	// the poll interval must not be shorter than a minute,
	// to avoid reading duplicates.
	pollInterval time.Duration
}

func ParseConfig(cfg map[string]string) (Config, error) {
	err := checkRequired(cfg)
	if err != nil {
		return Config{}, err
	}
	// set defaults
	parsed := Config{
		pollInterval: time.Minute,
	}
	parsed.token = cfg[Token]

	if t, ok := cfg[PollInterval]; ok {
		pi, err := time.ParseDuration(t)
		if err != nil {
			return Config{}, fmt.Errorf("cannot parse poll interval %q: %w", t, err)
		}
		if pi < time.Minute {
			return Config{}, fmt.Errorf("poll interval must not be shorter than a minute (provided: %v)", pi)
		}
		parsed.pollInterval = pi
	}
	return parsed, nil
}

func checkRequired(cfg map[string]string) error {
	var missing []string
	for _, r := range Required {
		if strings.Trim(cfg[r], " ") == "" {
			missing = append(missing, r)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("params %v: %w", missing, ErrRequiredParamMissing)
	}
	return nil
}
