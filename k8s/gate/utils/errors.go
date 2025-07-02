//
// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils // revive:disable:var-naming

import (
	"errors"
)

type Errors []error

func (e *Errors) Add(errors ...error) {
	for _, err := range errors {
		if err != nil {
			*e = append(*e, err)
		}
	}
}

func (e *Errors) Result() error {
	var result string
	for _, err := range *e {
		result += err.Error() + "\n"
	}
	if result == "" {
		return nil
	}
	return errors.New(result)
}

func (e *Errors) AddErrors(errors Errors) {
	for _, err := range errors {
		e.Add((err))
	}
}
