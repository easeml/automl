package dataset

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"testing"

	sch "github.com/ds3lab/easeml/schema/go/easemlschema/schema"
)

// Here we assume all test will be ran from the v0.1/go directory.
const relTestPath = "../../../test-examples"
const datasetInferPositivePath = "dataset/infer/positive"
const datasetInferNegativePath = "dataset/infer/negative"
const datasetGenPositivePath = "dataset/generate/positive"

func TestDatasetInfer(t *testing.T) {

	// Load positive test examples.
	files, err := ioutil.ReadDir(path.Join(relTestPath, datasetInferPositivePath))
	if err != nil {
		log.Fatal(err)
	}
	for i := range files {
		if files[i].IsDir() {
			var dirName = files[i].Name()
			var fileName = dirName + ".json"
			var testName = path.Join(datasetInferPositivePath, dirName)
			var exampleFile = path.Join(relTestPath, datasetInferPositivePath, fileName)
			var exampleDir = path.Join(relTestPath, datasetInferPositivePath, dirName)

			t.Run(testName, testDatasetInfer(exampleFile, exampleDir, true))
		}
	}

	// Load negative test examples.
	files, err = ioutil.ReadDir(path.Join(relTestPath, datasetInferNegativePath))
	if err != nil {
		log.Fatal(err)
	}
	for i := range files {
		if files[i].IsDir() {
			var dirName = files[i].Name()
			var testName = path.Join(datasetInferNegativePath, dirName)
			var exampleDir = path.Join(relTestPath, datasetInferNegativePath, dirName)

			t.Run(testName, testDatasetInfer("", exampleDir, false))
		}
	}

}

func testDatasetInfer(exampleFile string, exampleDir string, positiveExample bool) func(*testing.T) {
	return func(t *testing.T) {

		// Load the dataset.
		dataset, err := Load(exampleDir, true, DefaultOpener{})
		if err != nil {
			if positiveExample {
				t.Error("Dataset load failed.")
			}
			return
		}

		// Infer the schema.
		srcSchema, err := dataset.InferSchema()

		if positiveExample {
			if err != nil {
				dsErr, ok := err.(Error)
				if ok == false {
					t.Error("Resulting error should be a dataser.Error instance.")
				}
				msg := dsErr.Error()
				pth := dsErr.Path()

				t.Errorf("Found error '%s' on path '%s'.", msg, pth)
				return
			}

			// Load the file.
			dstData, err := ioutil.ReadFile(exampleFile)
			if err != nil {
				panic(err)
			}

			// Unmarshal JSON into a map.
			var dst map[string]interface{}
			err = json.Unmarshal(dstData, &dst)
			if err != nil {
				panic(err)
			}

			// Load the schema.
			dstSchema, err := sch.Load(dst)
			if err != nil {
				panic(err)
			}

			// Make sure the inferred dataset matches the schema.
			match, _ := dstSchema.Match(srcSchema, false)

			if match == false {
				t.Error("Schema match failed.")
				return
			}

		} else {

			if err == nil {
				t.Error("Expected an error but got none.")
			} else {
				_, ok := err.(Error)
				if ok == false {
					t.Error("Resulting error should be a dataser.Error instance.")
				}
			}
		}
	}
}

func TestDatasetGenerate(t *testing.T) {

	// Load positive test examples.
	files, err := ioutil.ReadDir(path.Join(relTestPath, datasetGenPositivePath))
	if err != nil {
		log.Fatal(err)
	}
	for i := range files {
		var fileName = files[i].Name()
		var testName = path.Join(datasetGenPositivePath, strings.TrimSuffix(fileName, ".json"))
		var exampleFile = path.Join(relTestPath, datasetGenPositivePath, fileName)

		t.Run(testName, testDatasetGenerate(exampleFile))
	}

}

func testDatasetGenerate(exampleFile string) func(*testing.T) {
	return func(t *testing.T) {

		// Load the file.
		dstData, err := ioutil.ReadFile(exampleFile)
		if err != nil {
			panic(err)
		}

		// Unmarshal JSON into a map.
		var dst map[string]interface{}
		err = json.Unmarshal(dstData, &dst)
		if err != nil {
			panic(err)
		}

		// Load the schema.
		dstSchema, err := sch.Load(dst)
		if err != nil {
			panic(err)
		}

		// Generate sample names.
		sampleNames := make([]string, 10)
		for i := range sampleNames {
			sampleNames[i] = RandomString(10, "")
		}

		// Generate random dataset from the schema.
		dataset, err := GenerateFromSchema("root", dstSchema, sampleNames, 10)
		if err != nil {
			t.Error("Schema generation error. " + err.Error())
			return
		}

		// Infer the schema.
		srcSchema, err := dataset.InferSchema()
		if err != nil {
			t.Error("Schema inference error. " + err.Error())
			return
		}

		// Make sure the inferred dataset matches the schema.
		match, _ := dstSchema.Match(srcSchema, false)

		if match == false {
			t.Error("Schema match failed.")
			return
		}
	}
}
