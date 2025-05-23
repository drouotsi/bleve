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

package upsidedown

import (
	"context"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	index "github.com/blevesearch/bleve_index_api"
	"github.com/drouotsi/bleve/v2/analysis"
	"github.com/drouotsi/bleve/v2/analysis/analyzer/standard"
	regexpTokenizer "github.com/drouotsi/bleve/v2/analysis/tokenizer/regexp"
	"github.com/drouotsi/bleve/v2/document"
	"github.com/drouotsi/bleve/v2/index/upsidedown/store/boltdb"
	"github.com/drouotsi/bleve/v2/index/upsidedown/store/null"
	"github.com/drouotsi/bleve/v2/registry"
)

var testAnalyzer = &analysis.DefaultAnalyzer{
	Tokenizer: regexpTokenizer.NewRegexpTokenizer(regexp.MustCompile(`\w+`)),
}

func TestIndexOpenReopen(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}

	var expectedCount uint64
	reader, err := idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err := reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	// opening the database should have inserted a version
	expectedLength := uint64(1)
	rowCount, err := idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}

	// now close it
	err = idx.Close()
	if err != nil {
		t.Fatal(err)
	}

	idx, err = NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}

	// now close it
	err = idx.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndexInsert(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	var expectedCount uint64
	reader, err := idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err := reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	reader, err = idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err = reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	// should have 4 rows (1 for version, 1 for schema field, and 1 for single term, and 1 for the term count, and 1 for the back index entry)
	expectedLength := uint64(1 + 1 + 1 + 1 + 1)
	rowCount, err := idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}
}

func TestIndexInsertThenDelete(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	var expectedCount uint64
	reader, err := idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err := reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	doc2 := document.NewDocument("2")
	doc2.AddField(document.NewTextField("name", []uint64{}, []byte("test")))
	err = idx.Update(doc2)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	reader, err = idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err = reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = idx.Delete("1")
	if err != nil {
		t.Errorf("Error deleting entry from index: %v", err)
	}
	expectedCount--

	reader, err = idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err = reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = idx.Delete("2")
	if err != nil {
		t.Errorf("Error deleting entry from index: %v", err)
	}
	expectedCount--

	reader, err = idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err = reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	// should have 2 rows (1 for version, 1 for schema field, 1 for dictionary row garbage)
	expectedLength := uint64(1 + 1 + 1)
	rowCount, err := idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}
}

func TestIndexInsertThenUpdate(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	// this update should overwrite one term, and introduce one new one
	doc = document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithAnalyzer("name", []uint64{}, []byte("test fail"), testAnalyzer))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error deleting entry from index: %v", err)
	}

	// should have 2 rows (1 for version, 1 for schema field, and 2 for the two term, and 2 for the term counts, and 1 for the back index entry)
	expectedLength := uint64(1 + 1 + 2 + 2 + 1)
	rowCount, err := idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}

	// now do another update that should remove one of the terms
	doc = document.NewDocument("1")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("fail")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error deleting entry from index: %v", err)
	}

	// should have 2 rows (1 for version, 1 for schema field, and 1 for the remaining term, and 2 for the term diciontary, and 1 for the back index entry)
	expectedLength = uint64(1 + 1 + 1 + 2 + 1)
	rowCount, err = idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}
}

func TestIndexInsertMultiple(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}

	var expectedCount uint64

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	doc = document.NewDocument("2")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	// should have 4 rows (1 for version, 1 for schema field, and 2 for single term, and 1 for the term count, and 2 for the back index entries)
	expectedLength := uint64(1 + 1 + 2 + 1 + 2)
	rowCount, err := idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}

	// close, reopen and add one more to test that counting works correctly
	err = idx.Close()
	if err != nil {
		t.Fatal(err)
	}

	idx, err = NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Fatalf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	doc = document.NewDocument("3")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	reader, err := idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err := reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndexInsertWithStore(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	var expectedCount uint64
	reader, err := idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err := reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test"), index.IndexField|index.StoreField))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	reader, err = idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err = reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	// should have 6 rows (1 for version, 1 for schema field, and 1 for single term, and 1 for the stored field and 1 for the term count, and 1 for the back index entry)
	expectedLength := uint64(1 + 1 + 1 + 1 + 1 + 1)
	rowCount, err := idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}

	indexReader, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := indexReader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	storedDocInt, err := indexReader.Document("1")
	if err != nil {
		t.Error(err)
	}
	storedDoc := storedDocInt.(*document.Document)

	if len(storedDoc.Fields) != 1 {
		t.Errorf("expected 1 stored field, got %d", len(storedDoc.Fields))
	}
	textField, ok := storedDoc.Fields[0].(*document.TextField)
	if !ok {
		t.Errorf("expected text field")
	}
	if string(textField.Value()) != "test" {
		t.Errorf("expected field content 'test', got '%s'", string(textField.Value()))
	}
}

