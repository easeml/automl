'use strict'

import assert from "assert";
import ReaderWriterCloser from "./reader-writer-closer";
import jsnpy from "./util/jsnpy";
import path from "path";
import sch from "./schema";

const LINK_FORMAT = new RegExp("^\\s*[a-z_]*[0-9a-z_]*(/[0-9]+)?\\s+[a-z_]*[0-9a-z_]*(/[0-9]+)?\\s*$");

function DatasetException(message, path="") {
    this.message = message;
    this.path = path;
}

const FILE_TYPES = {
    "directory" : Directory,
    "tensor" : Tensor,
    "category" : Category,
    "links" : Links,
    "class" : Class
};

const TYPE_EXTENSIONS = {
    "tensor" : { "default" : ".ten.npy", "csv" : ".ten.csv"},
    "category" : { "default" : ".cat.txt"},
    "class" : { "default" : ".class.txt"},
    "links" : { "default" : ".links.csv" }
};

const NODE_SOURCE = "SOURCE";
const NODE_SINK = "SINK";

function File(name, fileType, subtype="default") {
    assert(typeof name === "string");
    assert(fileType in FILE_TYPES);
    assert(typeof subtype === "string");
    this.name = name;
    this.fileType = fileType;
    this.subtype = subtype;
}

function Directory(name, children=null) {
    File.call(this, name, "directory");
    children = children || {};

    this.children = children;
}

Directory.prototype = Object.create(File.prototype);
Directory.prototype.constructor = Directory;

function loadDirectory(root, relPath, name, opener, metadataOnly=false, subtype="default") {
    let dirPath = name.length > 0 ? path.join(relPath, name) : relPath;
    let dirlist = opener(root, dirPath, true, true);
    let children = {};

    for (let i = 0; i < dirlist.length; i++) {

        let child = null;
        let childName = dirlist[i];

        let loaderFunction = {
            "tensor" : loadTensor,
            "category" : loadCategory,
            "class" : loadClass,
            "links" : loadLinks,
        }

        for (let fileType in TYPE_EXTENSIONS) {
            for (let subtype in TYPE_EXTENSIONS[fileType]) {
                let ext = TYPE_EXTENSIONS[fileType][subtype]
                if (childName.endsWith(ext)) {
                    childName = childName.slice(0, -ext.length);
                    let loader = loaderFunction[fileType];
                    child = loader(root, dirPath, childName, opener, metadataOnly, subtype);
                }
            }
        }

        if (child === null) {
            child = loadDirectory(root, dirPath, childName, opener, metadataOnly);
        }

        children[childName] = child;

    }

    return new Directory(name, children);
}

function dumpDirectory(self, root, relPath, name, opener) {

    let dirPath = name.length > 0 ? path.join(relPath, name) : relPath;
    dirlist = opener(root, dirPath, true, false);
    for (childName in self.children) {
        let child = self.children[childName];
        switch (child.fileType) {
            case "tensor":
                dumpTensor(child, root, dirPath, childName, opener);
                break;

            case "category":
                dumpCategory(child, root, dirPath, childName, opener);
            break;
            
            case "class":
                dumpClass(child, root, dirPath, childName, opener);
            break;
            
            case "links":
                dumpLinks(child, root, dirPath, childName, opener);
            break;
            
            case "directory":
                dumpDirectory(child, root, dirPath, childName, opener);
            break;
        
            default:
                break;
        }
    }

}

function Dataset(root, directory) {
    this.root = root;
    this.directory = directory;
}

function loadDataset(root, opener, metadataOnly=false) {
    let directory = loadDirectory(root, "", "", opener, metadataOnly);
    return new Dataset(root, directory);
}

function dumpDataset(self, root, opener) {
    dumpDirectory(self.directory, root, "", "", opener);
}

function Tensor(name, dimensions, data=null) {
    File.call(this, name, "tensor");
    this.dimensions = dimensions;
    this.data = data;
}

Tensor.prototype = Object.create(File.prototype);
Tensor.prototype.constructor = Tensor;

function isSuperset(set, subset) {
    for (var elem of subset) {
        if (!set.has(elem)) {
            return false;
        }
    }
    return true;
}

function difference(setA, setB) {
    var _difference = new Set(setA);
    for (var elem of setB) {
        _difference.delete(elem);
    }
    return _difference;
}

function arraysEqual(a, b) {
    if (a === b) return true;
    if (a == null || b == null) return false;
    if (a.length != b.length) return false;
  
    // If you don't care about the order of the elements inside
    // the array, you should sort both arrays here.
  
    for (var i = 0; i < a.length; ++i) {
      if (a[i] !== b[i]) return false;
    }
    return true;
  }

