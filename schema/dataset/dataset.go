package dataset // import "github.com/ds3lab/easeml/schema/dataset"

import (
	"fmt"
	"math/rand"
	"strings"

	sch "github.com/ds3lab/easeml/schema/schema"
)

// File is.
type File interface {
	Type() string
	Subtype() string
}

// Dataset is.
type Dataset struct {
	Directory
	Root string
}

// LoaderFunc is.
type LoaderFunc func(string, string, string, Opener, bool, string) (File, error)

// TypeExtensions is.
var TypeExtensions = map[string]map[string]string{
	"tensor":   map[string]string{"default": ".ten.npy", "csv": ".ten.csv"},
	"category": map[string]string{"default": ".cat.txt"},
	"class":    map[string]string{"default": ".class.txt"},
	"links":    map[string]string{"default": ".links.csv"},
}

// LoaderFunctions is.
var LoaderFunctions = map[string]LoaderFunc{
	"tensor":   loadTensor,
	"category": loadCategory,
	"class":    loadClass,
	"links":    loadLinks,
}

// Load is.
func Load(root string, metadataOnly bool, opener Opener) (*Dataset, error) {
	directory, err := loadDirectory(root, "", "", opener, metadataOnly, "default")
	if err != nil {
		return nil, err
	}
	return &Dataset{Root: root, Directory: *(directory.(*Directory))}, nil
}

// Dump is.
func (d *Dataset) Dump(root string, opener Opener) error {
	return d.dump(root, "", "", opener)
}

