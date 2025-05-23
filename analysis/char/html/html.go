//  Copyright (c) 2014 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package html

import (
	"bytes"
	"regexp"

	"github.com/drouotsi/bleve/v2/analysis"
	"github.com/drouotsi/bleve/v2/registry"
)

const Name = "html"

var htmlCharFilterRegexp = regexp.MustCompile(`</?[!\w]+((\s+\w+(\s*=\s*(?:".*?"|'.*?'|[^'">\s]+))?)+\s*|\s*)/?>`)

type CharFilter struct {
	r           *regexp.Regexp
	replacement []byte
}

func New() *CharFilter {
	return &CharFilter{
		r:           htmlCharFilterRegexp,
		replacement: []byte(" "),
	}
}

func (s *CharFilter) Filter(input []byte) []byte {
	return s.r.ReplaceAllFunc(
		input, func(in []byte) []byte {
			return bytes.Repeat(s.replacement, len(in))
		})
}

func CharFilterConstructor(config map[string]interface{}, cache *registry.Cache) (analysis.CharFilter, error) {
	return New(), nil
}

func init() {
	err := registry.RegisterCharFilter(Name, CharFilterConstructor)
	if err != nil {
		panic(err)
	}
}
