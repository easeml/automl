package schema // import "github.com/ds3lab/easeml/schema/go/easemlschema/schema"

// Schema is.
type Schema struct {
	Nodes        map[string]*Node
	Classes      map[string]*Class
	SrcDims      map[string]Dim
	IsUndirected bool
	IsCyclic     bool
	IsFanIn      bool
}

// Load is.
func Load(input interface{}) (*Schema, Error) {

	if schema, ok := input.(map[string]interface{}); ok {

		result := &Schema{}
		var err *schemaError

		result.Nodes = map[string]*Node{}
		schemaNodes, ok := schema["nodes"]
		if ok == false {
			err = &schemaError{err: "Schema must have a 'nodes' field."}
			return nil, err
		}
		schemaNodesMap, ok := schemaNodes.(map[string]interface{})
		if ok == false {
			err = &schemaError{err: "Schema nodes must be a key-value dictionary."}
			return nil, err
		}
		for k, v := range schemaNodesMap {

			if checkNameFormat(k) == false {
				err = &schemaError{err: "Schema node names may contain lowercase letters, numbers and underscores. They must start with a letter."}
				return nil, err
			}

			result.Nodes[k], err = loadNode(v)
			if err != nil {
				err.path = "nodes." + k + "." + err.path
				return nil, err
			}
		}
		if len(result.Nodes) < 1 {
			err = &schemaError{err: "Schema must have at least one node."}
			return nil, err
		}

		result.Classes = map[string]*Class{}
		schemaClasses, ok := schema["classes"]
		if ok {

			schemaClassesMap, ok := schemaClasses.(map[string]interface{})
			if ok == false {
				err = &schemaError{err: "Schema category classes must be a key-value dictionary."}
				return nil, err
			}

			for k, v := range schemaClassesMap {

				if checkNameFormat(k) == false {
					err = &schemaError{err: "Schema category class names may contain lowercase letters, numbers and underscores. They must start with a letter."}
					return nil, err
				}

				result.Classes[k], err = loadClass(v)
				if err != nil {
					err.path = "classes." + k + "." + err.path
					return nil, err
				}
			}

		}

		schemaRefConstraints, ok := schema["ref-constraints"]
		if ok {

			schemaRefConstraintsMap, ok := schemaRefConstraints.(map[string]interface{})
			if ok == false {
				err = &schemaError{err: "Reference constraints field must be a key-value dictionary."}
				return nil, err
			}

			cyclicConstraint, ok := schemaRefConstraintsMap["cyclic"]
			if ok {
				result.IsCyclic, ok = cyclicConstraint.(bool)
				if ok == false {
					err = &schemaError{err: "Schema cyclic flag must be a boolean."}
					return nil, err
				}
			}

			undirectedConstraint, ok := schemaRefConstraintsMap["undirected"]
			if ok {
				result.IsUndirected, ok = undirectedConstraint.(bool)
				if ok == false {
					err = &schemaError{err: "Schema undirected flag must be a boolean."}
					return nil, err
				}
			}

			faninConstraint, ok := schemaRefConstraintsMap["fan-in"]
			if ok {
				result.IsFanIn, ok = faninConstraint.(bool)
				if ok == false {
					err = &schemaError{err: "Schema fan-in flag must be a boolean."}
					return nil, err
				}
			}
		}

		// Reference checks.
		orphanClasses := map[string]interface{}{}
		for k := range result.Classes {
			orphanClasses[k] = nil
		}
		for _, v := range result.Nodes {

			// Check links.
			linkCount := 0
			for t := range v.Links {
				n, ok := result.Nodes[t]
				if ok == false {
					err = &schemaError{err: "Node link points to unknown node."}
					return nil, err
				}
				if n.IsSingleton {
					err = &schemaError{err: "Node link points to a singleton node."}
					return nil, err
				}
				if result.IsUndirected && !result.IsFanIn {
					if v.Links[t].IsUnbounded() {
						err = &schemaError{err: "Nodes in undirected schemas with fan-in cannot have infinite outgoing links."}
						return nil, err
					}
					linkCount += v.Links[t].UBound
				}
			}

			// Check category field classes.
			for f := range v.Fields {
				if category, ok := v.Fields[f].(*Category); ok {
					if _, ok := result.Classes[category.Class]; ok == false {
						err = &schemaError{err: "Field category class undefined."}
						return nil, err
					}
					delete(orphanClasses, category.Class)
				}
			}

			if result.IsUndirected && !result.IsFanIn && linkCount > 2 {
				err = &schemaError{err: "Nodes in undirected schemas with fan-in can have at most 2 outgoing links."}
				return nil, err
			}
		}
		if len(orphanClasses) > 0 {
			err = &schemaError{err: "Every declared class must be referenced in a category."}
			return nil, err
		}

		// Source dimensions check.
		schemaSrcDims, ok := schema["src-dims"]
		if ok {
			schemaSrcDimsMap, ok := schemaSrcDims.(map[string]interface{})
			if ok == false {
				err = &schemaError{err: "Source dimensions field must be a key-value dictionary."}
				return nil, err
			}
			result.SrcDims = map[string]Dim{}
			for k, v := range schemaSrcDimsMap {
				result.SrcDims[k], err = loadDim(v)
				if err != nil {
					return nil, err
				}
			}
		}

		return result, nil

	}

	return nil, &schemaError{err: "Schema must be a key-value dictionary."}
}

