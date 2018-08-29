package modules

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDump(t *testing.T) {
	assert := assert.New(t)

	input := map[string]interface{}{
		"a": 1.1,
		"b": map[string]interface{}{
			".choice": []interface{}{"c1", "c2", "c3"},
		},
	}

	conf, err := LoadConfig(input)
	assert.Nil(err)

	output := conf.Dump()
	var ok bool

	_, ok = output.(map[string]interface{})
	assert.True(ok)
}

func TestSample(t *testing.T) {
	assert := assert.New(t)

	input := map[string]interface{}{
		"a": 1.1,
		"b": map[string]interface{}{
			".choice": []interface{}{"c1", "c2", "c3"},
		},
		"c": map[string]interface{}{
			".int": []interface{}{1.0, 5.0},
		},
		"d": map[string]interface{}{
			".float": []interface{}{1.0, 5.0},
		},
	}

	conf, err := LoadConfig(input)
	assert.Nil(err)

	sample := conf.Sample()
	assert.NotNil(sample)

	b, err := json.Marshal(sample.Dump())
	assert.Nil(err)
	fmt.Println(string(b))
}

func TestExpand(t *testing.T) {
	assert := assert.New(t)

	input := map[string]interface{}{
		"a": 1.1,
		"b": map[string]interface{}{
			".choice": []interface{}{"c1", "c2", "c3"},
		},
		"c": map[string]interface{}{
			".int": []interface{}{1.0, 5.0},
		},
		"d": map[string]interface{}{
			".float": []interface{}{1.0, 5.0},
		},
	}

	conf, err := LoadConfig(input)
	assert.Nil(err)

	expansion := conf.Expand(3)
	assert.Equal(27, len(expansion))
}