Dataset.prototype.inferSchema = function() {

    let categoryClasses = {};
    let categoryClassSets = {};
    let schCategoryClasses = {};
    let samples = {};

    for (let childName in this.directory.children) {
        let child = this.directory.children[childName];

        if (child.fileType === "directory") {
            // A directory corresponds to a data sample.
            samples[childName] = child;

        } else if (child.fileType === "class") {
            // Collect class file and get its dimensions.
            categoryClasses[childName] = child;
            schCategoryClasses[childName] = new sch.Class(child.categories.length);
            categoryClassSets[childName] = new Set(child.categories);

        } else {
            // We forbid any other file type in the root directory. Maybe we should just ignore.
            throw new DatasetException("Files of type '" + child.fileType + "' are unexpected in dataset root.", ["", childName].join("/"));
        }
    }
    
    let schNodes = {};
    let firstSample = true;
    let linksFileFound = false;
    let schCyclic = false;
    let schFanin = false;
    let schUndirected = true;
    
    // Go through all data samples.
    for (let sampleName in samples) {
        let sample = samples[sampleName];

        // Sort all sample children according to their type.
        let sampleChildren = {};
        for (let k in FILE_TYPES) {
            sampleChildren[k] = {};
        }
        let sampleNodes = new Set();

        for (let childName in sample.children) {
            let child = sample.children[childName];
            assert(child.fileType in FILE_TYPES);

            if (child.fileType == "links") {
                linksFileFound = true;
            } else {
                sampleNodes.add(childName);
            }

            sampleChildren[child.fileType][childName] = child;
        }
        
        // Ensure that either all samples have links files or none of them have.

        if ((Object.keys(sampleChildren["links"]).length > 0) !== linksFileFound) {
            throw new DatasetException("Links file not found in all data samples.", ["", sampleName].join("/"));
        }
        
        // Ensure all samples have the same node names.
        let schemaNodesSet = new Set(Object.keys(schNodes));
        if (firstSample === false) {
            let schemaSampleSuperset = isSuperset(schemaNodesSet, sampleNodes);
            let sampleSchemaSuperset = isSuperset(sampleNodes, schemaNodesSet);

            if ((schemaSampleSuperset && sampleSchemaSuperset) === false) {
                if (schemaSampleSuperset) {
                    let diffSet = difference(schemaNodesSet, sampleNodes);
                    let childName = diffSet.values().next().value;
                    throw new DatasetException("Item expected but not found.", ["", sampleName, childName].join("/"));
                } else if (sampleSchemaSuperset) {
                    let diffSet = difference(sampleNodes, schemaNodesSet);
                    let childName = diffSet.values().next().value;
                    throw new DatasetException("Item found but not expected.", ["", sampleName, childName].join("/"));
                }
            }
        }
        
        // Handle tensor singleton nodes.
        for (let childName in sampleChildren["tensor"]) {
            let child = sampleChildren["tensor"][childName];

            if (firstSample) {
                let field = new sch.Tensor(child.dimensions);
                schNodes[childName] = new sch.Node(true, {"field" : field});
            } else {
                // Verify that the node is the same.
                let node = schNodes[childName];
                if (node.isSingleton === false || Object.keys(node.fields).length > 1 || node.fields["field"].fieldType !== "tensor") {
                    throw new DatasetException("Node '" + childName + "' not the same type in all samples.", ["", sampleName].join("/"));
                } else if (arraysEqual(node.fields["field"].dim, child.dimensions) === false) {
                    throw new DatasetException("Tensor dimensions mismatch.", ["", sampleName, childName].join("/"));
                }
            }
        }
        
        // Handle category singleton nodes.
        for (let childName in sampleChildren["category"]) {
            let child = sampleChildren["category"][childName];

            // Infer class by finding first class to which the node belongs.
            let categoryClass = null;
            for (let className in categoryClassSets) {
                let categorySet = categoryClassSets[className];
                if (child.belongsToSet(categorySet)) {
                    categoryClass = className;
                    break;
                }
            }
            if (categoryClass === null) {
                throw new DatasetException("Category file does not match any class.", ["", sampleName, childName].join("/"));
            }

            if (firstSample) {
                let field = new sch.Category(categoryClass);
                schNodes[childName] = new sch.Node(true, {"field" : field});
            } else {
                // Verify that the node is the same.
                let node = schNodes[childName];
                if (node.isSingleton === false || Object.keys(node.fields).length > 1 || node.fields["field"].fieldType !== "category") {
                    throw new DatasetException("Node '" + childName + "' not the same type in all samples.", ["", sampleName].join("/"));
                } else if (node.fields["field"].categoryClass !== categoryClass) {
                    throw new DatasetException("Category class mismatch.", ["", sampleName, childName].join("/"));
                }
            }
        }
        
        // This counts how many instances each node has. It is used to validate link targets.
        let nodeInstanceCount = {};
        
        // Handle regular non-singleton nodes.
        for (let childName in sampleChildren["directory"]) {
            let child = sampleChildren["directory"][childName];

            let fields = {};
            if (firstSample === false) {
                let node = schNodes[childName];
                fields = node.fields;
                if (node.isSingleton === true) {
                    throw new DatasetException("Node '" + childName + "' not the same type in all samples.", ["", sampleName].join("/"));
                }
            }
            
            // Ensure nodes in all samples have the same children.
            let fieldsSet = new Set(Object.keys(fields));
            let childrenSet = new Set(Object.keys(child.children));
            let setsMatch = isSuperset(fieldsSet, childrenSet) && isSuperset(childrenSet, fieldsSet);
            
            if (firstSample === false && setsMatch === false) {

                let fieldsChildrenSuperset = isSuperset(fieldsSet, childrenSet);
                let childrenFieldsSuperset = isSuperset(childrenSet, fieldsSet);

                if ((fieldsChildrenSuperset && childrenFieldsSuperset) === false) {
                    if (fieldsChildrenSuperset) {
                        let diffSet = difference(fieldsSet, childrenSet);
                        let nodeChildName = diffSet.values().next().value;
                        throw new DatasetException("Item expected but not found.", ["", sampleName, childName, nodeChildName].join("/"));
                    } else if (childrenFieldsSuperset) {
                        let diffSet = difference(childrenSet, fieldsSet);
                        let nodeChildName = diffSet.values().next().value;
                        throw new DatasetException("Item found but not expected.", ["", sampleName, childName, nodeChildName].join("/"));
                    }
                }
            }
            
            // Go through all fields of the non-singleton node.
            for (let nodeChildName in child.children) {
                let nodeChild = child.children[nodeChildName];

                if (nodeChild.fileType === "tensor") {

                    // Verify that all node fields have the same number of instances.
                    let count = nodeChild.dimensions[0];
                    let previousCount = nodeInstanceCount[childName] || count;
                    if (previousCount !== count) {
                        throw new DatasetException("Tensor instance count mismatch.", ["", sampleName, childName, nodeChildName].join("/"));
                    }
                    nodeInstanceCount[childName] = previousCount;

                    if (firstSample) {
                        fields[nodeChildName] = new sch.Tensor(nodeChild.dimensions.slice(1));
                    } else {
                        // Verify that the node is the same.
                        let field = fields[nodeChildName];
                        if (field.fieldType !== "tensor") {
                            throw new DatasetException("Node '" + childName + "' not the same type in all samples.", ["", sampleName, childName, nodeChildName].join("/"));
                        }
                        if (arraysEqual(field.dim, nodeChild.dimensions.slice(1)) === false) {
                            throw new DatasetException("Tensor dimensions mismatch.", ["", sampleName, childName, nodeChildName].join("/"));
                        }
                    }
                
                } else if (nodeChild.fileType == "category") {

                    // Infer class by finding first class to which the node belongs.
                    let categoryClass = null;
                    for (let className in categoryClassSets) {
                        let categorySet = categoryClassSets[className];
                        if (nodeChild.belongsToSet(categorySet)) {
                            categoryClass = className;
                            break;
                        }
                    }
                    if (categoryClass === null) {
                        throw new DatasetException("Category file does not match any class.", ["", sampleName, childName, nodeChildName].join("/"));
                    }
                    
                    // Verify that all node fields have the same number of instances.
                    let count = nodeChild.categories.length;
                    let previousCount = nodeInstanceCount[childName] || count;
                    if (previousCount !== count) {
                        throw new DatasetException("Category instance count mismatch.", ["", sampleName, childName, nodeChildName].join("/"));
                    }
                    nodeInstanceCount[childName] = previousCount;

                    if (firstSample) {
                        fields[nodeChildName] = new sch.Category(categoryClass);
                    } else {
                        // Verify that the node is the same.
                        let field = fields[nodeChildName];
                        if (field.fieldType !== "category") {
                            throw new DatasetException("Node '" + childName + "' not the same type in all samples.", ["", sampleName, childName, nodeChildName].join("/"));
                        }
                        if (field.categoryClass !== categoryClass) {
                            throw new DatasetException("Category class mismatch.", ["", sampleName, childName, nodeChildName].join("/"));
                        }
                    }
                
                } else {
                    // We forbid any other file type in the node directory. Maybe we should just ignore.
                    throw new DatasetException("Files of type '" + nodeChild.fileType + "' are unexpected in node directory.", ["", sampleName, childName, nodeChildName].join("/"));
                }
            }
                
            // Create the node with all found fields. We will add links later.
            schNodes[childName] = new sch.Node(false, fields);
        }

        // Handle links files. We allow at most one links file.
        if (Object.keys(sampleChildren["links"]).length > 1) {
            throw new DatasetException("At most one links file per data sample is allowed.", ["", sampleName].join("/"));
        }
        
        // If a links file is missing, we might have implicit links.
        if (Object.keys(sampleChildren["links"]).length === 0) {
            
            // If we have non-signleton nodes, then we assume a single directed chain.
            // To construct a graph without links, there must be an empty links file.
            for (let nodeName in schNodes) {
                let node = schNodes[nodeName];
                if (node.isSingleton === false) {
                    node.links[nodeName] = new sch.Link(1);
                    schUndirected = false
                }
            }
        
        } else {

            let links = Object.values(sampleChildren["links"])[0];

            if (Object.keys(nodeInstanceCount).length  === 0) {
                throw new DatasetException("Link file found but no non-singleton nodes.", ["", sampleName].join("/"));
            }

            // Check link counts.
            let linkCounts = links.getLinkDestinationCounts();
            for (let key in linkCounts) {

                let [srcNodeId, dstNodeName] = key.split(" ");
                let srcNodeName = loadInstnceId(srcNodeId).node;
                let count = linkCounts[key];

                let srcNode = schNodes[srcNodeName] || null;
                if (srcNode === null) {
                    throw new DatasetException("Link references unknown node '" + dstNodeName + "'.", ["", sampleName].join("/"));
                }
                if (srcNode.isSingleton) {
                    throw new DatasetException("Link references singleton node '" + dstNodeName + "'.", ["", sampleName].join("/"));
                }

                let dstNode = schNodes[dstNodeName] || null;
                if (dstNode === null) {
                    throw new DatasetException("Link references unknown node '" + dstNodeName + "'.", ["", sampleName].join("/"));
                }
                if (dstNode.isSingleton) {
                    throw new DatasetException("Link references singleton node '" + dstNodeName + "'.", ["", sampleName].join("/"));
                }
                
                // All link counts are merged together over the entire sample.
                let link = srcNode.links[dstNodeName] || new sch.Link(count);
                if (count < link.dim[0]) {
                    link.dim[0] = count;
                }
                if (count > link.dim[1]) {
                    link.dim[1] = count;
                }
                srcNode.links[dstNodeName] = link;
            }
            
            // Check if any link index overflows the number of node instances.
            let maxNodeIndices = links.getMaxIndicesPerNode();
            for (let nodeName in maxNodeIndices) {
                let count = maxNodeIndices[nodeName];
                if (count >= nodeInstanceCount[nodeName]) {
                    throw new DatasetException("Found link index " + count + " to node with " + nodeInstanceCount[nodeName] + " instances.", ["", sampleName].join("/"));
                }
            }

            // Get referential constraints if needed.
            if (schUndirected) {
                // If the undirected constraint has been violated
                // at least once, there is no point to check more.
                schUndirected = links.isUndirected();
            }
            if (schFanin === false) {
                // If a fan-in has been detected at any point, there is no point to check more.
                schFanin = links.isFanin(schUndirected);
            }
            if (schCyclic === false) {
                // If cycles have been detected at any point, the links are not acyclic.
                schCyclic = links.isCyclic(schUndirected);
            }
        }

        firstSample = false;
    }
    
    // Build and return schema result.
    return new sch.Schema(schNodes, schCategoryClasses, schCyclic, schUndirected, schFanin);
}