// Dump is.
func (s *Schema) Dump() interface{} {
	result := map[string]interface{}{}

	result["ref-constraints"] = map[string]interface{}{
		"cyclic":     s.IsCyclic,
		"undirected": s.IsUndirected,
		"fan-in":     s.IsFanIn,
	}

	resultNodesMap := map[string]interface{}{}
	for k, v := range s.Nodes {
		resultNodesMap[k] = v.dump()
	}
	result["nodes"] = resultNodesMap

	if len(s.Classes) > 0 {
		resultClassesMap := map[string]interface{}{}
		for k, v := range s.Classes {
			resultClassesMap[k] = v.dump()
		}
		result["classes"] = resultClassesMap
	}

	if len(s.SrcDims) > 0 {
		resultSrcDimsMap := map[string]interface{}{}
		for k, v := range s.SrcDims {
			resultSrcDimsMap[k] = v.dump()
		}
		result["src-dims"] = resultSrcDimsMap
	}

	return result
}

// Match is.
func (s *Schema) Match(source *Schema, buildMatching bool) (bool, *Schema) {

	if source == nil {
		return false, nil
	}

	dimMap := map[string]Dim{}
	classNameMap := map[string]string{}
	nodeNameMap := map[string]string{}
	nodes := map[string]*Node{}
	match := false

	// Next we split the nodes into singletons and non-singletons.
	selfSingletonNames := []string{}
	sourceSingletonNames := []string{}
	selfNonSingletonNames := []string{}
	sourceNonSingletonNames := []string{}

	for k, v := range s.Nodes {
		if v.IsSingleton {
			selfSingletonNames = append(selfSingletonNames, k)
		} else {
			selfNonSingletonNames = append(selfNonSingletonNames, k)
		}
	}

	for k, v := range source.Nodes {
		if v.IsSingleton {
			sourceSingletonNames = append(sourceSingletonNames, k)
		} else {
			sourceNonSingletonNames = append(sourceNonSingletonNames, k)
		}
	}

	// Simply dismiss in case the counts don't match.
	if len(selfSingletonNames) != len(sourceSingletonNames) || len(selfNonSingletonNames) != len(sourceNonSingletonNames) || len(s.Classes) != len(source.Classes) {
		return false, nil
	}

	// We only compare referential constraints if there are non-singleton nodes.
	if len(selfNonSingletonNames) > 0 {
		if source.IsCyclic && !s.IsCyclic {
			// A cyclic graph cannot be accepted by an acyclic destination.
			return false, nil
		}
		if !source.IsUndirected && s.IsUndirected {
			// A directed graph cannot be accepted by an undirected destination.
			return false, nil
		}
		if source.IsFanIn && !s.IsFanIn {
			// A graph that allows fan-in (multiple incoming pointers per node) cannot be accepted by
			// a destination that forbids it.
			return false, nil
		}
	}

	// Try all possible singleton node matchings. Individual field matchings are not independent
	// because of dimension matchings. Maybe this can be optimised.
	match = len(sourceSingletonNames) == 0
	orig := getRange(len(sourceSingletonNames))
	if len(orig) > 0 {
		for p := make([]int, len(orig)); p[0] < len(p); nextPerm(p) {
			perm := getPerm(orig, p)
			dimMapIter := map[string]Dim{}
			classNameMapIter := map[string]string{}
			nodesIter := map[string]*Node{}

			for i := 0; i < len(perm); i++ {
				selfNode := s.Nodes[selfSingletonNames[i]]
				sourceNode := source.Nodes[sourceSingletonNames[perm[i]]]

				var nodeMatching *Node
				var dimMapUpdatesNew map[string]Dim
				var classNameMapUpdatesNew map[string]string
				match, nodeMatching, dimMapUpdatesNew, classNameMapUpdatesNew = selfNode.match(sourceNode, dimMapIter, classNameMapIter, nodeNameMap, s.Classes, source.Classes, buildMatching)

				if match {
					// If there was a match, we update the dimension and class mappings.
					for k, v := range dimMapUpdatesNew {
						dimMapIter[k] = v
					}
					for k, v := range classNameMapUpdatesNew {
						classNameMapIter[k] = v
					}
					// Add node matching.
					if buildMatching {
						nodeMatching.SrcName = sourceSingletonNames[perm[i]]
						nodesIter[selfSingletonNames[i]] = nodeMatching
					}
				} else {
					// On the first failed match, we skip this permutation.
					break
				}
			}

			// If we have found a matching, end the search.
			if match {
				for k, v := range dimMapIter {
					dimMap[k] = v
				}
				for k, v := range classNameMapIter {
					classNameMap[k] = v
				}
				for i := 0; i < len(perm); i++ {
					nodeNameMap[selfSingletonNames[i]] = sourceSingletonNames[perm[i]]
				}
				// Add matched nodes to map.
				if buildMatching {
					for k, v := range nodesIter {
						nodes[k] = v
					}
				}
				break
			}
		}
	}

	// If no matching was found, we don't need to go on further.
	if match == false {
		return false, nil
	}

	// Try all possible non-singleton node matchings. Individual field matchings are not independent
	// because of dimension matchings. Maybe this can be optimised.
	match = len(sourceNonSingletonNames) == 0
	orig = getRange(len(sourceNonSingletonNames))
	if len(orig) > 0 {
		for p := make([]int, len(orig)); p[0] < len(p); nextPerm(p) {
			perm := getPerm(orig, p)
			dimMapIter := map[string]Dim{}
			dimMapUpdatesIter := map[string]Dim{}
			classNameMapIter := map[string]string{}
			classNameMapUpdatesIter := map[string]string{}
			nodesIter := map[string]*Node{}
			nodeNameMapIter := map[string]string{}

			for k, v := range dimMap {
				dimMapIter[k] = v
			}
			for k, v := range classNameMap {
				classNameMapIter[k] = v
			}
			for k, v := range nodeNameMap {
				nodeNameMapIter[k] = v
			}
			for i := 0; i < len(perm); i++ {
				nodeNameMapIter[selfNonSingletonNames[i]] = sourceNonSingletonNames[perm[i]]
			}

			for i := 0; i < len(perm); i++ {
				selfNode := s.Nodes[selfNonSingletonNames[i]]
				sourceNode := source.Nodes[sourceNonSingletonNames[perm[i]]]

				var nodeMatching *Node
				var dimMapUpdatesNew map[string]Dim
				var classNameMapUpdatesNew map[string]string
				match, nodeMatching, dimMapUpdatesNew, classNameMapUpdatesNew = selfNode.match(sourceNode, dimMapIter, classNameMapIter, nodeNameMapIter, s.Classes, source.Classes, buildMatching)

				if match {
					// If there was a match, we update the dimension and class mappings.
					for k, v := range dimMapUpdatesNew {
						dimMapIter[k] = v
						dimMapUpdatesIter[k] = v
					}
					for k, v := range classNameMapUpdatesNew {
						classNameMapIter[k] = v
						classNameMapUpdatesIter[k] = v
					}
					// Add node matching.
					if buildMatching {
						nodeMatching.SrcName = sourceNonSingletonNames[perm[i]]
						nodesIter[selfNonSingletonNames[i]] = nodeMatching
					}
				} else {
					// On the first failed match, we skip this permutation.
					break
				}
			}

			// If we have found a matching, end the search.
			if match {
				for k, v := range dimMapUpdatesIter {
					dimMap[k] = v
				}
				for k, v := range classNameMapUpdatesIter {
					classNameMap[k] = v
				}
				for i := 0; i < len(perm); i++ {
					nodeNameMap[selfNonSingletonNames[i]] = sourceNonSingletonNames[perm[i]]
				}
				// Add matched nodes to map.
				if buildMatching {
					for k, v := range nodesIter {
						nodes[k] = v
					}
				}
				break
			}
		}
	}

	// If no matching was found, we don't need to go on further.
	if match == false {
		return false, nil
	}

	// If a matching was found, build a resulting matching node if needed.
	var schema *Schema
	if buildMatching {
		classes := map[string]*Class{}
		for k, v := range s.Classes {
			classes[k] = &Class{Dim: v.Dim, SrcName: classNameMap[k]}
		}
		schema = &Schema{
			IsCyclic:     s.IsCyclic,
			IsUndirected: s.IsUndirected,
			IsFanIn:      s.IsFanIn,
			Classes:      classes,
			Nodes:        nodes,
			SrcDims:      dimMap,
		}
	}

	return true, schema
}
