package schema

import (
	"fmt"
	"reflect"
	"strings"
)

// Dim is.
type Dim interface {
	Equals(other Dim) bool
	IsVariable() bool
	IsWildcard() bool
	Match(source Dim, dimMap map[string]Dim) (bool, map[string]Dim)
	canOccurZeroTimes() bool
	canOccurInfiniteTimes() bool
	dump() interface{}
}

func loadDim(input interface{}) (result Dim, err *schemaError) {
	if intDim, ok := getInt(input); ok {
		if intDim < 1 {
			err = &schemaError{err: "Dimension must be a positive integer."}
			return
		}

		return &ConstDim{Value: intDim}, nil

	} else if stringDim, ok := input.(string); ok {
		if checkDimFormat(stringDim) == false {
			err = &schemaError{err: "Dimension can contain lowercase letters, numbers and underscores. They must start with a letter."}
			return
		}

		return &VarDim{Value: stringDim}, nil

	} else {
		err = &schemaError{err: fmt.Sprintf("Dimension must be an integer or a string, found %s.", reflect.TypeOf(input))}
		return
	}
}

// VarDim is.
type VarDim struct {
	Value string
}

// Equals is.
func (d *VarDim) Equals(other Dim) bool {
	if otherVar, ok := other.(*VarDim); ok {
		return d.Value == otherVar.Value
	}
	return false
}

// IsVariable is.
func (d *VarDim) IsVariable() bool {
	return true
}

// IsWildcard is.
func (d *VarDim) IsWildcard() bool {
	return len(d.Value) > 0 && strings.IndexAny(d.Value[len(d.Value)-1:], "?+*") != -1
}

func (d *VarDim) canOccurZeroTimes() bool {
	return len(d.Value) > 0 && strings.IndexAny(d.Value[len(d.Value)-1:], "?*") != -1
}

func (d *VarDim) canOccurInfiniteTimes() bool {
	return len(d.Value) > 0 && strings.IndexAny(d.Value[len(d.Value)-1:], "+*") != -1
}

// Match is.
func (d *VarDim) Match(source Dim, dimMap map[string]Dim) (bool, map[string]Dim) {
	if source == nil {
		return false, map[string]Dim{}
	}
	dimValue := d.Value
	if d.IsWildcard() {
		dimValue = dimValue[:len(dimValue)-1]
	}
	dimMapUpdate := map[string]Dim{}
	if mappedDim, ok := dimMap[dimValue]; ok {
		return mappedDim.Equals(source), dimMapUpdate
	}
	dimMapUpdate[dimValue] = source
	return true, dimMapUpdate
}

func (d *VarDim) dump() interface{} {
	return d.Value
}

// ConstDim is.
type ConstDim struct {
	Value int
}

// Equals is.
func (d *ConstDim) Equals(other Dim) bool {
	if otherVar, ok := other.(*ConstDim); ok {
		return d.Value == otherVar.Value
	}
	return false
}

// IsVariable is.
func (d *ConstDim) IsVariable() bool {
	return false
}

// IsWildcard is.
func (d *ConstDim) IsWildcard() bool {
	return false
}

func (d *ConstDim) canOccurZeroTimes() bool {
	return false
}

func (d *ConstDim) canOccurInfiniteTimes() bool {
	return false
}

// Match is.
func (d *ConstDim) Match(source Dim, dimMap map[string]Dim) (bool, map[string]Dim) {
	if source == nil {
		return false, map[string]Dim{}
	}
	emptyDimMap := map[string]Dim{}
	if sourceConst, ok := source.(*ConstDim); ok {
		return d.Value == sourceConst.Value, emptyDimMap
	}
	return false, emptyDimMap
}

func (d *ConstDim) dump() interface{} {
	return d.Value
}

func matchDimList(listA []Dim, listB []Dim, dimMap map[string]Dim) (result bool, dimMapUpdate map[string]Dim) {
	emptyDimMap := map[string]Dim{}
	var dimA, dimB Dim

	// Get first dimension of list A if possible.
	if len(listA) > 0 {
		dimA = listA[0]
	}

	// Get first dimension of list B if possible.
	if len(listB) > 0 {
		dimB = listB[0]
	}

	// If both lists are empty we simply return True.
	if dimA == nil && dimB == nil {
		return true, emptyDimMap
	}

	// Handle the case when only list A is empty.
	if dimA == nil {
		if dimB.canOccurZeroTimes() {
			return matchDimList(listA, listB[1:], dimMap)
		}
		return false, emptyDimMap
	}

	// Handle the case when only list B is empty.
	if dimB == nil {
		if dimA.canOccurZeroTimes() {
			return matchDimList(listA[1:], listB, dimMap)
		}
		return false, emptyDimMap
	}

	// Check if the dimensions match.
	match, newDimMapUpdate := dimA.Match(dimB, dimMap)
	//newDimMapUpdate := map[string]Dim{}
	for k, v := range dimMap {
		newDimMapUpdate[k] = v
	}

	// If we can match we can try to move on.
	if match {
		recMatch, recDimMapUpdate := matchDimList(listA[1:], listB[1:], newDimMapUpdate)
		if recMatch {
			for k, v := range recDimMapUpdate {
				newDimMapUpdate[k] = v
			}
			return true, newDimMapUpdate
		}
	}

	// We can match and move dim A to match current B with more dims.
	if match && dimB.canOccurInfiniteTimes() {
		recMatch, recDimMapUpdate := matchDimList(listA[1:], listB, newDimMapUpdate)
		if recMatch {
			for k, v := range recDimMapUpdate {
				newDimMapUpdate[k] = v
			}
			return true, newDimMapUpdate
		}
	}

	// We can skip dim A if it is skippable.
	if dimA.canOccurZeroTimes() {
		recMatch, recDimMapUpdate := matchDimList(listA[1:], listB, dimMap)
		if recMatch {
			for k, v := range dimMap {
				recDimMapUpdate[k] = v
			}
			return true, recDimMapUpdate
		}
	}

	// We can match and move dim B to match current A with more dims.
	if match && dimA.canOccurInfiniteTimes() {
		recMatch, recDimMapUpdate := matchDimList(listA, listB[1:], newDimMapUpdate)
		if recMatch {
			for k, v := range recDimMapUpdate {
				newDimMapUpdate[k] = v
			}
			return true, newDimMapUpdate
		}
	}

	// We can skip dim B if it is skippable.
	if dimB.canOccurZeroTimes() {
		recMatch, recDimMapUpdate := matchDimList(listA, listB[1:], dimMap)
		if recMatch {
			for k, v := range dimMap {
				recDimMapUpdate[k] = v
			}
			return true, recDimMapUpdate
		}
	}

	return false, emptyDimMap
}