function randomString(size, chars="abcdefghijklmnopqrstuvwxyz0123456789") {
    var text = "";
  
    for (var i = 0; i < size; i++) {
      text += chars.charAt(Math.floor(Math.random() * chars.length));
    }
  
    return text;
}

function randomIndices(size) {

    let list = Array.from(Array(size).keys());

    for (let i = 0; i < size; i++) {
        let j = Math.floor(Math.random() * size);
        let t = list[i];
        list[i] = list[j];
        list[j] = t;
    }

    return list;
}

function generateFromSchema(root, schema, numSamples = 10, numNodeInstances = 10) {
    assert(schema.isVariable() == false);

    // Generate classes.
    let classes = {};
    for (let className in schema.categoryClasses) {
        let categoryClass = schema.categoryClasses[className];
        assert(Number.isInteger(categoryClass.dim));
        let categories = [];
        for (let i = 0; i < categoryClass.dim; i++) {
            categories.push(randomString(16));
        }
        classes[className] = new Class(className, categories);
    }

    // Generate samples.
    let samples = {};
    for (let s = 0; s < numSamples; s++) {

        let sampleName = randomString(16);
        let nodes = {};

        // Generate nodes.
        for (let nodeName in schema.nodes) {
            let node = schema.nodes[nodeName];

            if (node.isSingleton) {

                let field = Object.values(node.fields)[0];
                assert(["tensor", "category"].indexOf(field.fieldType) >= 0);

                // Generate singleton tensor.
                if (field.fieldType === "tensor") {

                    let size = 1;
                    for (let i = 0; i < field.dim.length; i++) {
                        size *= field.dim[i];
                    }
                    assert(Number.isInteger(size));

                    let data = Array(size).fill(0).map(() => Math.random());
                    nodes[nodeName] = new Tensor(nodeName, field.dim, data);
                
                // Generate singleton category.
                } else if (field.fieldType === "category") {
                    let choices = classes[field.categoryClass].categories;
                    let index = Math.floor(Math.random() * choices.length);
                    let categories = [choices[index]];
                    nodes[nodeName] = new Category(nodeName, categories);
                }
            
            } else {

                let nodeChildren = {};

                for (let fieldName in node.fields) {

                    let field = node.fields[fieldName];
                    assert(["tensor", "category"].indexOf(field.fieldType) >= 0);

                    // Generate non-singleton tensor.
                    if (field.fieldType === "tensor") {

                        let size = 1;
                        let dim = [numNodeInstances].concat(field.dim);
                        for (let i = 0; i < dim.length; i++) {
                            size *= dim[i];
                        }
                        assert(Number.isInteger(size));

                        let data = Array(size).fill(0).map(() => Math.random());
                        nodeChildren[fieldName] = new Tensor(fieldName, dim, data);
                    
                    // Generate non-singleton category.
                    } else if (field.fieldType === "category") {
                        let choices = classes[field.categoryClass].categories;

                        let categories = [];
                        for (let i = 0; i < numNodeInstances; i++) {
                            let index = Math.floor(Math.random() * choices.length);
                            categories.push(choices[index]);
                        }
                        nodeChildren[fieldName] = new Category(nodeName, categories);
                    }
                }

                // Generate the actual node directory.
                nodes[nodeName] = new Directory(nodeName, nodeChildren);
            }
        }

        // Generate links.
        let links = new Set();
        let allInstances = {};
        let [countIn, countOut] = [{}, {}];

        for (let nodeName in schema.nodes) {
            let node = schema.nodes[nodeName];

            if (node.isSingleton === false) {
                allInstances[nodeName] = randomIndices(numNodeInstances);
                for (let i = 0; i < numNodeInstances; i++) {
                    countIn[(new InstanceId(nodeName, i)).dump()] = 0;
                    countOut[(new InstanceId(nodeName, i)).dump()] = 0;
                }
            }
        }
        let maxIdxIn = {};
        
        // TODO: Fix this. Take advantage of SOURCE and SINK.
        for (let node in allInstances) {
            let instances = allInstances[node];

            for (let i = 0; i < instances.length; i++) {
                for (let target in schema.nodes[node].links) {
                    let link = schema.nodes[node].links[target];

                    let l_bound = link.dim[0];
                    let u_bound = link.dim[1];
                    if (u_bound === "inf" || u_bound > numNodeInstances) {
                        u_bound = numNodeInstances;
                    }
                    assert(l_bound <= u_bound);

                    let count = Math.floor(Math.random() * (u_bound - l_bound + 1)) - l_bound;
                    count -= countOut[(new InstanceId(node, i)).dump()];

                    // If there are no links to create, simply skip.
                    if (count <= 0) {
                        continue;
                    }

                    let candidates = Array(allInstances[target].length).fill(true);

                    if (schema.cyclic === false) {
                        if (schema.undirected) {
                            for (let x = 0; x < candidates.length; x++) {
                                if (x === i || countIn[(new InstanceId(target, x)).dump()] !== 0) {
                                    candidates[x] = false;
                                }
                            }

                        } else if (target === node) {
                            for (let x = 0; x < candidates.length; x++) {
                                if (x <= i) {
                                    candidates[x] = false;
                                }
                            }

                        } else {

                            let key = (new InstanceId(node, i)).dump() + " " + target;
                            let idx = maxIdxIn[key] || -1;

                            for (let x = 0; x < candidates.length; x++) {
                                if (x <= idx) {
                                    candidates[x] = false;
                                }
                            }
                        }
                    }

                    if (schema.fanin === false) {
                        for (let x = 0; x < candidates.length; x++) {
                            if (countIn[(new InstanceId(target, x)).dump()] !== 0) {
                                candidates[x] = false;
                            }
                        }
                    }
                    
                    // assert(len(candidates) >= count)
                    for (let j = 0; j < candidates.length; j++) {

                        if (candidates[j]) {

                            let nodeInstance = new InstanceId(node, i);
                            let targetInstance = new InstanceId(target, j);

                            let nodeInstanceKey = nodeInstance.dump();
                            let targetInstanceKey = targetInstance.dump();

                            countOut[nodeInstanceKey] += 1;
                            countIn[targetInstanceKey] += 1;
                            links.add(new Link(nodeInstance, targetInstance));

                            let idx = maxIdxIn[nodeInstanceKey + " " + target] || 0;
                            if (j > idx) {
                                maxIdxIn[nodeInstanceKey + " " + target] = j;
                            }

                            if (schema.undirected) {
                                countOut[targetInstanceKey] += 1;
                                countIn[nodeInstanceKey] += 1;
                                links.add(new Link(targetInstance, nodeInstance));

                                let idx = maxIdxIn[targetInstanceKey + " " + node] || 0;
                                if (i > idx) {
                                    maxIdxIn[nodeInstanceKey + " " + target] = i;
                                }
                            }

                            count -= 1;
                            if (count <= 0) {
                                break;
                            }

                        }
                    }
                }
            }
        }
        
        // Conect nodes without an incoming link to the SOURCE and
        // without an outgoing one to the SINK.
        // for ((nodeName, i), count) in countIn.items():
        //     if count == 0:
        //         links.add(Link(NODE_SOURCE, None, nodeName, i))
        // for ((nodeName, i), count) in countOut.items():
        //     if count == 0:
        //         links.add(Link(nodeName, i, NODE_SINK, None))

        // Create a links instance if there were non-singleton nodes.
        if (Object.keys(allInstances).length > 0) {
            let linksMap = new Map();
            for (let l of links) {
                linksMap.set(l.dump(), l);
            }

            nodes["links"] = new Links("links", linksMap);
        }

        // Generate the actual sample directory.
        samples[sampleName] = new Directory(sampleName, nodes);
    }
    
    // Generate the dataset.
    let allChildren = Object.assign({}, samples, classes);
    return new Dataset(root, new Directory("", allChildren));
}

