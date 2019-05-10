package schema

import (
	"fmt"
)

// Field is.
type Field interface {
	Type() string
	dump() interface{}
}

// Tensor is.
type Tensor struct {
	Dim     []Dim
	SrcDim  []Dim
	SrcName string
}

// Category is.
type Category struct {
	Class   string
	SrcName string
}

func loadField(input interface{}) (result Field, err *schemaError) {

	if field, ok := input.(map[string]interface{}); ok {

		fieldType, ok := field["type"]
		if ok == false {
			err = &schemaError{err: "Field must have a 'type' field."}
			return
		}
		switch fieldType {
		case "tensor":
			return loadTensor(field)
		case "category":
			return loadCategory(field)
		default:
			err = &schemaError{err: fmt.Sprintf("Unknown field type '%s'.", fieldType)}
			return
		}
	}

	err = &schemaError{err: "Field must be a key-value dictionary."}
	return
}

// Type is.
func (f *Tensor) Type() string {
	return "tensor"
}

func (f *Tensor) match(source *Tensor, dimMap map[string]Dim) (result bool, dimMapUpdate map[string]Dim) {
	if source == nil {
		return false, map[string]Dim{}
	}
	return matchDimList(f.Dim, source.Dim, dimMap)
}

func (f *Tensor) dump() interface{} {
	result := map[string]interface{}{}
	result["type"] = "tensor"

	dim := make([]interface{}, len(f.Dim))
	for i := range f.Dim {
		dim[i] = f.Dim[i].dump()
	}
	result["dim"] = dim

	if len(f.SrcDim) != len(f.Dim) {
		srcDim := make([]interface{}, len(f.Dim))
		for i := range f.Dim {
			srcDim[i] = f.Dim[i].dump()
		}
		result["src-dim"] = srcDim
	}

	if f.SrcName != "" {
		result["src-name"] = f.SrcName
	}

	return result
}

func loadTensor(input map[string]interface{}) (result *Tensor, err *schemaError) {

	result = &Tensor{}

	tensorDim, ok := input["dim"]
	if ok == false {
		err = &schemaError{err: "Tensor must have a 'dim' field."}
		return nil, err
	}
	tensorDimList, ok := tensorDim.([]interface{})
	if ok == false {
		err = &schemaError{err: "Tensor dim field must be a list of dimension definitions."}
		return nil, err
	}
	if len(tensorDimList) < 1 {
		err = &schemaError{err: "Tensor must have at least one dimension."}
		return nil, err
	}
	result.Dim = make([]Dim, len(tensorDimList))
	for i := range tensorDimList {
		result.Dim[i], err = loadDim(tensorDimList[i])
		if err != nil {
			return nil, err
		}
	}

	tensorSrcDim, ok := input["src-dim"]
	if ok {
		tensorSrcDimList, ok := tensorSrcDim.([]interface{})
		if ok == false {
			err = &schemaError{err: "Tensor dim field must be a list of dimension definitions."}
			return nil, err
		}
		if len(tensorSrcDimList) < 1 {
			err = &schemaError{err: "Tensor must have at least one dimension."}
			return nil, err
		}
		result.SrcDim = make([]Dim, len(tensorSrcDimList))
		for i := range tensorSrcDimList {
			result.SrcDim[i], err = loadDim(tensorSrcDimList[i])
			if err != nil {
				return nil, err
			}
		}
	}

	if srcName, ok := input["src-name"]; ok {
		result.SrcName, ok = srcName.(string)
		if ok == false {
			err = &schemaError{err: "Source name must be a string."}
			return nil, err
		}
		if checkNameFormat(result.SrcName) == false {
			err = &schemaError{err: "Source name may contain lowercase letters, numbers and underscores. They must start with a letter."}
			return nil, err
		}
	}

	foundWildcard := false
	for i := range result.Dim {
		if result.Dim[i].IsWildcard() {
			if foundWildcard {
				err = &schemaError{err: "Tensor can have at most one variable count dimension."}
				return nil, err
			}
			foundWildcard = true
		}
	}

	if len(result.Dim) == 1 && result.Dim[0].canOccurZeroTimes() {
		err = &schemaError{err: "Tensors cannot have zero dimensions. Having only one dimension suffixed with '?' or '*' permits this."}
		return nil, err
	}

	return result, nil
}

// Type is.
func (f *Category) Type() string {
	return "category"
}

func (f *Category) match(
	source *Category,
	dimMap map[string]Dim,
	classNameMap map[string]string,
	selfClasses map[string]*Class,
	sourceClasses map[string]*Class,
) (result bool, dimMapUpdate map[string]Dim, classNameMapUpdate map[string]string) {

	emptyDimMap := map[string]Dim{}
	emptyClassNameMap := map[string]string{}

	if source == nil {
		return false, emptyDimMap, emptyClassNameMap
	}

	// If the class has already been mapped then we simply compare.
	if mappedClass, ok := classNameMap[f.Class]; ok {
		return mappedClass == source.Class, emptyDimMap, emptyClassNameMap
	}

	selfClass := selfClasses[f.Class]
	sourceClass := sourceClasses[source.Class]
	match, dimMapUpdate := selfClass.match(sourceClass, dimMap)
	if match {
		emptyClassNameMap[f.Class] = source.Class
		return true, dimMapUpdate, emptyClassNameMap
	}

	return false, emptyDimMap, emptyClassNameMap
}

func (f *Category) dump() interface{} {
	result := map[string]interface{}{}
	result["type"] = "category"
	result["class"] = f.Class
	if f.SrcName != "" {
		result["src-name"] = f.SrcName
	}
	return result
}

func loadCategory(input map[string]interface{}) (result *Category, err *schemaError) {

	result = &Category{}

	categoryClass, ok := input["class"]
	if ok == false {
		err = &schemaError{err: "Category must have a 'class' field."}
		return nil, err
	}
	result.Class, ok = categoryClass.(string)
	if ok == false {
		err = &schemaError{err: "Category class must be a string."}
		return nil, err
	}
	if checkNameFormat(result.Class) == false {
		err = &schemaError{err: "Category class may contain lowercase letters, numbers and underscores. They must start with a letter."}
		return nil, err
	}

	categorySrcName, ok := input["src-name"]
	if ok {
		result.SrcName, ok = categorySrcName.(string)
		if ok == false {
			err = &schemaError{err: "Source name must be a string."}
			return nil, err
		}
		if checkNameFormat(result.SrcName) == false {
			err = &schemaError{err: "Source name may contain lowercase letters, numbers and underscores. They must start with a letter."}
			return nil, err
		}
	}

	return
}
