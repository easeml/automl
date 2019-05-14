package schema

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"testing"
)

// Here we assume all test will be ran from the v0.1/go directory.
const relTestPath = "../../../test-examples"
const schemaValPositivePath = "schema/validate/positive"
const schemaValNegativePath = "schema/validate/negative"
const schemaMatchPositivePath = "schema/match/positive"
const schemaMatchNegativePath = "schema/match/negative"

func TestSchemaValidate(t *testing.T) {

	// Load positive test examples.
	files, err := ioutil.ReadDir(path.Join(relTestPath, schemaValPositivePath))
	if err != nil {
		log.Fatal(err)
	}
	for i := range files {
		var testName = path.Join(schemaValPositivePath, files[i].Name())
		var exampleFile = path.Join(relTestPath, schemaValPositivePath, files[i].Name())
		t.Run(testName, testSchemaValidate(exampleFile, true))
	}

	// Load negative test examples.
	files, err = ioutil.ReadDir(path.Join(relTestPath, schemaValNegativePath))
	if err != nil {
		log.Fatal(err)
	}
	for i := range files {
		var testName = path.Join(schemaValNegativePath, files[i].Name())
		var exampleFile = path.Join(relTestPath, schemaValNegativePath, files[i].Name())
		t.Run(testName, testSchemaValidate(exampleFile, false))
	}

}

func testSchemaValidate(exampleFile string, positiveExample bool) func(*testing.T) {
	return func(t *testing.T) {
		// Load the file.
		srcData, err := ioutil.ReadFile(exampleFile)
		if err != nil {
			panic(err)
		}

		// Unmarshal JSON into a map.
		var src map[string]interface{}
		err = json.Unmarshal(srcData, &src)
		if err != nil {
			panic(err)
		}

		// Load the schema.
		_, err = Load(src)
		if positiveExample && err != nil {
			schErr, ok := err.(Error)
			if ok == false {
				t.Error("Resulting error should be a schema.Error instance.")
			}
			msg := schErr.Error()
			pth := schErr.Path()

			t.Errorf("Found error '%s' on path '%s'.", msg, pth)

		} else if !positiveExample {

			if err == nil {
				t.Error("Expected an error but got none.")
			} else {
				_, ok := err.(Error)
				if ok == false {
					t.Error("Resulting error should be a schema.Error instance.")
				}
			}
		}
	}
}

func TestSchemaMatch(t *testing.T) {

	// Load positive test examples.
	files, err := ioutil.ReadDir(path.Join(relTestPath, schemaMatchPositivePath))
	if err != nil {
		log.Fatal(err)
	}
	for i := range files {
		var fileName = files[i].Name()
		if strings.HasSuffix(fileName, "-src.json") {

			fileName = strings.TrimSuffix(fileName, "-src.json")
			var fileNameSrc = fileName + "-src.json"
			var fileNameDst = fileName + "-dst.json"

			var testName = path.Join(schemaMatchPositivePath, fileName)
			var exampleFileSrc = path.Join(relTestPath, schemaMatchPositivePath, fileNameSrc)
			var exampleFileDst = path.Join(relTestPath, schemaMatchPositivePath, fileNameDst)
			t.Run(testName, testSchemaMatch(exampleFileSrc, exampleFileDst, true))
		}
	}

	// Load negative test examples.
	files, err = ioutil.ReadDir(path.Join(relTestPath, schemaMatchNegativePath))
	if err != nil {
		log.Fatal(err)
	}
	for i := range files {
		var fileName = files[i].Name()
		if strings.HasSuffix(fileName, "-src.json") {

			fileName = strings.TrimSuffix(fileName, "-src.json")
			var fileNameSrc = fileName + "-src.json"
			var fileNameDst = fileName + "-dst.json"

			var testName = path.Join(schemaMatchNegativePath, fileName)
			var exampleFileSrc = path.Join(relTestPath, schemaMatchNegativePath, fileNameSrc)
			var exampleFileDst = path.Join(relTestPath, schemaMatchNegativePath, fileNameDst)
			t.Run(testName, testSchemaMatch(exampleFileSrc, exampleFileDst, false))
		}
	}

}

func testSchemaMatch(exampleFileSrc, exampleFileDst string, positiveExample bool) func(*testing.T) {
	return func(t *testing.T) {
		// Load the file.
		srcData, err := ioutil.ReadFile(exampleFileSrc)
		if err != nil {
			panic(err)
		}
		dstData, err := ioutil.ReadFile(exampleFileDst)
		if err != nil {
			panic(err)
		}

		// Unmarshal JSON into a map.
		var src map[string]interface{}
		err = json.Unmarshal(srcData, &src)
		if err != nil {
			panic(err)
		}
		var dst map[string]interface{}
		err = json.Unmarshal(dstData, &dst)
		if err != nil {
			panic(err)
		}

		// Load the schema.
		srcSchema, err := Load(src)
		if err != nil {
			panic(err)
		}
		dstSchema, err := Load(dst)
		if err != nil {
			panic(err)
		}
		match, schMatching := dstSchema.Match(srcSchema, true)

		if positiveExample {
			if match == false {
				t.Error("Schema match failed.")
			}

			if schMatching == nil {
				t.Error("Schema matching not returned.")
			}

		} else if !positiveExample && (match == true || schMatching != nil) {

			t.Error("Expected a failure but got a match.")

		}
	}
}