function loadTensor(root, relPath, name, opener, metadataOnly=false, subtype="default") {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["tensor"][subtype]);
    let reader = opener(root, filePath, false, true);

    if (subtype === "default") {
        let npyReader = new jsnpy.NpyReader(reader);

        let data = null;
        if (metadataOnly === false) {
            data = npyReader.read(false);
        }
        reader.close();

        if (npyReader.dtype !== "f8") {
            throw new DatasetException("Tensor datatype must be float64.", filePath);
        }

        return new Tensor(name, npyReader.shape, data);

    } else if (subtype === "csv") {

        let lines = reader.readLines();
        let data = [];
        let lineLength = null;
        for (let i = 0; i < lines.length; i++) {
            let values = lines[i].split(",");
            if (lineLength !== null && lineLength !== values.length){
                throw new DatasetException("Each row of the CSV file must have the same number of elements.", filePath);
            }
            data = data.concat(values.map(x => parseFloat(x)));
            lineLength = values.length;
        }
        data = new Float64Array(data);
        let shape = lines.length > 1 ? [lines.length, lineLength] : [lineLength];

        return new Tensor(name, shape, data);

    } else {
        throw new DatasetException("Unknown tensor subtype '" + subtype + "'.", filePath);
    }
}

function dumpTensor(self, root, relPath, name, opener) {

    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["tensor"][self.subtype]);
    let writer = opener(root, filePath, false, false);

    if (self.subtype === "default") {

        let npyWriter = new jsnpy.NpyWriter(writer, self.dimensions, "f8");
        npyWriter.write(self.data);

    } else if (self.subtype === "csv") {

        let numLines = self.dimensions.length > 1 ? self.dimensions[0] : 1;
        let lineLength = self.dimensions.length > 1 ? self.dimensions[1] : self.dimensions[0];
        let lines = [];
        let pos = 0;

        for (let i = 0; i < numLines; i++) {
            let line = [];
            for (let j = 0; j < lineLength; j++) {
                line.push(self.data[pos]);
                pos++;
            }
            lines.push(line.join(","));
        }
        writer.writeLines(lines);


    } else {
        throw new DatasetException("Unknown tensor subtype '" + subtype + "'.", filePath);
    }

}

