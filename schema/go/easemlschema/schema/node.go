package schema

// Node is.
type Node struct {
	IsSingleton bool
	Fields      map[string]Field
	Links       map[string]*Link
	SrcName     string
}

func (n *Node) match(source *Node, dimMap map[string]Dim, classNameMap map[string]string, nodeNameMap map[string]string, selfClasses map[string]*Class, sourceClasses map[string]*Class, buildMatching bool) (result bool, matching *Node, dimMapUpdates map[string]Dim, classNameMapUpdates map[string]string) {

	fieldNameMap := map[string]string{}
	dimMapUpdates = map[string]Dim{}
	classNameMapUpdates = map[string]string{}

	if source == nil {
		return false, nil, dimMapUpdates, classNameMapUpdates
	}

	// Split fields by type.
	selfTensorNames := []string{}
	sourceTensorNames := []string{}
	selfCategoryNames := []string{}
	sourceCategoryNames := []string{}

	for k, v := range n.Fields {
		switch v.Type() {
		case "tensor":
			selfTensorNames = append(selfTensorNames, k)
		case "category":
			selfCategoryNames = append(selfCategoryNames, k)
		}
	}

	for k, v := range source.Fields {
		switch v.Type() {
		case "tensor":
			sourceTensorNames = append(sourceTensorNames, k)
		case "category":
			sourceCategoryNames = append(sourceCategoryNames, k)
		}
	}

	// Simply dismiss in case the counts don't match.
	if len(selfTensorNames) != len(sourceTensorNames) || len(selfCategoryNames) != len(sourceCategoryNames) || len(n.Links) != len(source.Links) {
		return false, nil, dimMapUpdates, classNameMapUpdates
	}

	// We first try to match links as this the cheapest operation. We match based on
	// the node matching which must be given when this function is called.
	for k, v := range n.Links {
		sourceLink := source.Links[nodeNameMap[k]]
		if v.match(sourceLink) == false {
			return false, nil, dimMapUpdates, classNameMapUpdates
		}
	}

	// Try all possible tensor matchings. Individual field matchings are not independent
	// because of dimension matchings. Maybe this can be optimised.
	result = len(sourceTensorNames) == 0
	orig := getRange(len(sourceTensorNames))
	if len(orig) > 0 {
		for p := make([]int, len(orig)); p[0] < len(p); nextPerm(p) {
			perm := getPerm(orig, p)
			dimMapUpdatesIter := map[string]Dim{}
			dimMapIter := map[string]Dim{}
			for k, v := range dimMap {
				dimMapIter[k] = v
			}

			for i := 0; i < len(perm); i++ {
				selfField := n.Fields[selfTensorNames[i]].(*Tensor)
				sourceField := source.Fields[sourceTensorNames[perm[i]]].(*Tensor)

				var dimMapUpdatesNew map[string]Dim
				result, dimMapUpdatesNew = selfField.match(sourceField, dimMapIter)

				if result {
					// If there was a match, we update the dimension mappings.
					for k, v := range dimMapUpdatesNew {
						dimMapIter[k] = v
						dimMapUpdatesIter[k] = v
					}
				} else {
					// On the first failed match, we skip this permutation.
					break
				}
			}

			// If we have found a matching, end the search.
			if result {
				for k, v := range dimMapUpdatesIter {
					dimMapUpdates[k] = v
				}
				// Add matched field names to name map.
				for i := 0; i < len(perm); i++ {
					selfFieldName := selfTensorNames[i]
					sourceFieldName := sourceTensorNames[perm[i]]
					fieldNameMap[selfFieldName] = sourceFieldName
				}
			}
		}
	}

	// If no matching was found, we don't need to go on further.
	if result == false {
		return false, nil, dimMapUpdates, classNameMapUpdates
	}

	// Try all possible category matchings. Individual field matchings are not independent
	// because of dimension matchings. Maybe this can be optimised.
	result = len(sourceCategoryNames) == 0
	orig = getRange(len(sourceCategoryNames))
	if len(orig) > 0 {
		for p := make([]int, len(orig)); p[0] < len(p); nextPerm(p) {
			perm := getPerm(orig, p)
			dimMapUpdatesIter := map[string]Dim{}
			dimMapIter := map[string]Dim{}
			for k, v := range dimMap {
				dimMapIter[k] = v
			}
			for k, v := range dimMapUpdates {
				dimMapIter[k] = v
			}
			classNameMapUpdatesIter := map[string]string{}
			classNameMapIter := map[string]string{}
			for k, v := range classNameMap {
				classNameMapIter[k] = v
			}

			for i := 0; i < len(perm); i++ {
				selfField := n.Fields[selfCategoryNames[i]].(*Category)
				sourceField := source.Fields[sourceCategoryNames[perm[i]]].(*Category)

				var dimMapUpdatesNew map[string]Dim
				var classNameMapUpdatesNew map[string]string
				result, dimMapUpdatesNew, classNameMapUpdatesNew = selfField.match(sourceField, dimMapIter, classNameMapIter, selfClasses, sourceClasses)

				if result {
					// If there was a match, we update the dimension and class mappings.
					for k, v := range dimMapUpdatesNew {
						dimMapIter[k] = v
						dimMapUpdatesIter[k] = v
					}
					for k, v := range classNameMapUpdatesNew {
						classNameMapIter[k] = v
						classNameMapUpdatesIter[k] = v
					}
				} else {
					// On the first failed match, we skip this permutation.
					break
				}
			}

			// If we have found a matching, end the search.
			if result {
				for k, v := range dimMapUpdatesIter {
					dimMapUpdates[k] = v
				}
				for k, v := range classNameMapUpdatesIter {
					classNameMapUpdates[k] = v
				}
				// Add matched field names to name map.
				for i := 0; i < len(perm); i++ {
					selfFieldName := selfCategoryNames[i]
					sourceFieldName := sourceCategoryNames[perm[i]]
					fieldNameMap[selfFieldName] = sourceFieldName
				}
			}
		}
	}

	// If no matching was found, we don't need to go on further.
	if result == false {
		return false, nil, dimMapUpdates, classNameMapUpdates
	}

	// If a matching was found, build a resulting matching node if needed.
	if buildMatching {
		matching, err := loadNode(n.dump())
		if err != nil {
			panic(err)
		}
		for k, v := range matching.Fields {
			switch v.(type) {
			case *Tensor:
				// Assign the source name.
				v.(*Tensor).SrcName = fieldNameMap[k]

				// Assign the source dimensions.
				sourceTensor := source.Fields[fieldNameMap[k]].(*Tensor)
				clonedSourceField, err := loadField(sourceTensor.dump())
				if err != nil {
					panic(err)
				}
				v.(*Tensor).SrcDim = clonedSourceField.(*Tensor).Dim

			case *Category:
				// Assign the source name.
				v.(*Category).SrcName = fieldNameMap[k]
			}
		}
		return true, matching, dimMapUpdates, classNameMapUpdates
	}

	return true, nil, dimMapUpdates, classNameMapUpdates
}

