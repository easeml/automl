package model

import (
	"bytes"
	"encoding/json"
	"fmt"

	sch "github.com/ds3lab/easeml/schema/schema"

	"github.com/globalsign/mgo/bson"
	"github.com/pkg/errors"
)

func setDefault(document *bson.M, key string, value interface{}) {
	if _, ok := (*document)[key]; ok == false {
		(*document)[key] = value
	}
}

func deserializeSchema(data string) (schema *sch.Schema, err error) {

	if data == "" {
		return nil, nil
	}

	// Decode JSON.
	var dataJSON map[string]interface{}
	err = json.Unmarshal([]byte(data), &dataJSON)
	if err != nil {
		fmt.Printf(data)
		err = errors.Wrap(err, "the given schema is not a valid JSON")
		return
	}
	// Load the schema.
	schema, err = sch.Load(dataJSON)
	if err != nil {
		err = errors.Wrap(ErrBadInput, "the given input schema is not valid")
		return
	}
	return
}

func jsonCompact(input string) (string, error) {
	var buf bytes.Buffer
	err := json.Compact(&buf, []byte(input))
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

/* func deserializeSchemas(data string) (schInput *sch.Schema, schOutput *sch.Schema, err error) {
	// Decode JSON.
	var dataJSON map[string]interface{}
	err = json.Unmarshal([]byte(data), &dataJSON)
	if err != nil {
		fmt.Printf(data)
		err = errors.Wrap(err, "the given schema is not a valid JSON")
		return
	}

	// Extract input and output fields.
	if dataInput, ok := dataJSON["input"]; ok {
		schInput, err = sch.Load(dataInput)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "the given input schema is not valid")
			return
		}
	}

	if dataOutput, ok := dataJSON["output"]; ok {
		schOutput, err = sch.Load(dataOutput)
		if err != nil {
			err = errors.Wrap(ErrBadInput, "the given output schema is not valid")
			return
		}
	}

	return
} */