func TestIndexInternalCRUD(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	indexReader, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}

	// get something that doesn't exist yet
	val, err := indexReader.GetInternal([]byte("key"))
	if err != nil {
		t.Error(err)
	}
	if val != nil {
		t.Errorf("expected nil, got %s", val)
	}

	err = indexReader.Close()
	if err != nil {
		t.Fatal(err)
	}

	// set
	err = idx.SetInternal([]byte("key"), []byte("abc"))
	if err != nil {
		t.Error(err)
	}

	indexReader2, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}

	// get
	val, err = indexReader2.GetInternal([]byte("key"))
	if err != nil {
		t.Error(err)
	}
	if string(val) != "abc" {
		t.Errorf("expected %s, got '%s'", "abc", val)
	}

	err = indexReader2.Close()
	if err != nil {
		t.Fatal(err)
	}

	// delete
	err = idx.DeleteInternal([]byte("key"))
	if err != nil {
		t.Error(err)
	}

	indexReader3, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}

	// get again
	val, err = indexReader3.GetInternal([]byte("key"))
	if err != nil {
		t.Error(err)
	}
	if val != nil {
		t.Errorf("expected nil, got %s", val)
	}

	err = indexReader3.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndexBatch(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	var expectedCount uint64

	// first create 2 docs the old fashioned way
	doc := document.NewDocument("1")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	doc = document.NewDocument("2")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test2")))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	// now create a batch which does 3 things
	// insert new doc
	// update existing doc
	// delete existing doc
	// net document count change 0

	batch := index.NewBatch()
	doc = document.NewDocument("3")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test3")))
	batch.Update(doc)
	doc = document.NewDocument("2")
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test2updated")))
	batch.Update(doc)
	batch.Delete("1")

	err = idx.Batch(batch)
	if err != nil {
		t.Error(err)
	}

	indexReader, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := indexReader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	docCount, err := indexReader.DocCount()
	if err != nil {
		t.Fatal(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}

	docIDReader, err := indexReader.DocIDReaderAll()
	if err != nil {
		t.Error(err)
	}
	var docIds []index.IndexInternalID
	docID, err := docIDReader.Next()
	for docID != nil && err == nil {
		docIds = append(docIds, docID)
		docID, err = docIDReader.Next()
	}
	if err != nil {
		t.Error(err)
	}
	expectedDocIds := []index.IndexInternalID{index.IndexInternalID("2"), index.IndexInternalID("3")}
	if !reflect.DeepEqual(docIds, expectedDocIds) {
		t.Errorf("expected ids: %v, got ids: %v", expectedDocIds, docIds)
	}
}

func TestIndexInsertUpdateDeleteWithMultipleTypesStored(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	var expectedCount uint64
	reader, err := idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err := reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test"), index.IndexField|index.StoreField))
	doc.AddField(document.NewNumericFieldWithIndexingOptions("age", []uint64{}, 35.99, index.IndexField|index.StoreField))
	df, err := document.NewDateTimeFieldWithIndexingOptions("unixEpoch", []uint64{}, time.Unix(0, 0), time.RFC3339, index.IndexField|index.StoreField)
	if err != nil {
		t.Error(err)
	}
	doc.AddField(df)
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}
	expectedCount++

	reader, err = idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err = reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	// should have 72 rows
	// 1 for version
	// 3 for schema fields
	// 1 for text term
	// 16 for numeric terms
	// 16 for date terms
	// 3 for the stored field
	// 1 for the text term count
	// 16 for numeric term counts
	// 16 for date term counts
	// 1 for the back index entry
	expectedLength := uint64(1 + 3 + 1 + (64 / document.DefaultPrecisionStep) + (64 / document.DefaultPrecisionStep) + 3 + 1 + (64 / document.DefaultPrecisionStep) + (64 / document.DefaultPrecisionStep) + 1)
	rowCount, err := idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}

	indexReader, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}

	storedDocInt, err := indexReader.Document("1")
	if err != nil {
		t.Error(err)
	}
	storedDoc := storedDocInt.(*document.Document)

	err = indexReader.Close()
	if err != nil {
		t.Error(err)
	}

	if len(storedDoc.Fields) != 3 {
		t.Errorf("expected 3 stored field, got %d", len(storedDoc.Fields))
	}
	textField, ok := storedDoc.Fields[0].(*document.TextField)
	if !ok {
		t.Errorf("expected text field")
	}
	if string(textField.Value()) != "test" {
		t.Errorf("expected field content 'test', got '%s'", string(textField.Value()))
	}
	numField, ok := storedDoc.Fields[1].(*document.NumericField)
	if !ok {
		t.Errorf("expected numeric field")
	}
	numFieldNumer, err := numField.Number()
	if err != nil {
		t.Error(err)
	} else {
		if numFieldNumer != 35.99 {
			t.Errorf("expected numeric value 35.99, got %f", numFieldNumer)
		}
	}
	dateField, ok := storedDoc.Fields[2].(*document.DateTimeField)
	if !ok {
		t.Errorf("expected date field")
	}
	dateFieldDate, _, err := dateField.DateTime()
	if err != nil {
		t.Error(err)
	} else {
		if dateFieldDate != time.Unix(0, 0).UTC() {
			t.Errorf("expected date value unix epoch, got %v", dateFieldDate)
		}
	}

	// now update the document, but omit one of the fields
	doc = document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("testup"), index.IndexField|index.StoreField))
	doc.AddField(document.NewNumericFieldWithIndexingOptions("age", []uint64{}, 36.99, index.IndexField|index.StoreField))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	indexReader2, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}

	// expected doc count shouldn't have changed
	docCount, err = indexReader2.DocCount()
	if err != nil {
		t.Fatal(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}

	// should only get 2 fields back now though
	storedDocInt, err = indexReader2.Document("1")
	if err != nil {
		t.Error(err)
	}
	storedDoc = storedDocInt.(*document.Document)

	err = indexReader2.Close()
	if err != nil {
		t.Error(err)
	}

	if len(storedDoc.Fields) != 2 {
		t.Errorf("expected 3 stored field, got %d", len(storedDoc.Fields))
	}
	textField, ok = storedDoc.Fields[0].(*document.TextField)
	if !ok {
		t.Errorf("expected text field")
	}
	if string(textField.Value()) != "testup" {
		t.Errorf("expected field content 'testup', got '%s'", string(textField.Value()))
	}
	numField, ok = storedDoc.Fields[1].(*document.NumericField)
	if !ok {
		t.Errorf("expected numeric field")
	}
	numFieldNumer, err = numField.Number()
	if err != nil {
		t.Error(err)
	} else {
		if numFieldNumer != 36.99 {
			t.Errorf("expected numeric value 36.99, got %f", numFieldNumer)
		}
	}

	// now delete the document
	err = idx.Delete("1")
	if err != nil {
		t.Errorf("Error deleting entry from index: %v", err)
	}

	expectedCount--

	// expected doc count shouldn't have changed
	reader, err = idx.Reader()
	if err != nil {
		t.Fatal(err)
	}
	docCount, err = reader.DocCount()
	if err != nil {
		t.Error(err)
	}
	if docCount != expectedCount {
		t.Errorf("Expected document count to be %d got %d", expectedCount, docCount)
	}
	err = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndexInsertFields(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test"), index.IndexField|index.StoreField))
	doc.AddField(document.NewNumericFieldWithIndexingOptions("age", []uint64{}, 35.99, index.IndexField|index.StoreField))
	dateField, err := document.NewDateTimeFieldWithIndexingOptions("unixEpoch", []uint64{}, time.Unix(0, 0), time.RFC3339, index.IndexField|index.StoreField)
	if err != nil {
		t.Error(err)
	}
	doc.AddField(dateField)
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	indexReader, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := indexReader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	fields, err := indexReader.Fields()
	if err != nil {
		t.Error(err)
	} else {
		expectedFields := []string{"name", "age", "unixEpoch"}
		if !reflect.DeepEqual(fields, expectedFields) {
			t.Errorf("expected fields: %v, got %v", expectedFields, fields)
		}
	}
}

func TestIndexUpdateComposites(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test"), index.IndexField|index.StoreField))
	doc.AddField(document.NewTextFieldWithIndexingOptions("title", []uint64{}, []byte("mister"), index.IndexField|index.StoreField))
	doc.AddField(document.NewCompositeFieldWithIndexingOptions("_all", true, nil, nil, index.IndexField))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	// should have 72 rows
	// 1 for version
	// 3 for schema fields
	// 4 for text term
	// 2 for the stored field
	// 4 for the text term count
	// 1 for the back index entry
	expectedLength := uint64(1 + 3 + 4 + 2 + 4 + 1)
	rowCount, err := idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}

	// now lets update it
	doc = document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("testupdated"), index.IndexField|index.StoreField))
	doc.AddField(document.NewTextFieldWithIndexingOptions("title", []uint64{}, []byte("misterupdated"), index.IndexField|index.StoreField))
	doc.AddField(document.NewCompositeFieldWithIndexingOptions("_all", true, nil, nil, index.IndexField))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	indexReader, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := indexReader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// make sure new values are in index
	storedDocInt, err := indexReader.Document("1")
	if err != nil {
		t.Error(err)
	}
	storedDoc := storedDocInt.(*document.Document)
	if len(storedDoc.Fields) != 2 {
		t.Errorf("expected 2 stored field, got %d", len(storedDoc.Fields))
	}
	textField, ok := storedDoc.Fields[0].(*document.TextField)
	if !ok {
		t.Errorf("expected text field")
	}
	if string(textField.Value()) != "testupdated" {
		t.Errorf("expected field content 'test', got '%s'", string(textField.Value()))
	}

	// should have the same row count as before, plus 4 term dictionary garbage rows
	expectedLength += 4
	rowCount, err = idx.(*UpsideDownCouch).rowCount()
	if err != nil {
		t.Error(err)
	}
	if rowCount != expectedLength {
		t.Errorf("expected %d rows, got: %d", expectedLength, rowCount)
	}
}

