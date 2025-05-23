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

package ckb

import (
	"reflect"
	"testing"

	"github.com/drouotsi/bleve/v2/analysis"
	"github.com/drouotsi/bleve/v2/registry"
)

func TestSoraniAnalyzer(t *testing.T) {
	tests := []struct {
		input  []byte
		output analysis.TokenStream
	}{
		// stop word removal
		{
			input: []byte("ئەم پیاوە"),
			output: analysis.TokenStream{
				&analysis.Token{
					Term:     []byte("پیاو"),
					Position: 2,
					Start:    7,
					End:      17,
				},
			},
		},
		{
			input: []byte("پیاوە"),
			output: analysis.TokenStream{
				&analysis.Token{
					Term:     []byte("پیاو"),
					Position: 1,
					Start:    0,
					End:      10,
				},
			},
		},
		{
			input: []byte("پیاو"),
			output: analysis.TokenStream{
				&analysis.Token{
					Term:     []byte("پیاو"),
					Position: 1,
					Start:    0,
					End:      8,
				},
			},
		},
	}

	cache := registry.NewCache()
	analyzer, err := cache.AnalyzerNamed(AnalyzerName)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		actual := analyzer.Analyze(test.input)
		if !reflect.DeepEqual(actual, test.output) {
			t.Errorf("expected %v, got %v", test.output, actual)
		}
	}
}