// InferSchema is.
func (d *Dataset) InferSchema() (*sch.Schema, Error) {

	classes := map[string]*Class{}
	classSets := map[string]map[string]interface{}{}
	schClasses := map[string]*sch.Class{}
	samples := map[string]*Directory{}

	for k, v := range d.Directory.Children {
		if directory, ok := v.(*Directory); ok {

			// A directory corresponds to a data sample.
			samples[k] = directory

		} else if class, ok := v.(*Class); ok {

			// Collectclass file and get its dimensions.
			classes[k] = class
			schClasses[k] = &sch.Class{Dim: &sch.ConstDim{Value: len(class.Categories)}}

			classSet := map[string]interface{}{}
			for i := range class.Categories {
				classSet[class.Categories[i]] = nil
			}
			classSets[k] = classSet

		} else {
			msg := fmt.Sprintf("Files of type '%s' are unexpected in dataset root.", v.Type())
			pth := strings.Join([]string{"", k}, "/")
			err := &datasetError{err: msg, path: pth}
			return nil, err
		}
	}

	schNodes := map[string]*sch.Node{}
	firstSample := true
	linksFileFound := false
	schCyclic := false
	schFanin := false
	schUndirected := true

	// Go through all data samples.
	for sampleName, sample := range samples {

		// Sort all sample hidren according to their type.
		sampleLinks := map[string]*Links{}
		sampleTensors := map[string]*Tensor{}
		sampleCategories := map[string]*Category{}
		sampleDirectories := map[string]*Directory{}
		sampleNodes := map[string]interface{}{}

		for childName, child := range sample.Children {

			if tensor, ok := child.(*Tensor); ok {
				sampleTensors[childName] = tensor
				sampleNodes[childName] = nil

			} else if category, ok := child.(*Category); ok {
				sampleCategories[childName] = category
				sampleNodes[childName] = nil

			} else if links, ok := child.(*Links); ok {
				sampleLinks[childName] = links
				if firstSample {
					linksFileFound = true
				} else if linksFileFound == false {
					msg := "Links file not found in all data samples."
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}

			} else if directory, ok := child.(*Directory); ok {
				sampleDirectories[childName] = directory
				sampleNodes[childName] = nil
			}
		}

		// Ensure that either all samples have links files or none of them have.
		if (len(sampleLinks) > 0) != linksFileFound {
			msg := "Links file not found in all data samples."
			pth := strings.Join([]string{"", sampleName}, "/")
			err := &datasetError{err: msg, path: pth}
			return nil, err
		}

		// Ensure all samples have the same node names.
		schemaNodesSet := map[string]interface{}{}
		for k := range schNodes {
			schemaNodesSet[k] = nil
		}
		if firstSample == false {
			schemaSampleSuperset := isSuperset(schemaNodesSet, sampleNodes)
			sampleSchemaSuperset := isSuperset(sampleNodes, schemaNodesSet)
			if !(schemaSampleSuperset && sampleSchemaSuperset) {
				var childName string
				var msg string

				if schemaSampleSuperset {
					for k := range schemaNodesSet {
						if _, ok := sampleNodes[k]; ok == false {
							childName = k
							break
						}
					}
					msg = "Item expected but not found."

				} else if sampleSchemaSuperset {
					for k := range sampleNodes {
						if _, ok := schemaNodesSet[k]; ok == false {
							childName = k
							break
						}
					}
					msg = "Item found but not expected."
				}
				pth := strings.Join([]string{"", sampleName, childName}, "/")
				err := &datasetError{err: msg, path: pth}
				return nil, err
			}
		}

		// Handle tensor singleton nodes.
		for childName, child := range sampleTensors {
			if firstSample {
				dimensions := make([]sch.Dim, len(child.Dimensions))
				for i := range child.Dimensions {
					dimensions[i] = &sch.ConstDim{Value: child.Dimensions[i]}
				}
				tensor := &sch.Tensor{Dim: dimensions}
				schNodes[childName] = &sch.Node{IsSingleton: true, Fields: map[string]sch.Field{"field": tensor}}

			} else {
				// Verify that the node is the same.
				node := schNodes[childName]
				if node.IsSingleton == false || len(node.Fields) > 1 || node.Fields["field"].Type() != "tensor" {
					msg := fmt.Sprintf("Node '%s' not the same type in all samples.", childName)
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err

				}
				missmatch := len(child.Dimensions) != len(node.Fields["field"].(*sch.Tensor).Dim)
				if missmatch == false {
					for i := range child.Dimensions {
						if node.Fields["field"].(*sch.Tensor).Dim[i].(*sch.ConstDim).Value != child.Dimensions[i] {
							missmatch = true
							break
						}
					}
				}
				if missmatch {
					msg := "Tensor dimensions mismatch."
					pth := strings.Join([]string{"", sampleName, childName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
			}
		}

		// Handle category singleton nodes.
		for childName, child := range sampleCategories {

			// Infer class by finding first class to which the node belongs.
			var class string
			for className, categorySet := range classSets {
				if child.belongsToSet(categorySet) {
					class = className
					break
				}
			}
			if class == "" {
				msg := "Category file does not match any class."
				pth := strings.Join([]string{"", sampleName, childName}, "/")
				err := &datasetError{err: msg, path: pth}
				return nil, err
			}

			if firstSample {
				category := &sch.Category{Class: class}
				schNodes[childName] = &sch.Node{IsSingleton: true, Fields: map[string]sch.Field{"field": category}}
			} else {
				// Verify that the node is the same.
				node := schNodes[childName]
				if node.IsSingleton == false || len(node.Fields) > 1 || node.Fields["field"].Type() != "category" {
					msg := fmt.Sprintf("Node '%s' not the same type in all samples.", childName)
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
				if node.Fields["field"].(*sch.Category).Class != class {
					msg := "Category class mismatch."
					pth := strings.Join([]string{"", sampleName, childName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
			}
		}

		// This counts how many instances each node has. It is used to validate link targets.
		nodeInstanceCount := map[string]int{}

		// Handle regular non-singleton nodes.
		for childName, child := range sampleDirectories {
			fields := map[string]sch.Field{}
			if firstSample == false {
				node := schNodes[childName]
				fields = node.Fields
				if node.IsSingleton {
					msg := fmt.Sprintf("Node '%s' not the same type in all samples.", childName)
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
			}

			// Ensure nodes in all samples have the same children.
			fieldsSet := map[string]interface{}{}
			for k := range fields {
				fieldsSet[k] = nil
			}
			childrenSet := map[string]interface{}{}
			for k := range child.Children {
				childrenSet[k] = nil
			}
			if firstSample == false {
				fieldsChildrenSuperset := isSuperset(fieldsSet, childrenSet)
				childrenFieldsSuperset := isSuperset(childrenSet, fieldsSet)
				if !(fieldsChildrenSuperset && childrenFieldsSuperset) {
					var nodeChildName string
					var msg string

					if fieldsChildrenSuperset {
						for k := range fieldsSet {
							if _, ok := childrenSet[k]; ok == false {
								nodeChildName = k
								break
							}
						}
						msg = "Item expected but not found."

					} else if childrenFieldsSuperset {
						for k := range childrenSet {
							if _, ok := fieldsSet[k]; ok == false {
								nodeChildName = k
								break
							}
						}
						msg = "Item found but not expected."
					}
					pth := strings.Join([]string{"", sampleName, childName, nodeChildName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
			}

			// Go through all fields of the non-singleton node.
			for nodeChildName, nodeChild := range child.Children {

				if tensorNodeChild, ok := nodeChild.(*Tensor); ok {

					// Verify that all node fields have the same number of instances.
					childCount := tensorNodeChild.Dimensions[0]
					if count, ok := nodeInstanceCount[childName]; ok && count != childCount {
						msg := "Tensor instance count mismatch."
						pth := strings.Join([]string{"", sampleName, childName, nodeChildName}, "/")
						err := &datasetError{err: msg, path: pth}
						return nil, err
					}
					nodeInstanceCount[childName] = childCount

					if firstSample {
						dimensions := make([]sch.Dim, len(tensorNodeChild.Dimensions)-1)
						for i := 0; i < len(tensorNodeChild.Dimensions)-1; i++ {
							dimensions[i] = &sch.ConstDim{Value: tensorNodeChild.Dimensions[i+1]}
						}
						fields[nodeChildName] = &sch.Tensor{Dim: dimensions}

					} else {
						// Verify that the node is the same.
						field := fields[nodeChildName]
						tensorField, ok := field.(*sch.Tensor)
						if ok == false {
							msg := fmt.Sprintf("Node '%s' not the same type in all samples.", childName)
							pth := strings.Join([]string{"", sampleName, childName, nodeChildName}, "/")
							err := &datasetError{err: msg, path: pth}
							return nil, err
						}
						missmatch := len(tensorNodeChild.Dimensions) != len(tensorField.Dim)+1
						if missmatch == false {
							for i := range tensorField.Dim {
								if tensorField.Dim[i].(*sch.ConstDim).Value != tensorNodeChild.Dimensions[i+1] {
									missmatch = true
									break
								}
							}
						}
						if missmatch {
							msg := "Tensor dimensions mismatch."
							pth := strings.Join([]string{"", sampleName, childName}, "/")
							err := &datasetError{err: msg, path: pth}
							return nil, err
						}
					}

				} else if categoryNodeChild, ok := nodeChild.(*Category); ok {

					// Infer class by finding first class to which the node belongs.
					var class string
					for className, categorySet := range classSets {
						if categoryNodeChild.belongsToSet(categorySet) {
							class = className
							break
						}
					}
					if class == "" {
						msg := "Category file does not match any class."
						pth := strings.Join([]string{"", sampleName, childName, nodeChildName}, "/")
						err := &datasetError{err: msg, path: pth}
						return nil, err
					}

					// Verify that all node fields have the same number of instances.
					childCount := len(categoryNodeChild.Categories)
					if count, ok := nodeInstanceCount[childName]; ok && count != childCount {
						msg := "Category instance count mismatch."
						pth := strings.Join([]string{"", sampleName, childName, nodeChildName}, "/")
						err := &datasetError{err: msg, path: pth}
						return nil, err
					}
					nodeInstanceCount[childName] = childCount

					if firstSample {
						fields[nodeChildName] = &sch.Category{Class: class}
					} else {
						// Verify that the node is the same.
						field := fields[nodeChildName]
						categoryField, ok := field.(*sch.Category)
						if ok == false {
							msg := fmt.Sprintf("Node '%s' not the same type in all samples.", childName)
							pth := strings.Join([]string{"", sampleName, childName, nodeChildName}, "/")
							err := &datasetError{err: msg, path: pth}
							return nil, err
						}
						if categoryField.Class != class {
							msg := "Category class mismatch."
							pth := strings.Join([]string{"", sampleName, childName}, "/")
							err := &datasetError{err: msg, path: pth}
							return nil, err
						}
					}

				} else {
					msg := fmt.Sprintf("Files of type '%s' are unexpected in dataset root.", nodeChild.Type())
					pth := strings.Join([]string{"", sampleName, childName, nodeChildName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
			}

			// Create the node with all found fields. We will add links later.
			schNodes[childName] = &sch.Node{IsSingleton: false, Fields: fields, Links: map[string]*sch.Link{}}
		}

		// Handle links files. We allow at most one links file.
		if len(sampleLinks) > 1 {
			msg := "At most one links file per data sample is allowed."
			pth := strings.Join([]string{"", sampleName}, "/")
			err := &datasetError{err: msg, path: pth}
			return nil, err
		}

		// If a links file is missing, we might have implicit links.
		if len(sampleLinks) == 0 {

			// If we have non-signleton nodes, then we assume a single undirected chain.
			// To construct a graph without links, there must be an empty links file.
			for nodeName, node := range schNodes {
				if node.IsSingleton == false {
					node.Links[nodeName] = &sch.Link{LBound: 1, UBound: 1}
					schUndirected = false
				}
			}

		} else {

			var links *Links
			for _, v := range sampleLinks {
				links = v
			}

			if len(nodeInstanceCount) == 0 {
				msg := "Link file found but no non-singleton nodes."
				pth := strings.Join([]string{"", sampleName}, "/")
				err := &datasetError{err: msg, path: pth}
				return nil, err
			}

			// Check link counts.
			linkCounts := links.getLinkDestinationCounts()
			for key, count := range linkCounts {
				srcNode, ok := schNodes[key.src.Node]
				if ok == false {
					msg := fmt.Sprintf("Link references unknown node '%s'.", key.src.Node)
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
				if srcNode.IsSingleton {
					msg := fmt.Sprintf("Link references singleton node '%s'.", key.src.Node)
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}

				dstNode, ok := schNodes[key.dstNode]
				if ok == false {
					msg := fmt.Sprintf("Link references unknown node '%s'.", key.dstNode)
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
				if dstNode.IsSingleton {
					msg := fmt.Sprintf("Link references singleton node '%s'.", key.dstNode)
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}

				// All link counts are merged together over the entire sample.
				link, ok := srcNode.Links[key.dstNode]
				if ok {
					if count < link.LBound {
						link.LBound = count
					}
					if count > link.UBound {
						link.UBound = count
					}
				} else {
					srcNode.Links[key.dstNode] = &sch.Link{LBound: count, UBound: count}
				}
			}

			// Check if any link index overflows the number of node instances.
			maxNodeInstances := links.getMaxIndicesPerNode()
			for nodeName, count := range maxNodeInstances {
				if count >= nodeInstanceCount[nodeName] {
					msg := fmt.Sprintf("Found link index %d to node with %d instances.", count, nodeInstanceCount[nodeName])
					pth := strings.Join([]string{"", sampleName}, "/")
					err := &datasetError{err: msg, path: pth}
					return nil, err
				}
			}

			// Get referential constraints if needed.
			if schUndirected {
				// If the undirected constraint has been violated
				// at least once, there is no point to check more.
				schUndirected = links.IsUndirected()
			}
			if !schFanin {
				// If a fan-in has been detected at any point, there is no point to check more.
				schFanin = links.IsFanin(schUndirected)
			}
			if !schCyclic {
				// If cycles have been detected at any point, the links are not acyclic.
				schCyclic = links.IsCyclic(schUndirected)
			}

		}

		firstSample = false
	}

	// Build and return schema result.
	result := &sch.Schema{
		Nodes:        schNodes,
		Classes:      schClasses,
		IsCyclic:     schCyclic,
		IsUndirected: schUndirected,
		IsFanIn:      schFanin,
	}
	return result, nil
}

func isSuperset(set1, set2 map[string]interface{}) bool {
	for k := range set1 {
		if _, ok := set2[k]; ok == false {
			return false
		}
	}
	return true
}

const defaultRandomStringChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// RandomString returns a random string of given length from a given set of characters.
func RandomString(size int, chars string) string {
	if chars == "" {
		chars = defaultRandomStringChars
	}
	b := make([]byte, size)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func randomVector(dimensions []int) []float64 {
	size := 1
	for i := range dimensions {
		size = size * dimensions[i]
	}

	result := make([]float64, size)
	for i := range result {
		result[i] = rand.Float64()
	}
	return result
}

// GenerateFromSchema is.
func GenerateFromSchema(root string, schema *sch.Schema, sampleNames []string, numNodeInstances int) (*Dataset, error) {

	// Generate classes.
	classes := map[string]*Class{}
	for className, class := range schema.Classes {
		dim := class.Dim.(*sch.ConstDim).Value
		categories := make([]string, dim)
		for i := 0; i < dim; i++ {
			categories[i] = RandomString(16, defaultRandomStringChars)
		}
		classes[className] = &Class{Categories: categories}
	}

	// Generate samples.
	datasetRootChildren := map[string]File{}
	for _, sampleName := range sampleNames {
		nodes := map[string]File{}

		// Generate nodes.
		for nodeName, node := range schema.Nodes {

			if node.IsSingleton {

				var field sch.Field
				for _, v := range node.Fields {
					field = v
					break
				}

				if tensorField, ok := field.(*sch.Tensor); ok {
					// Generate singleton tensor.
					dimensions := make([]int, len(tensorField.Dim))
					for i := range tensorField.Dim {
						dim := tensorField.Dim[i].(*sch.ConstDim)
						dimensions[i] = dim.Value
					}
					data := randomVector(dimensions)
					nodes[nodeName] = &Tensor{Name: nodeName, Dimensions: dimensions, Data: data}

				} else if categoryField, ok := field.(*sch.Category); ok {
					// Generate singleton category.
					categories := make([]string, 1)
					classCategories := classes[categoryField.Class].Categories
					categories[0] = classCategories[rand.Intn(len(classCategories))]
					nodes[nodeName] = &Category{Name: nodeName, Categories: categories}
				}

			} else {

				nodeChildren := map[string]File{}

				for fieldName, field := range node.Fields {

					if tensorField, ok := field.(*sch.Tensor); ok {
						// Generate non-singleton tensor.
						dimensions := make([]int, len(tensorField.Dim)+1)
						dimensions[0] = numNodeInstances
						for i := range tensorField.Dim {
							dim := tensorField.Dim[i].(*sch.ConstDim)
							dimensions[i+1] = dim.Value
						}
						data := randomVector(dimensions)
						nodeChildren[fieldName] = &Tensor{Name: fieldName, Dimensions: dimensions, Data: data}

					} else if categoryField, ok := field.(*sch.Category); ok {
						// Generate non-singleton category.
						categories := make([]string, numNodeInstances)
						classCategories := classes[categoryField.Class].Categories
						for i := range categories {
							categories[i] = classCategories[rand.Intn(len(classCategories))]
						}
						nodeChildren[fieldName] = &Category{Name: fieldName, Categories: categories}
					}
				}

				// Generate the actual node directory.
				nodes[nodeName] = &Directory{Name: nodeName, Children: nodeChildren}
			}
		}

		// Generate links.
		links := map[Link]interface{}{}
		allInstances := map[string][]int{}
		countIn, countOut := map[InstanceID]int{}, map[InstanceID]int{}
		for nodeName, node := range schema.Nodes {
			if node.IsSingleton == false {
				// Generate a random permutation of indices.
				randIndices := make([]int, numNodeInstances)
				for i := range randIndices {
					randIndices[i] = i
				}
				for i := range randIndices {
					j := rand.Intn(len(randIndices))
					t := randIndices[j]
					randIndices[j] = randIndices[i]
					randIndices[i] = t
				}
				allInstances[nodeName] = randIndices
			}
		}
		maxIdxIn := map[destCountKey]int{}

		for nodeName, instances := range allInstances {
			for i := range instances {
				for targetName, link := range schema.Nodes[nodeName].Links {

					lBound := link.LBound
					uBound := link.UBound
					if uBound > numNodeInstances || uBound <= 0 {
						uBound = numNodeInstances
					}
					count := rand.Intn(uBound-lBound+1) + lBound - countOut[InstanceID{Node: nodeName, Index: i}]

					// If there are no links to create, simply skip.
					if count <= 0 {
						continue
					}

					exclude := make([]bool, len(allInstances[targetName]))
					if schema.IsCyclic == false {
						if schema.IsUndirected {
							for j := range exclude {
								if i == j || countIn[InstanceID{Node: targetName, Index: j}] > 0 {
									exclude[j] = true
								}
							}
						} else if targetName == nodeName {
							for j := 0; j <= i; j++ {
								exclude[j] = true
							}
						} else {
							idx := maxIdxIn[destCountKey{src: InstanceID{Node: nodeName, Index: i}, dstNode: targetName}]
							for j := 0; j <= idx; j++ {
								exclude[j] = true
							}
						}
					}
					if schema.IsFanIn == false {
						for j := range exclude {
							maxCount := 1
							if schema.IsUndirected {
								maxCount = 2
							}
							if exclude[j] == false && countIn[InstanceID{Node: targetName, Index: j}] >= maxCount {
								exclude[j] = true
							}
						}
					}

					iterCount := 0
					for j := range exclude {
						if exclude[j] == false {

							src := InstanceID{Node: nodeName, Index: i}
							dst := InstanceID{Node: targetName, Index: j}

							countOut[src]++
							countIn[dst]++
							links[Link{Src: src, Dst: dst}] = nil

							idx := maxIdxIn[destCountKey{src: src, dstNode: targetName}]
							if j > idx {
								maxIdxIn[destCountKey{src: src, dstNode: targetName}] = j
							}

							if schema.IsUndirected {

								countOut[dst]++
								countIn[src]++
								links[Link{Src: dst, Dst: src}] = nil

								idx := maxIdxIn[destCountKey{src: dst, dstNode: nodeName}]
								if j > idx {
									maxIdxIn[destCountKey{src: dst, dstNode: nodeName}] = j
								}
							}
							iterCount++
						}
						if iterCount >= count {
							break
						}
					}
				}
			}
		}

		// Create a links instance if there were non-singleton nodes.
		if len(allInstances) > 0 {
			nodes["links"] = &Links{Name: "links", Links: links}
		}

		// Generate the actual sample directory.
		datasetRootChildren[sampleName] = &Directory{Name: sampleName, Children: nodes}
	}

	// Add classes to the root.
	for k, v := range classes {
		datasetRootChildren[k] = v
	}

	// Generate the dataset.
	result := &Dataset{Root: root, Directory: Directory{Name: root, Children: datasetRootChildren}}
	return result, nil
}