func TestIndexFieldsMisc(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test"), index.IndexField|index.StoreField))
	doc.AddField(document.NewTextFieldWithIndexingOptions("title", []uint64{}, []byte("mister"), index.IndexField|index.StoreField))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	fieldName1 := idx.(*UpsideDownCouch).fieldCache.FieldIndexed(0)
	if fieldName1 != "name" {
		t.Errorf("expected field named 'name', got '%s'", fieldName1)
	}
	fieldName2 := idx.(*UpsideDownCouch).fieldCache.FieldIndexed(1)
	if fieldName2 != "title" {
		t.Errorf("expected field named 'title', got '%s'", fieldName2)
	}
	fieldName3 := idx.(*UpsideDownCouch).fieldCache.FieldIndexed(2)
	if fieldName3 != "" {
		t.Errorf("expected field named '', got '%s'", fieldName3)
	}
}

func TestIndexTermReaderCompositeFields(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test"), index.IndexField|index.StoreField|index.IncludeTermVectors))
	doc.AddField(document.NewTextFieldWithIndexingOptions("title", []uint64{}, []byte("mister"), index.IndexField|index.StoreField|index.IncludeTermVectors))
	doc.AddField(document.NewCompositeFieldWithIndexingOptions("_all", true, nil, nil, index.IndexField|index.IncludeTermVectors))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	indexReader, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := indexReader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	termFieldReader, err := indexReader.TermFieldReader(context.TODO(), []byte("mister"), "_all", true, true, true)
	if err != nil {
		t.Error(err)
	}

	tfd, err := termFieldReader.Next(nil)
	for tfd != nil && err == nil {
		if !tfd.ID.Equals(index.IndexInternalID("1")) {
			t.Errorf("expected to find document id 1")
		}
		tfd, err = termFieldReader.Next(nil)
	}
	if err != nil {
		t.Error(err)
	}
}

