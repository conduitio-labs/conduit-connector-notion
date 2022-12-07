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
)

const (
	Token = "token"
)

var Required = []string{Token}

var (
	ErrEmptyConfig          = errors.New("missing or empty config")
	ErrRequiredParamMissing = errors.New("required parameter missing")
)

type Config struct {
	token string
}

func ParseConfig(cfg map[string]string) (Config, error) {
	err := checkEmpty(cfg)
	if err != nil {
		return Config{}, err
	}
	err = checkRequired(cfg)
	if err != nil {
		return Config{}, err
	}
	return Config{token: cfg[Token]}, nil
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

func checkEmpty(cfg map[string]string) error {
	if len(cfg) == 0 {
		return ErrEmptyConfig
	}
	return nil
}