function Category(name, categories) {
    File.call(this, name, "category");
    this.categories = categories;
}

Category.prototype = Object.create(File.prototype);
Category.prototype.constructor = Category;

function loadCategory(root, relPath, name, opener, metadataOnly=false, subtype="default") {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["category"][subtype]);
    let reader = opener(root, filePath, false, true);
    let lines = reader.readLines();

    let categories = [];
    for (let i = 0; i < lines.length; i++) {
        let l = lines[i].trim();
        if (l.length > 0) {
            categories.push(l);
        }
    }

    return new Category(name, categories);
}

function dumpCategory(self, root, relPath, name, opener) {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["category"][self.subtype]);
    let writer = opener(root, filePath, false, false);
    writer.writeLines(self.categories);
}

Category.prototype.belongsToSet = function(categorySet) {
    assert(categorySet instanceof Set);
    for (let i = 0; i < this.categories.length; i++) {
        if (categorySet.has(this.categories[i]) === false) {
            return false;
        }
    }
    return true;
}

function InstanceId(node, index) {
    assert(typeof node === "string");
    assert(Number.isInteger(index));
    this.node = node;
    this.index = index;
}

function loadInstnceId(input) {
    let splits = input.split("/");
    let node = splits[0];
    let index = null;
    if (splits.length == 2) {
        index = parseInt(splits[1]);
    }
    return new InstanceId(node, index);
}

