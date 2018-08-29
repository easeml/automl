package schema

// Class is.
type Class struct {
	Dim     Dim
	SrcName string
}

func (l *Class) match(source *Class, dimMap map[string]Dim) (bool, map[string]Dim) {
	if source == nil {
		return false, map[string]Dim{}
	}
	return l.Dim.Match(source.Dim, dimMap)
}

func loadClass(input interface{}) (result *Class, err *schemaError) {

	if class, ok := input.(map[string]interface{}); ok {

		result = &Class{}

		classDim, ok := class["dim"]
		if ok == false {
			err = &schemaError{err: "Class must have a 'dim' field."}
			return
		}
		result.Dim, err = loadDim(classDim)
		if err != nil {
			return nil, err
		}
		if srcName, ok := class["src-name"]; ok {
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

		return result, nil

	}

	err = &schemaError{err: "Class must be a key-value dictionary."}
	return
}

func (l *Class) dump() interface{} {
	result := map[string]interface{}{}
	result["dim"] = l.Dim.dump()
	if l.SrcName != "" {
		result["src-name"] = l.SrcName
	}
	return result
}
