//  Copyright (c) 2022 Couchbase, Inc.
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

package plain

import (
	"github.com/drouotsi/bleve/v2/registry"
	"github.com/drouotsi/bleve/v2/search/highlight"
)

const Name = "plain"

const defaultPlainHighlightBefore = "<start>"
const defaultPlainHighlightAfter = "<end>"

type FragmentFormatter struct {
	before string
	after  string
}

func NewFragmentFormatter(before, after string) *FragmentFormatter {
	return &FragmentFormatter{
		before: before,
		after:  after,
	}
}

func (a *FragmentFormatter) Format(f *highlight.Fragment, orderedTermLocations highlight.TermLocations) string {
	rv := ""
	curr := f.Start
	for _, termLocation := range orderedTermLocations {
		if termLocation == nil {
			continue
		}
		// make sure the array positions match
		if !termLocation.ArrayPositions.Equals(f.ArrayPositions) {
			continue
		}
		if termLocation.Start < curr {
			continue
		}
		if termLocation.End > f.End {
			break
		}
		// add the stuff before this location
		rv += string(f.Orig[curr:termLocation.Start])
		// start the highlight tag
		rv += a.before
		// add the term itself
		rv += string(f.Orig[termLocation.Start:termLocation.End])
		// end the highlight tag
		rv += a.after
		// update current
		curr = termLocation.End
	}
	// add any remaining text after the last token
	rv += string(f.Orig[curr:f.End])

	return rv
}

func Constructor(config map[string]interface{}, cache *registry.Cache) (highlight.FragmentFormatter, error) {
	before := defaultPlainHighlightBefore
	beforeVal, ok := config["before"].(string)
	if ok {
		before = beforeVal
	}
	after := defaultPlainHighlightAfter
	afterVal, ok := config["after"].(string)
	if ok {
		after = afterVal
	}
	return NewFragmentFormatter(before, after), nil
}

func init() {
	err := registry.RegisterFragmentFormatter(Name, Constructor)
	if err != nil {
		panic(err)
	}
}
