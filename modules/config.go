package modules

import (
	"errors"
	"math"
	"math/rand"
)

// ConfigElem is.
type ConfigElem interface {
	Sample() ConfigElem
	Expand(rangeCount int) []ConfigElem
	Dump() interface{}
}

// LoadConfig is.
func LoadConfig(input interface{}) (ConfigElem, error) {
	if stringInput, ok := input.(string); ok {
		return &ConfigConst{Value: stringInput}, nil
	} else if floatInput, ok := input.(float64); ok {
		return &ConfigConst{Value: floatInput}, nil
	} else if mapInput, ok2 := input.(map[string]interface{}); ok2 {

		// Check if we are dealing with a special element.
		if _, ok := mapInput[".choice"]; ok {

			if choices, ok := mapInput[".choice"].([]interface{}); ok {

				loadedChoices := make([]ConfigElem, len(choices))
				for i := range choices {
					var err error
					loadedChoices[i], err = LoadConfig(choices[i])
					if err != nil {
						return nil, errors.New("invalid input config format")
					}
				}
				return &ConfigChoice{Choices: loadedChoices}, nil

			}
			return nil, errors.New("invalid input config format")

		} else if _, ok := mapInput[".float"]; ok {

			floatRange, ok := mapInput[".float"].([]interface{})
			if ok == false {
				return nil, errors.New("invalid input config format")
			}
			floatRangeFrom, ok := floatRange[0].(float64)
			if ok == false {
				return nil, errors.New("invalid input config format")
			}
			floatRangeTo, ok := floatRange[1].(float64)
			if ok == false {
				return nil, errors.New("invalid input config format")
			}

			result := ConfigFloat{From: floatRangeFrom, To: floatRangeTo}

			if scale, ok := mapInput[".scale"]; ok {
				result.Scale, ok = scale.(string)
				if ok == false {
					return nil, errors.New("invalid input config format")
				}
			}

			return &result, nil

		} else if _, ok := mapInput[".int"]; ok {

			intRange, ok := mapInput[".int"].([]interface{})
			if ok == false {
				return nil, errors.New("invalid input config format")
			}
			intRangeFrom, ok := intRange[0].(float64)
			if ok == false {
				return nil, errors.New("invalid input config format")
			}
			intRangeTo, ok := intRange[1].(float64)
			if ok == false {
				return nil, errors.New("invalid input config format")
			}

			result := ConfigInt{From: int(math.Round(intRangeFrom)), To: int(math.Round(intRangeTo))}

			if scale, ok := mapInput[".scale"]; ok {
				result.Scale, ok = scale.(string)
				if ok == false {
					return nil, errors.New("invalid input config format")
				}
			}

			return &result, nil

		} else {

			result := ConfigMap{}
			for k, v := range mapInput {
				var err error
				result[k], err = LoadConfig(v)
				if err != nil {
					return nil, errors.New("invalid input config format")
				}
			}
			return &result, nil
		}
	}
	return nil, errors.New("invalid input config format")
}

// ConfigConst is.
type ConfigConst struct {
	Value interface{}
}

// Sample is.
func (c *ConfigConst) Sample() ConfigElem { return c }

// Expand is.
func (c *ConfigConst) Expand(rangeCount int) []ConfigElem { return []ConfigElem{c} }

// Dump is.
func (c *ConfigConst) Dump() interface{} { return c.Value }

// ConfigChoice is.
type ConfigChoice struct {
	Choices []ConfigElem
}

// Sample is.
func (c *ConfigChoice) Sample() ConfigElem { return c.Choices[rand.Intn(len(c.Choices))] }

// Expand is.
func (c *ConfigChoice) Expand(rangeCount int) []ConfigElem { return c.Choices }

// Dump is.
func (c *ConfigChoice) Dump() interface{} {

	result := map[string]interface{}{}

	choices := make([]interface{}, len(c.Choices))
	for i := range c.Choices {
		choices[i] = c.Choices[i].Dump()
	}
	result[".choices"] = choices

	return result
}

// ConfigFloat is.
type ConfigFloat struct {
	From, To float64
	Scale    string
}

// Sample is.
func (c *ConfigFloat) Sample() ConfigElem {
	if c.Scale == "linear" || c.Scale == "" {
		return &ConfigConst{Value: rand.Float64()*(c.To-c.From) + c.From}
	}
	panic("not implemented yet")
}

// Expand is.
func (c *ConfigFloat) Expand(rangeCount int) []ConfigElem {
	result := make([]ConfigElem, rangeCount)
	for i := range result {
		if c.Scale == "linear" || c.Scale == "" {
			result[i] = &ConfigConst{
				Value: ((c.To-c.From)/float64(rangeCount))*float64(i) + c.From,
			}
		} else {
			panic("not implemented yet")
		}
	}
	return result
}

// Dump is.
func (c *ConfigFloat) Dump() interface{} {
	result := map[string]interface{}{}

	result[".float"] = []float64{c.From, c.To}
	if c.Scale != "" {
		result[".scale"] = c.Scale
	}

	return result
}

// ConfigInt is.
type ConfigInt struct {
	From, To int
	Scale    string
}

// Sample is.
func (c *ConfigInt) Sample() ConfigElem {
	if c.Scale == "linear" || c.Scale == "" {
		return &ConfigConst{Value: rand.Intn(c.To-c.From) + c.From}
	}
	panic("not implemented yet")
}

// Expand is.
func (c *ConfigInt) Expand(rangeCount int) []ConfigElem {
	result := make([]ConfigElem, rangeCount)
	for i := range result {
		if c.Scale == "linear" || c.Scale == "" {
			result[i] = &ConfigConst{
				Value: ((c.To-c.From)/rangeCount)*i + c.From,
			}
		} else {
			panic("not implemented yet")
		}
	}
	return result
}

// Dump is.
func (c *ConfigInt) Dump() interface{} {
	result := map[string]interface{}{}

	result[".float"] = []int{c.From, c.To}
	if c.Scale != "" {
		result[".scale"] = c.Scale
	}

	return result
}

// ConfigMap is.
type ConfigMap map[string]ConfigElem

// Sample is.
func (c *ConfigMap) Sample() ConfigElem {
	result := ConfigMap{}
	for k, v := range *c {
		result[k] = v.Sample()
	}
	return &result
}

// Expand is.
func (c *ConfigMap) Expand(rangeCount int) []ConfigElem {

	expansions := map[string][]ConfigElem{}
	keys := []string{}
	counts := []int{}
	for k, v := range *c {
		expansions[k] = v.Expand(rangeCount)
		keys = append(keys, k)
		counts = append(counts, len(expansions[k]))
	}

	result := []ConfigElem{}
	pos := make([]int, len(keys))
	for {

		// Build the current iteration of expansions.
		newMap := ConfigMap{}
		for i, k := range keys {
			newMap[k] = expansions[k][pos[i]]
		}
		result = append(result, &newMap)

		// Adjust positions.
		allZeros := true
		for i := range pos {
			pos[i] = (pos[i] + 1) % counts[i]
			if pos[i] > 0 {
				allZeros = false
				break
			}
		}

		// If we are back to all zeros, then we have enumerated them all.
		if allZeros {
			break
		}
	}
	return result
}

// Dump is.
func (c *ConfigMap) Dump() interface{} {
	result := map[string]interface{}{}

	for k, v := range *c {
		result[k] = v.Dump()
	}

	return result
}