func loadNode(input interface{}) (result *Node, err *schemaError) {

	if node, ok := input.(map[string]interface{}); ok {

		result = &Node{}

		nodeSingleton, ok := node["singleton"]
		if ok {
			result.IsSingleton, ok = nodeSingleton.(bool)
			if ok == false {
				err = &schemaError{err: "Node singleton flag must be a boolean."}
				return nil, err
			}
		}

		result.Fields = map[string]Field{}
		nodeFields, ok := node["fields"]
		if ok {
			nodeFieldsMap, ok := nodeFields.(map[string]interface{})
			if ok == false {
				err = &schemaError{err: "Node field must be a key-value dictionary."}
				return nil, err
			}
			for k, v := range nodeFieldsMap {

				if checkNameFormat(k) == false {
					err = &schemaError{err: "Node field targets may contain lowercase letters, numbers and underscores. They must start with a letter."}
					return nil, err
				}

				result.Fields[k], err = loadField(v)
				if err != nil {
					err.path = "fields." + k
					return nil, err
				}
			}
		} else if result.IsSingleton {
			result.Fields["field"], err = loadField(input)
			if err != nil {
				return nil, err
			}
		}

		nodeLinks, ok := node["links"]
		if ok {
			nodeLinksMap, ok := nodeLinks.(map[string]interface{})
			if ok == false {
				err = &schemaError{err: "Node links must be a key-value dictionary."}
				return nil, err
			}
			result.Links = map[string]*Link{}
			for k, v := range nodeLinksMap {

				if checkNameFormat(k) == false {
					err = &schemaError{err: "Node link targets may contain lowercase letters, numbers and underscores. They must start with a letter."}
					return nil, err
				}

				result.Links[k], err = loadLink(v)
				if err != nil {
					err.path = "links." + k
					return nil, err
				}
			}
		}

		if result.IsSingleton {
			if len(result.Fields) != 1 {
				err = &schemaError{err: "Singleton nodes must have a single field."}
				return nil, err
			}
			if len(result.Links) != 0 {
				err = &schemaError{err: "Singleton nodes cannot have links."}
				return nil, err
			}
		}

		if len(result.Fields)+len(result.Links) < 1 {
			err = &schemaError{err: "Node must have at least one field or link."}
			return nil, err
		}

		if srcName, ok := node["src-name"]; ok {
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

	err = &schemaError{err: "Node must be a key-value dictionary."}
	return nil, err
}

func (n *Node) dump() interface{} {
	result := map[string]interface{}{}

	result["singleton"] = n.IsSingleton

	if n.IsSingleton {
		for _, v := range n.Fields {
			fieldDump := v.dump().(map[string]interface{})
			for k, v := range fieldDump {
				result[k] = v
			}
		}
	} else {
		fieldsDump := map[string]interface{}{}
		for k, v := range n.Fields {
			fieldsDump[k] = v.dump()
		}
		result["fields"] = fieldsDump

		linksDump := map[string]interface{}{}
		for k, v := range n.Links {
			linksDump[k] = v.dump()
		}
		result["links"] = linksDump
	}

	if n.SrcName != "" {
		result["src-name"] = n.SrcName
	}

	return result
}