InstanceId.prototype.dump = function() {
    if (this.index !== null) {
        return this.node + "/" + this.index;
    } else {
        return this.node;
    }
}

function Link(src, dst) {
    assert(src instanceof InstanceId);
    assert(dst instanceof InstanceId);
    this.src = src;
    this.dst = dst;
}

function loadLink(input) {

    if (LINK_FORMAT.test(input) === false) {
        throw new DatasetException("Link must have a source and a destination separated by whitespace.");
    }

    let [src, dst] = input.split(" ");
    return new Link(loadInstnceId(src), loadInstnceId(dst));
}

Link.prototype.dump = function() {
    return this.src.dump() + " " + this.dst.dump();
}

Link.prototype.getReverse = function() {
    return new Link(new InstanceId(this.dst.node, this.dst.index), new InstanceId(this.src.node, this.src.index));
}

function Links(name, links) {
    File.call(this, name, "links");
    assert(links instanceof Map);
    this.links = links;
}

Links.prototype = Object.create(File.prototype);
Links.prototype.constructor = Links;

function loadLinks(root, relPath, name, opener, metadataOnly=false, subtype="default") {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["links"][subtype]);
    let reader = opener(root, filePath, false, true);
    let lines = reader.readLines();

    let links = new Map();
    for (let i = 0; i < lines.length; i++) {
        let key = lines[i].trim().replace(/\s+/g," ");
        if (key.length > 0) {
            let link = loadLink(key);
            links.set(key, link);
        }
    }

    return new Links(name, links);
}