func TestIndexDocValueReader(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	doc := document.NewDocument("1")
	doc.AddField(document.NewTextFieldWithIndexingOptions("name", []uint64{}, []byte("test"), index.IndexField|index.StoreField|index.IncludeTermVectors))
	doc.AddField(document.NewTextFieldWithIndexingOptions("title", []uint64{}, []byte("mister"), index.IndexField|index.StoreField|index.IncludeTermVectors))
	err = idx.Update(doc)
	if err != nil {
		t.Errorf("Error updating index: %v", err)
	}

	indexReader, err := idx.Reader()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := indexReader.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	actualFieldTerms := make(fieldTerms)

	dvr, err := indexReader.DocValueReader([]string{"name", "title"})
	if err != nil {
		t.Error(err)
	}

	err = dvr.VisitDocValues(index.IndexInternalID("1"), func(field string, term []byte) {
		actualFieldTerms[field] = append(actualFieldTerms[field], string(term))
	})
	if err != nil {
		t.Error(err)
	}
	expectedFieldTerms := fieldTerms{
		"name":  []string{"test"},
		"title": []string{"mister"},
	}
	if !reflect.DeepEqual(actualFieldTerms, expectedFieldTerms) {
		t.Errorf("expected field terms: %#v, got: %#v", expectedFieldTerms, actualFieldTerms)
	}
}

