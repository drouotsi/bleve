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

package it

import (
	"reflect"
	"testing"

	"github.com/drouotsi/bleve/v2/analysis"
	"github.com/drouotsi/bleve/v2/registry"
)

func TestItalianElision(t *testing.T) {
	tests := []struct {
		input  analysis.TokenStream
		output analysis.TokenStream
	}{
		{
			input: analysis.TokenStream{
				&analysis.Token{
					Term: []byte("dell'Italia"),
				},
			},
			output: analysis.TokenStream{
				&analysis.Token{
					Term: []byte("Italia"),
				},
			},
		},
	}

	cache := registry.NewCache()
	elisionFilter, err := cache.TokenFilterNamed(ElisionName)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		actual := elisionFilter.Filter(test.input)
		if !reflect.DeepEqual(actual, test.output) {
			t.Errorf("expected %s, got %s", test.output[0].Term, actual[0].Term)
		}
	}
}