function dumpLinks(self, root, relPath, name, opener) {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["links"][self.subtype]);
    let writer = opener(root, filePath, false, false);
    let lines = Array.from(self.links.keys());
    writer.writeLines(lines);
}

Links.prototype.adjacencyMap = function(nodes) {
    assert(nodes instanceof Set);
    nodes = new Set(nodes);

    // Build the adjacency map for the given set of nodes.
    let adjacency = new Map();
    for (let node of nodes) {
        adjacency.set(node, []);
    }

    for (let link of this.links.values()) {
        let src = link.src.dump();
        let dst = link.dst.dump();
        adjacency.get(src).push(dst);
        nodes.delete(dst);
    }

    // Make sure that the implicit source node points to all nodes that have no incoming links
    // and that all nodes without an outgoing link point to the sink.
    for ([node, adj] in adjacency.entries()) {
        if (node != NODE_SOURCE && adj.length == 0) {
            adj.append(NODE_SINK);
        }
    }
    
    adjacency[NODE_SOURCE] = Array.from(nodes);

    return adjacency;
}

Links.prototype.isUndirected = function() {

    for (let link of this.links.values()) {
        let reversedLinkKey = link.getReverse().dump();
        if (this.links.has(reversedLinkKey) === false) {
            return false;
        }
    }
    return true;
}

Links.prototype.isFanin = function(undirected) {

    let dstNodes = new Map();

    for (let link of this.links.values()) {
        // If a node has more than one incoming link, this constitutes a fan-in in a directed graph.
        // In an undirected graph, a fan-in happens when a node is connected to more than 2 nodes.
        let dst = link.dst.dump();
        let count = dstNodes.has(dst) ? dstNodes.get(dst) : 0;
        if ((undirected && count >= 2) || (!undirected && count >= 1)) {
            return true;
        }
        dstNodes.set(dst, count + 1);
    }

    return false;

}