func BenchmarkBatch(b *testing.B) {
	cache := registry.NewCache()
	analyzer, err := cache.AnalyzerNamed(standard.Name)
	if err != nil {
		b.Fatal(err)
	}

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(null.Name, nil, analysisQueue)
	if err != nil {
		b.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		b.Fatal(err)
	}

	batch := index.NewBatch()
	for i := 0; i < 100; i++ {
		d := document.NewDocument(strconv.Itoa(i))
		f := document.NewTextFieldWithAnalyzer("desc", nil, bleveWikiArticle1K, analyzer)
		d.AddField(f)
		batch.Update(d)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err = idx.Batch(batch)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestConcurrentUpdate(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// do some concurrent updates
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			doc := document.NewDocument("1")
			doc.AddField(document.NewTextFieldWithIndexingOptions(strconv.Itoa(i), []uint64{}, []byte(strconv.Itoa(i)), index.StoreField))
			err := idx.Update(doc)
			if err != nil {
				t.Errorf("Error updating index: %v", err)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	// now load the name field and see what we get
	r, err := idx.Reader()
	if err != nil {
		log.Fatal(err)
	}

	docInt, err := r.Document("1")
	if err != nil {
		log.Fatal(err)
	}
	doc := docInt.(*document.Document)

	if len(doc.Fields) > 1 {
		t.Errorf("expected single field, found %d", len(doc.Fields))
	}

	err = r.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestLargeField(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	var largeFieldValue []byte
	for len(largeFieldValue) < RowBufferSize {
		largeFieldValue = append(largeFieldValue, bleveWikiArticle1K...)
	}
	t.Logf("large field size: %d", len(largeFieldValue))

	d := document.NewDocument("large")
	f := document.NewTextFieldWithIndexingOptions("desc", nil, largeFieldValue, index.IndexField|index.StoreField)
	d.AddField(f)

	err = idx.Update(d)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndexBatchPersistedCallbackWithErrorUpsideDown(t *testing.T) {
	defer func() {
		err := DestroyTest()
		if err != nil {
			t.Fatal(err)
		}
	}()

	analysisQueue := index.NewAnalysisQueue(1)
	idx, err := NewUpsideDownCouch(boltdb.Name, boltTestConfig, analysisQueue)
	if err != nil {
		t.Fatal(err)
	}
	err = idx.Open()
	if err != nil {
		t.Errorf("error opening index: %v", err)
	}
	defer func() {
		err := idx.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	var callbackExecuted bool
	batch := index.NewBatch()
	batch.SetPersistedCallback(func(e error) {
		callbackExecuted = true
	})
	// By using a really large ID, we ensure that the batch will fail,
	// because the key generated by upside down will be too large for BoltDB
	reallyBigId := strings.Repeat("x", 32768+1)
	doc := document.NewDocument(reallyBigId)
	doc.AddField(document.NewTextField("name", []uint64{}, []byte("test3")))
	batch.Update(doc)

	_ = idx.Batch(batch)
	// don't fail on this error, that isn't what we're testing

	if !callbackExecuted {
		t.Fatal("expected callback to fire, it did not")
	}
}

// fieldTerms contains the terms used by a document, keyed by field
type fieldTerms map[string][]string

// FieldsNotYetCached returns a list of fields not yet cached out of a larger list of fields
func (f fieldTerms) FieldsNotYetCached(fields []string) []string {
	rv := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, ok := f[field]; !ok {
			rv = append(rv, field)
		}
	}
	return rv
}

// Merge will combine two fieldTerms
// it assumes that the terms lists are complete (thus do not need to be merged)
// field terms from the other list always replace the ones in the receiver
func (f fieldTerms) Merge(other fieldTerms) {
	for field, terms := range other {
		f[field] = terms
	}
}