Links.prototype.isCyclic = function(undirected) {
    // This is a set of unvisited nodes.
    let nodes = new Set();

    // Build adjacency list for each node.
    let adjacency = new Map();
    for (let link of this.links.values()) {
        let src = link.src.dump();
        let dst = link.dst.dump();
        nodes.add(src);
        nodes.add(dst);

        if (adjacency.has(src) === false) {
            adjacency.set(src, []);
        }
        adjacency.get(src).push(dst);
    }
    
    // The algorithm differs for undirected and directed graphs.
    if (undirected) {
        
        while (nodes.size > 0) {

            // Get arbitrary unvisited node.
            let x = nodes.values().next().value;
            nodes.delete(x);

            // For undirected graphs, we need to remember the parent x.
            let stack = []
            let adj = adjacency.get(x);
            for (let i = 0; i < adj.length; i++) {
                stack.push([x, adj[i]]);
            }

            while (stack.length > 0) {

                // Get the node and its parent.
                let [parent, x] = stack.pop();

                // A node is missing from nodes only if it is visited.
                if (nodes.has(x) === false) {
                    return true;
                } else {
                    nodes.delete(x);
                }

                // In undirected graphs, all edges are bidirectional. We don't count this as a cycle.
                let adj = adjacency.get(x);
                for (let i = 0; i < adj.length; i++) {
                    if (adj[i] !== parent) {
                        stack.push([x, adj[i]]);
                    }
                }
            }
        }
    
    } else {
        while (nodes.size > 0) {

            // We keep a set of nodes that are ancestors in the DFS tree.
            let ancestors = new Set();

            // Get arbitrary unvisited node and push it to the stack.
            let x = nodes.values().next().value;
            let stack = [x];

            while (stack.length > 0) {
                
                // We encounter each node twice.
                let x = stack.slice(-1)[0];

                // If it is not in ancestors, then this is a first encounter.
                if (ancestors.has(x) === false) {

                    // Mark node as visited and add it to active ancestors.
                    nodes.delete(x);
                    ancestors.add(x);
                    let adj = [];
                    if (adjacency.has(x)) {
                        adj = adjacency.get(x);
                    }

                    // A cycle is detected if we find a back edge (i.e. edge pointing to an ancestor).
                    for (let i = 0; i < adj.length; i++) {
                        if (ancestors.has(adj[i])) {
                            return true;
                        }
                    }

                    // Add all adjacent nodes to the stack.
                    for (let i = 0; i < adj.length; i++) {
                        if (nodes.has(adj[i])) {
                            stack.push(adj[i]);
                        }
                    }
                
                } else {

                    // Since we encountered x the second time, we can pop it
                    // from the stack and remove it from active ancestors.
                    stack.pop();
                    ancestors.delete(x);
                
                }
            }
        }
    }

    // No cycle was found.
    return false;
}

Links.prototype.getLinkDestinationCounts = function() {
    let counter = {};
    for (let link of this.links.values()) {
        let key = link.src.dump() + " " + link.dst.node;
        let count = counter[key] || 0;
        counter[key] = count + 1;
    }
    return counter;
}

Links.prototype.getMaxIndicesPerNode = function() {
    let maxNodeIndices = {};
    for (let link of this.links.values()) {
        let maxIndexSrc = maxNodeIndices[link.src.node] || 0;
        if (link.src.index > maxIndexSrc) {
            maxNodeIndices[link.src.node] = link.src.index;
        }

        let maxIndexDst = maxNodeIndices[link.dst.node] || 0;
        if (link.dst.index > maxIndexDst) {
            maxNodeIndices[link.dst.node] = link.dst.index;
        }
    }
    return maxNodeIndices;
}

function Class(name, categories) {
    File.call(this, name, "class");
    this.categories = categories;
}

Class.prototype = Object.create(File.prototype);
Class.prototype.constructor = Class;

function loadClass(root, relPath, name, opener, metadataOnly=false, subtype="default") {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["class"][subtype]);
    let reader = opener(root, filePath, false, true);
    let lines = reader.readLines();

    let categories = []
    for (let i = 0; i < lines.length; i++) {
        let l = lines[i].trim();
        if (l.length > 0) {
            categories.push(l);
        }
    }

    let categoriesSet = new Set(categories);
    if (categories.length !== categoriesSet.size) {
        throw new DatasetException("Class file contains duplicate entries.", filePath);
    }

    return new Class(name, categories);
}

function dumpClass(self, root, relPath, name, opener) {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["class"][self.subtype]);
    let writer = opener(root, filePath, false, false);
    writer.writeLines(self.categories);
}

export default {
    "FILE_TYPES" : FILE_TYPES,
    "TYPE_EXTENSIONS" : TYPE_EXTENSIONS,
    "load" : loadDataset,
    "dump" : dumpDataset,
    "generateFromSchema": generateFromSchema,
    "Dataset" : Dataset,
    "Directory" : Directory,
    "Tensor" : Tensor,
    "Category" : Category,
    "InstanceId" :InstanceId,
    "Link" : Link,
    "Links" : Links,
    "Class" : Class,
    "DatasetException" : DatasetException,
    "ReaderWriterCloser" : ReaderWriterCloser,
};
