(function(){function r(e,n,t){function o(i,f){if(!n[i]){if(!e[i]){var c="function"==typeof require&&require;if(!f&&c)return c(i,!0);if(u)return u(i,!0);var a=new Error("Cannot find module '"+i+"'");throw a.code="MODULE_NOT_FOUND",a}var p=n[i]={exports:{}};e[i][0].call(p.exports,function(r){var n=e[i][1][r];return o(n||r)},p,p.exports,r,e,n,t)}return n[i].exports}for(var u="function"==typeof require&&require,i=0;i<t.length;i++)o(t[i]);return o}return r})()({1:[function(require,module,exports){
'use strict';

var assert = require("assert");
var ReaderWriterCloser = require('./reader-writer-closer');
var jsnpy = require('./jsnpy');
var path = require('path');
var sch = require('./schema');

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
    "tensor" : ".ten.npy",
    "category" : ".cat.txt",
    "class" : ".class.txt",
    "links" : ".links.csv"
};

const NODE_SOURCE = "SOURCE";
const NODE_SINK = "SINK";

function File(name, fileType) {
    assert(typeof name === "string");
    assert(fileType in FILE_TYPES);
    this.name = name;
    this.fileType = fileType;
}

function Directory(name, children=null) {
    File.call(this, name, "directory");
    children = children || {};

    this.children = children;
}

Directory.prototype = Object.create(File.prototype);
Directory.prototype.constructor = Directory;

function loadDirectory(root, relPath, name, opener, metadataOnly=false) {
    let dirPath = name.length > 0 ? path.join(relPath, name) : relPath;
    let dirlist = opener(root, dirPath, true, true);
    let children = {};

    for (let i = 0; i < dirlist.length; i++) {

        let child = null;
        let childName = dirlist[i];

        if (childName.endsWith(TYPE_EXTENSIONS["tensor"])) {
            childName = childName.slice(0, -TYPE_EXTENSIONS["tensor"].length);
            child = loadTensor(root, dirPath, childName, opener, metadataOnly);

        } else if (childName.endsWith(TYPE_EXTENSIONS["category"])) {
            childName = childName.slice(0, -TYPE_EXTENSIONS["category"].length);
            child = loadCategory(root, dirPath, childName, opener, metadataOnly);

        } else if (childName.endsWith(TYPE_EXTENSIONS["class"])) {
            childName = childName.slice(0, -TYPE_EXTENSIONS["class"].length);
            child = loadClass(root, dirPath, childName, opener, metadataOnly);

        } else if (childName.endsWith(TYPE_EXTENSIONS["links"])) {
            childName = childName.slice(0, -TYPE_EXTENSIONS["links"].length);
            child = loadLinks(root, dirPath, childName, opener, metadataOnly);

        } else {
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
            
            // If we have non-signleton nodes, then we assume a single chain.
            // To construct a graph without links, there must be an empty links file.
            for (let nodeName in schNodes) {
                let node = schNodes[nodeName];
                if (node.isSingleton === false) {
                    node.links[nodeName] = new sch.Link(1);
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

function loadTensor(root, relPath, name, opener, metadataOnly=false) {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["tensor"]);
    let reader = opener(root, filePath, false, true);
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
}

function dumpTensor(self, root, relPath, name, opener) {

    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["tensor"]);
    let writer = opener(root, filePath, false, false);
    let npyWriter = new jsnpy.NpyWriter(writer, self.dimensions, "f8");
    npyWriter.write(self.data);

}

function Category(name, categories) {
    File.call(this, name, "category");
    this.categories = categories;
}

Category.prototype = Object.create(File.prototype);
Category.prototype.constructor = Category;

function loadCategory(root, relPath, name, opener, metadataOnly=false) {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["category"]);
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
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["category"]);
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

function loadLinks(root, relPath, name, opener, metadataOnly=false) {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["links"]);
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
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["links"]);
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

function loadClass(root, relPath, name, opener, metadataOnly=false) {
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["class"]);
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
    let filePath = path.join(relPath, name + TYPE_EXTENSIONS["class"]);
    let writer = opener(root, filePath, false, false);
    writer.writeLines(self.categories);
}

module.exports = {
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

},{"./jsnpy":4,"./reader-writer-closer":11,"./schema":12,"assert":5,"path":9}],2:[function(require,module,exports){
'use strict';

var schema = require("./schema")
var dataset = require("./dataset")

module.exports = {
    "schema" : schema,
    "dataset" : dataset,
}

},{"./dataset":1,"./schema":12}],3:[function(require,module,exports){
/**
 * iterative-permutation.js
 * 
 * An iterative form of heap's algorithm. This emulates the algorithm by
 * encoding the program stack in the `stack` variable.  It iteratively unrolls
 * this as new entries are needed.
 * 
 * See https://en.wikipedia.org/wiki/Heap's_algorithm
 *
 * License: MIT
 * Author: Brian Card
 */
 
 /**
  * Creates a new Permutation of the given array x. Use the `next` method
  * to get a new permutation and the `hasNext` to check to see if there
  * are any left.
  * 
  * @param {Array} x an array of two or more elements
  */
 function Permutation(x) {
    this.maxIterations = factorial(x.length);
    this.iterations = 0;
    this.x = x;
    this.n = x.length;
    this.stack = [];
    for (var i=this.n; i>0; i--) {
      this.stack.push({n:i, index:0});
    }
 }
 
 /**
  * Returns the next element in the permutation.  This will return a copy
  * of the array with the elements that are passed in, you are free to modify
  * this array.  This may be called after `hasNext` returns false, it will 
  * repeat the permutation sequence again.
  * 
  * @return an array of elements that are swapped to represent a new ordering
  */
 Permutation.prototype.next = function () {
   this.iterations++;
   return this.doNext();
 };
  
 // helper to perform the next calculation, separated out for clairity 
 Permutation.prototype.doNext = function () {
   var s = this.stack.pop();
   var skipSwap = false;
 
   while(s.n !== 1) {
     if (!skipSwap) {
       if (s.n % 2 === 0) {
         this.swap(s.index, s.n-1);
       } else {
         this.swap(0, s.n-1);
       }
       s.index++;
     }
     
     if (s.index < s.n) {
       this.stack.push(s);
       this.stack.push({"n":s.n-1, index:0});
       skipSwap = true;
     }
     
     s = this.stack.pop();
   }
     
   return this.x.slice(0);
  };
  
  // swaps two elements
  Permutation.prototype.swap = function (i, j) {
    var tmp = this.x[i];
    this.x[i] = this.x[j];
    this.x[j] = tmp;
  };
  
  /**
   * Returns `true` if there are more permutations to generate, `false`
   * if all permutations have been exhausted.
   */
  Permutation.prototype.hasNext = function () {
    return this.iterations < this.maxIterations;
  };
  
  /**
   * Returns the total number of permutations avaiable, which is
   * n! for a set of length n.
   */
  Permutation.prototype.getTotal = function () {
    return this.maxIterations;
  };
  
  function factorial(num) {
    var result = num;
    while (num > 1) {
      num--;
      result = result * num;
    }
    return result;
  }
  
  module.exports = Permutation;
  
},{}],4:[function(require,module,exports){
"use strict";

var assert = require("assert");
var ReaderWriterCloser = require("./reader-writer-closer");

const data_types = ["f8", "f4", "i8", "i4", "i2", "i1"];
const bytes_per_element = {
    "f8" : 8,
    "f4" : 4,
    "i8" : 8,
    "i4" : 4,
    "i2" : 2,
    "i1" : 1,
};

function NpyWriter(writer, shape, dtype, column_major = false, big_endian=false, version = 1) {
    assert(writer instanceof ReaderWriterCloser)
    assert(data_types.indexOf(dtype) > -1);

    this.writer = writer;
    this.shape = shape;
    this.dtype = dtype;
    this.column_major = column_major;
    this.big_endian = big_endian;
    this.version = version;

    let pos = write_header(writer, shape, dtype, version, column_major, big_endian);
    this.pos = pos;
}

function NpyReader(reader) {
    assert(reader instanceof ReaderWriterCloser)

    let header = read_header(reader);

    this.reader = reader;
    this.shape = header.shape;
    this.dtype = header.dtype;
    this.column_major = header.column_major;
    this.big_endian = header.big_endian;
    this.version = header.version;
    this.pos = header.pos;
}

function write_header(writer, shape, dtype, version=1, column_major=false, big_endian=false) {
    assert(writer instanceof ReaderWriterCloser);
    assert(version in [1,2]);

    // Assemble all parameters.
    const magicString = "\x93NUMPY"
    const versionString = (version == 1) ? "\x01\x00" : "\x02\x00";
    const descrString = (big_endian ? ">" : "<") + dtype;
    const shapeString = "(" + String(shape.join(",")) + "," + ")";
    const fortranString = column_major ? "True" : "False";

    // Assemble the header.
    const header = "{'descr': '" + descrString + "', 'fortran_order': " + fortranString +
        ", 'shape': " + shapeString + ", }";
    
    // Compute the padding.
    const lengthBytes = (version === 1) ? 2 : 4;
    const unpaddedLength = header.length;
    const padMul = (version === 1) ? 16 : 16;
    const padLength = (padMul - unpaddedLength % padMul) % padMul;
    const padding = " ".repeat(padLength);
    const headerLength = unpaddedLength + padLength;
    const totalHeaderLength = magicString.length + versionString.length + lengthBytes + headerLength;
    assert(headerLength % padMul === 0);
    
    // Build the array buffer.
    const buffer = new ArrayBuffer(totalHeaderLength);
    const view = new DataView(buffer);
    let pos = 0;

    // Write the magic string and version.
    pos = writeStringToDataView(view, magicString + versionString, pos);

    // Write header length.
    if (version === 1) {
        view.setUint16(pos, headerLength, true);
    } else {
        view.setUint32(pos, headerLength, true);
    }
    pos += lengthBytes;

    // Write header.
    pos = writeStringToDataView(view, header + padding, pos);

    // Write the buffer.
    writer.write(buffer, 0, totalHeaderLength, 0);

    return totalHeaderLength;
}

function read_header(reader) {
    assert(reader instanceof ReaderWriterCloser);

    // Build a buffer for the magic string and version.
    const magicStringBuffer = new ArrayBuffer(10);

    // Read the magic string and version.
    reader.read(magicStringBuffer, 0, 8, 0);
    const magicStringView = new DataView(magicStringBuffer, 0, 6);
    const magicString = readDataViewAsString(magicStringView);
    if (magicString !== "\x93NUMPY") {
        throw new Error("The given file is not a valid NUMPY file.");
    }
    const versionView = new DataView(magicStringBuffer, 0, 8);
    const [versionMajor, versionMinor] = [versionView.getUint8(6), versionView.getUint8(7)];
    if ((versionMajor in [1,2]) === false || versionMinor !== 0) {
        throw new Error("Unknown NUMPY file version " + versionMajor + "." + versionMinor);
    }

    // Read header size.
    const lengthBytes = (versionMajor === 1) ? 2 : 4;
    const lengthBuffer = new ArrayBuffer(lengthBytes);
    reader.read(lengthBuffer, 0, lengthBytes, 8);
    const lengthView = new DataView(lengthBuffer);
    const headerLength = (versionMajor === 1) ? lengthView.getUint16(0, true) : lengthView.getUint32(0, true);

    // Read the header.
    const headerDictLength = headerLength - lengthBytes - 8;
    const headerBuffer = new ArrayBuffer(headerDictLength);
    reader.read(headerBuffer, 0, headerDictLength, lengthBytes + 8);
    const headerView = new DataView(headerBuffer);
    const headerString = readDataViewAsString(headerView);

    // Parse the header.
    const headerJson = headerString
        .replace("True", "true")
        .replace("False", "false")
        .replace(/'/g, `"`)
        .replace(/,\s*}/, " }")
        .replace(/,?\)/, "]")
        .replace("(", "[");
    const header = JSON.parse(headerJson);
    
    // Extract properties.
    const big_endian = header.descr[0] === ">";
    const column_major = header.fortran_order;
    const dtype = header.descr.slice(1);
    const shape = header.shape;
    const version = versionMajor;

    let result = {
        "big_endian" : big_endian,
        "column_major" : column_major,
        "dtype" : dtype,
        "shape" : shape,
        "version" : version,
        "pos" : headerLength + lengthBytes + 8,
    };

    return result;
}

function writeStringToDataView(view, str, pos) {
    for (let i = 0; i < str.length; i++) {
        view.setInt8(pos + i, str.charCodeAt(i));
    }
    return pos + str.length;
}

function readDataViewAsString(view) {
    let out = "";
    for (let i = 0; i < view.byteLength; i++) {
        const val = view.getUint8(i);
        if (val === 0) {
            break;
        }
        out += String.fromCharCode(val);
    }
    return out;
}

function numberOfElements(shape) {
    if (shape.length === 0) {
        return 1;
    } else {
        return shape.reduce((a, b) => a * b);
    }
}

NpyWriter.prototype.write = function(data, close=true) {
    assert(data.length === numberOfElements(this.shape));

    // Build an array buffer to store the data.
    const elem_bytes = bytes_per_element[this.dtype];
    const bufferSize = data.length * elem_bytes;
    const buffer = new ArrayBuffer(bufferSize);
    const view = new DataView(buffer);
    let pos = 0;
    
    // Write to the buffer in the proper format.
    switch (this.dtype) {
        case "f8":
            for (let i = 0; i < data.length; i++) {
                view.setFloat64(pos, data[i], !this.big_endian);
                pos += elem_bytes;
            }

            break;
        
        case "f4":
            for (let i = 0; i < data.length; i++) {
                view.setFloat32(pos, data[i], !this.big_endian);
                pos += elem_bytes;
            }
            break;
        case "i8":
            for (let i = 0; i < data.length; i++) {
                view.setInt64(pos, data[i], !this.big_endian);
                pos += elem_bytes;
            }

            break;
        
        case "i4":
            for (let i = 0; i < data.length; i++) {
                view.setInt32(pos, data[i], !this.big_endian);
                pos += elem_bytes;
            }
            break;
        case "i2":
            for (let i = 0; i < data.length; i++) {
                view.setInt16(pos, data[i], !this.big_endian);
                pos += elem_bytes;
            }

            break;
        
        case "i1":
            for (let i = 0; i < data.length; i++) {
                view.setInt8(pos, data[i], !this.big_endian);
                pos += elem_bytes;
            }
            break;
    }

    // Close the writer if specified.
    if (close) {
        this.writer.close();
    }

    // Write the buffer to the file.
    this.writer.write(buffer, 0, bufferSize, this.pos);

    // Shift the position by the amount of data we've just written.
    this.pos += bufferSize;
}

NpyReader.prototype.read = function(close=true) {

    // Compute the buffer size and read the data.
    const dataLength = numberOfElements(this.shape);
    const elem_bytes = bytes_per_element[this.dtype];
    const bufferSize = dataLength * elem_bytes;
    const buffer = new ArrayBuffer(bufferSize);
    this.reader.read(buffer, 0, bufferSize, this.pos);
    this.pos += bufferSize;

    // Close the reader if specified.
    if (close) {
        this.reader.close();
    }

    // Feed the data into an appropriate array and return.
    switch (this.dtype) {
        case "f8":
            return new Float64Array(buffer);

        case "f4":
            return new Float32Array(buffer);

        case "i8":
            return new Int64Array(buffer);
        
        case "i4":
            return new Int32Array(buffer);

        case "i2":
            return new Int16Array(buffer);
        
        case "i1":
            return new Int8Array(buffer);
    }
}

module.exports = {
    "NpyWriter" : NpyWriter,
    "NpyReader" : NpyReader,
};

},{"./reader-writer-closer":11,"assert":5}],5:[function(require,module,exports){
(function (global){
'use strict';

// compare and isBuffer taken from https://github.com/feross/buffer/blob/680e9e5e488f22aac27599a57dc844a6315928dd/index.js
// original notice:

/*!
 * The buffer module from node.js, for the browser.
 *
 * @author   Feross Aboukhadijeh <feross@feross.org> <http://feross.org>
 * @license  MIT
 */
function compare(a, b) {
  if (a === b) {
    return 0;
  }

  var x = a.length;
  var y = b.length;

  for (var i = 0, len = Math.min(x, y); i < len; ++i) {
    if (a[i] !== b[i]) {
      x = a[i];
      y = b[i];
      break;
    }
  }

  if (x < y) {
    return -1;
  }
  if (y < x) {
    return 1;
  }
  return 0;
}
function isBuffer(b) {
  if (global.Buffer && typeof global.Buffer.isBuffer === 'function') {
    return global.Buffer.isBuffer(b);
  }
  return !!(b != null && b._isBuffer);
}

// based on node assert, original notice:

// http://wiki.commonjs.org/wiki/Unit_Testing/1.0
//
// THIS IS NOT TESTED NOR LIKELY TO WORK OUTSIDE V8!
//
// Originally from narwhal.js (http://narwhaljs.org)
// Copyright (c) 2009 Thomas Robinson <280north.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the 'Software'), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
// ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

var util = require('util/');
var hasOwn = Object.prototype.hasOwnProperty;
var pSlice = Array.prototype.slice;
var functionsHaveNames = (function () {
  return function foo() {}.name === 'foo';
}());
function pToString (obj) {
  return Object.prototype.toString.call(obj);
}
function isView(arrbuf) {
  if (isBuffer(arrbuf)) {
    return false;
  }
  if (typeof global.ArrayBuffer !== 'function') {
    return false;
  }
  if (typeof ArrayBuffer.isView === 'function') {
    return ArrayBuffer.isView(arrbuf);
  }
  if (!arrbuf) {
    return false;
  }
  if (arrbuf instanceof DataView) {
    return true;
  }
  if (arrbuf.buffer && arrbuf.buffer instanceof ArrayBuffer) {
    return true;
  }
  return false;
}
// 1. The assert module provides functions that throw
// AssertionError's when particular conditions are not met. The
// assert module must conform to the following interface.

var assert = module.exports = ok;

// 2. The AssertionError is defined in assert.
// new assert.AssertionError({ message: message,
//                             actual: actual,
//                             expected: expected })

var regex = /\s*function\s+([^\(\s]*)\s*/;
// based on https://github.com/ljharb/function.prototype.name/blob/adeeeec8bfcc6068b187d7d9fb3d5bb1d3a30899/implementation.js
function getName(func) {
  if (!util.isFunction(func)) {
    return;
  }
  if (functionsHaveNames) {
    return func.name;
  }
  var str = func.toString();
  var match = str.match(regex);
  return match && match[1];
}
assert.AssertionError = function AssertionError(options) {
  this.name = 'AssertionError';
  this.actual = options.actual;
  this.expected = options.expected;
  this.operator = options.operator;
  if (options.message) {
    this.message = options.message;
    this.generatedMessage = false;
  } else {
    this.message = getMessage(this);
    this.generatedMessage = true;
  }
  var stackStartFunction = options.stackStartFunction || fail;
  if (Error.captureStackTrace) {
    Error.captureStackTrace(this, stackStartFunction);
  } else {
    // non v8 browsers so we can have a stacktrace
    var err = new Error();
    if (err.stack) {
      var out = err.stack;

      // try to strip useless frames
      var fn_name = getName(stackStartFunction);
      var idx = out.indexOf('\n' + fn_name);
      if (idx >= 0) {
        // once we have located the function frame
        // we need to strip out everything before it (and its line)
        var next_line = out.indexOf('\n', idx + 1);
        out = out.substring(next_line + 1);
      }

      this.stack = out;
    }
  }
};

// assert.AssertionError instanceof Error
util.inherits(assert.AssertionError, Error);

function truncate(s, n) {
  if (typeof s === 'string') {
    return s.length < n ? s : s.slice(0, n);
  } else {
    return s;
  }
}
function inspect(something) {
  if (functionsHaveNames || !util.isFunction(something)) {
    return util.inspect(something);
  }
  var rawname = getName(something);
  var name = rawname ? ': ' + rawname : '';
  return '[Function' +  name + ']';
}
function getMessage(self) {
  return truncate(inspect(self.actual), 128) + ' ' +
         self.operator + ' ' +
         truncate(inspect(self.expected), 128);
}

// At present only the three keys mentioned above are used and
// understood by the spec. Implementations or sub modules can pass
// other keys to the AssertionError's constructor - they will be
// ignored.

// 3. All of the following functions must throw an AssertionError
// when a corresponding condition is not met, with a message that
// may be undefined if not provided.  All assertion methods provide
// both the actual and expected values to the assertion error for
// display purposes.

function fail(actual, expected, message, operator, stackStartFunction) {
  throw new assert.AssertionError({
    message: message,
    actual: actual,
    expected: expected,
    operator: operator,
    stackStartFunction: stackStartFunction
  });
}

// EXTENSION! allows for well behaved errors defined elsewhere.
assert.fail = fail;

// 4. Pure assertion tests whether a value is truthy, as determined
// by !!guard.
// assert.ok(guard, message_opt);
// This statement is equivalent to assert.equal(true, !!guard,
// message_opt);. To test strictly for the value true, use
// assert.strictEqual(true, guard, message_opt);.

function ok(value, message) {
  if (!value) fail(value, true, message, '==', assert.ok);
}
assert.ok = ok;

// 5. The equality assertion tests shallow, coercive equality with
// ==.
// assert.equal(actual, expected, message_opt);

assert.equal = function equal(actual, expected, message) {
  if (actual != expected) fail(actual, expected, message, '==', assert.equal);
};

// 6. The non-equality assertion tests for whether two objects are not equal
// with != assert.notEqual(actual, expected, message_opt);

assert.notEqual = function notEqual(actual, expected, message) {
  if (actual == expected) {
    fail(actual, expected, message, '!=', assert.notEqual);
  }
};

// 7. The equivalence assertion tests a deep equality relation.
// assert.deepEqual(actual, expected, message_opt);

assert.deepEqual = function deepEqual(actual, expected, message) {
  if (!_deepEqual(actual, expected, false)) {
    fail(actual, expected, message, 'deepEqual', assert.deepEqual);
  }
};

assert.deepStrictEqual = function deepStrictEqual(actual, expected, message) {
  if (!_deepEqual(actual, expected, true)) {
    fail(actual, expected, message, 'deepStrictEqual', assert.deepStrictEqual);
  }
};

function _deepEqual(actual, expected, strict, memos) {
  // 7.1. All identical values are equivalent, as determined by ===.
  if (actual === expected) {
    return true;
  } else if (isBuffer(actual) && isBuffer(expected)) {
    return compare(actual, expected) === 0;

  // 7.2. If the expected value is a Date object, the actual value is
  // equivalent if it is also a Date object that refers to the same time.
  } else if (util.isDate(actual) && util.isDate(expected)) {
    return actual.getTime() === expected.getTime();

  // 7.3 If the expected value is a RegExp object, the actual value is
  // equivalent if it is also a RegExp object with the same source and
  // properties (`global`, `multiline`, `lastIndex`, `ignoreCase`).
  } else if (util.isRegExp(actual) && util.isRegExp(expected)) {
    return actual.source === expected.source &&
           actual.global === expected.global &&
           actual.multiline === expected.multiline &&
           actual.lastIndex === expected.lastIndex &&
           actual.ignoreCase === expected.ignoreCase;

  // 7.4. Other pairs that do not both pass typeof value == 'object',
  // equivalence is determined by ==.
  } else if ((actual === null || typeof actual !== 'object') &&
             (expected === null || typeof expected !== 'object')) {
    return strict ? actual === expected : actual == expected;

  // If both values are instances of typed arrays, wrap their underlying
  // ArrayBuffers in a Buffer each to increase performance
  // This optimization requires the arrays to have the same type as checked by
  // Object.prototype.toString (aka pToString). Never perform binary
  // comparisons for Float*Arrays, though, since e.g. +0 === -0 but their
  // bit patterns are not identical.
  } else if (isView(actual) && isView(expected) &&
             pToString(actual) === pToString(expected) &&
             !(actual instanceof Float32Array ||
               actual instanceof Float64Array)) {
    return compare(new Uint8Array(actual.buffer),
                   new Uint8Array(expected.buffer)) === 0;

  // 7.5 For all other Object pairs, including Array objects, equivalence is
  // determined by having the same number of owned properties (as verified
  // with Object.prototype.hasOwnProperty.call), the same set of keys
  // (although not necessarily the same order), equivalent values for every
  // corresponding key, and an identical 'prototype' property. Note: this
  // accounts for both named and indexed properties on Arrays.
  } else if (isBuffer(actual) !== isBuffer(expected)) {
    return false;
  } else {
    memos = memos || {actual: [], expected: []};

    var actualIndex = memos.actual.indexOf(actual);
    if (actualIndex !== -1) {
      if (actualIndex === memos.expected.indexOf(expected)) {
        return true;
      }
    }

    memos.actual.push(actual);
    memos.expected.push(expected);

    return objEquiv(actual, expected, strict, memos);
  }
}

function isArguments(object) {
  return Object.prototype.toString.call(object) == '[object Arguments]';
}

function objEquiv(a, b, strict, actualVisitedObjects) {
  if (a === null || a === undefined || b === null || b === undefined)
    return false;
  // if one is a primitive, the other must be same
  if (util.isPrimitive(a) || util.isPrimitive(b))
    return a === b;
  if (strict && Object.getPrototypeOf(a) !== Object.getPrototypeOf(b))
    return false;
  var aIsArgs = isArguments(a);
  var bIsArgs = isArguments(b);
  if ((aIsArgs && !bIsArgs) || (!aIsArgs && bIsArgs))
    return false;
  if (aIsArgs) {
    a = pSlice.call(a);
    b = pSlice.call(b);
    return _deepEqual(a, b, strict);
  }
  var ka = objectKeys(a);
  var kb = objectKeys(b);
  var key, i;
  // having the same number of owned properties (keys incorporates
  // hasOwnProperty)
  if (ka.length !== kb.length)
    return false;
  //the same set of keys (although not necessarily the same order),
  ka.sort();
  kb.sort();
  //~~~cheap key test
  for (i = ka.length - 1; i >= 0; i--) {
    if (ka[i] !== kb[i])
      return false;
  }
  //equivalent values for every corresponding key, and
  //~~~possibly expensive deep test
  for (i = ka.length - 1; i >= 0; i--) {
    key = ka[i];
    if (!_deepEqual(a[key], b[key], strict, actualVisitedObjects))
      return false;
  }
  return true;
}

// 8. The non-equivalence assertion tests for any deep inequality.
// assert.notDeepEqual(actual, expected, message_opt);

assert.notDeepEqual = function notDeepEqual(actual, expected, message) {
  if (_deepEqual(actual, expected, false)) {
    fail(actual, expected, message, 'notDeepEqual', assert.notDeepEqual);
  }
};

assert.notDeepStrictEqual = notDeepStrictEqual;
function notDeepStrictEqual(actual, expected, message) {
  if (_deepEqual(actual, expected, true)) {
    fail(actual, expected, message, 'notDeepStrictEqual', notDeepStrictEqual);
  }
}


// 9. The strict equality assertion tests strict equality, as determined by ===.
// assert.strictEqual(actual, expected, message_opt);

assert.strictEqual = function strictEqual(actual, expected, message) {
  if (actual !== expected) {
    fail(actual, expected, message, '===', assert.strictEqual);
  }
};

// 10. The strict non-equality assertion tests for strict inequality, as
// determined by !==.  assert.notStrictEqual(actual, expected, message_opt);

assert.notStrictEqual = function notStrictEqual(actual, expected, message) {
  if (actual === expected) {
    fail(actual, expected, message, '!==', assert.notStrictEqual);
  }
};

function expectedException(actual, expected) {
  if (!actual || !expected) {
    return false;
  }

  if (Object.prototype.toString.call(expected) == '[object RegExp]') {
    return expected.test(actual);
  }

  try {
    if (actual instanceof expected) {
      return true;
    }
  } catch (e) {
    // Ignore.  The instanceof check doesn't work for arrow functions.
  }

  if (Error.isPrototypeOf(expected)) {
    return false;
  }

  return expected.call({}, actual) === true;
}

function _tryBlock(block) {
  var error;
  try {
    block();
  } catch (e) {
    error = e;
  }
  return error;
}

function _throws(shouldThrow, block, expected, message) {
  var actual;

  if (typeof block !== 'function') {
    throw new TypeError('"block" argument must be a function');
  }

  if (typeof expected === 'string') {
    message = expected;
    expected = null;
  }

  actual = _tryBlock(block);

  message = (expected && expected.name ? ' (' + expected.name + ').' : '.') +
            (message ? ' ' + message : '.');

  if (shouldThrow && !actual) {
    fail(actual, expected, 'Missing expected exception' + message);
  }

  var userProvidedMessage = typeof message === 'string';
  var isUnwantedException = !shouldThrow && util.isError(actual);
  var isUnexpectedException = !shouldThrow && actual && !expected;

  if ((isUnwantedException &&
      userProvidedMessage &&
      expectedException(actual, expected)) ||
      isUnexpectedException) {
    fail(actual, expected, 'Got unwanted exception' + message);
  }

  if ((shouldThrow && actual && expected &&
      !expectedException(actual, expected)) || (!shouldThrow && actual)) {
    throw actual;
  }
}

// 11. Expected to throw an error:
// assert.throws(block, Error_opt, message_opt);

assert.throws = function(block, /*optional*/error, /*optional*/message) {
  _throws(true, block, error, message);
};

// EXTENSION! This is annoying to write outside this module.
assert.doesNotThrow = function(block, /*optional*/error, /*optional*/message) {
  _throws(false, block, error, message);
};

assert.ifError = function(err) { if (err) throw err; };

var objectKeys = Object.keys || function (obj) {
  var keys = [];
  for (var key in obj) {
    if (hasOwn.call(obj, key)) keys.push(key);
  }
  return keys;
};

}).call(this,typeof global !== "undefined" ? global : typeof self !== "undefined" ? self : typeof window !== "undefined" ? window : {})
},{"util/":8}],6:[function(require,module,exports){
if (typeof Object.create === 'function') {
  // implementation from standard node.js 'util' module
  module.exports = function inherits(ctor, superCtor) {
    ctor.super_ = superCtor
    ctor.prototype = Object.create(superCtor.prototype, {
      constructor: {
        value: ctor,
        enumerable: false,
        writable: true,
        configurable: true
      }
    });
  };
} else {
  // old school shim for old browsers
  module.exports = function inherits(ctor, superCtor) {
    ctor.super_ = superCtor
    var TempCtor = function () {}
    TempCtor.prototype = superCtor.prototype
    ctor.prototype = new TempCtor()
    ctor.prototype.constructor = ctor
  }
}

},{}],7:[function(require,module,exports){
module.exports = function isBuffer(arg) {
  return arg && typeof arg === 'object'
    && typeof arg.copy === 'function'
    && typeof arg.fill === 'function'
    && typeof arg.readUInt8 === 'function';
}
},{}],8:[function(require,module,exports){
(function (process,global){
// Copyright Joyent, Inc. and other Node contributors.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to permit
// persons to whom the Software is furnished to do so, subject to the
// following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
// OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN
// NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
// OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE
// USE OR OTHER DEALINGS IN THE SOFTWARE.

var formatRegExp = /%[sdj%]/g;
exports.format = function(f) {
  if (!isString(f)) {
    var objects = [];
    for (var i = 0; i < arguments.length; i++) {
      objects.push(inspect(arguments[i]));
    }
    return objects.join(' ');
  }

  var i = 1;
  var args = arguments;
  var len = args.length;
  var str = String(f).replace(formatRegExp, function(x) {
    if (x === '%%') return '%';
    if (i >= len) return x;
    switch (x) {
      case '%s': return String(args[i++]);
      case '%d': return Number(args[i++]);
      case '%j':
        try {
          return JSON.stringify(args[i++]);
        } catch (_) {
          return '[Circular]';
        }
      default:
        return x;
    }
  });
  for (var x = args[i]; i < len; x = args[++i]) {
    if (isNull(x) || !isObject(x)) {
      str += ' ' + x;
    } else {
      str += ' ' + inspect(x);
    }
  }
  return str;
};


// Mark that a method should not be used.
// Returns a modified function which warns once by default.
// If --no-deprecation is set, then it is a no-op.
exports.deprecate = function(fn, msg) {
  // Allow for deprecating things in the process of starting up.
  if (isUndefined(global.process)) {
    return function() {
      return exports.deprecate(fn, msg).apply(this, arguments);
    };
  }

  if (process.noDeprecation === true) {
    return fn;
  }

  var warned = false;
  function deprecated() {
    if (!warned) {
      if (process.throwDeprecation) {
        throw new Error(msg);
      } else if (process.traceDeprecation) {
        console.trace(msg);
      } else {
        console.error(msg);
      }
      warned = true;
    }
    return fn.apply(this, arguments);
  }

  return deprecated;
};


var debugs = {};
var debugEnviron;
exports.debuglog = function(set) {
  if (isUndefined(debugEnviron))
    debugEnviron = process.env.NODE_DEBUG || '';
  set = set.toUpperCase();
  if (!debugs[set]) {
    if (new RegExp('\\b' + set + '\\b', 'i').test(debugEnviron)) {
      var pid = process.pid;
      debugs[set] = function() {
        var msg = exports.format.apply(exports, arguments);
        console.error('%s %d: %s', set, pid, msg);
      };
    } else {
      debugs[set] = function() {};
    }
  }
  return debugs[set];
};


/**
 * Echos the value of a value. Trys to print the value out
 * in the best way possible given the different types.
 *
 * @param {Object} obj The object to print out.
 * @param {Object} opts Optional options object that alters the output.
 */
/* legacy: obj, showHidden, depth, colors*/
function inspect(obj, opts) {
  // default options
  var ctx = {
    seen: [],
    stylize: stylizeNoColor
  };
  // legacy...
  if (arguments.length >= 3) ctx.depth = arguments[2];
  if (arguments.length >= 4) ctx.colors = arguments[3];
  if (isBoolean(opts)) {
    // legacy...
    ctx.showHidden = opts;
  } else if (opts) {
    // got an "options" object
    exports._extend(ctx, opts);
  }
  // set default options
  if (isUndefined(ctx.showHidden)) ctx.showHidden = false;
  if (isUndefined(ctx.depth)) ctx.depth = 2;
  if (isUndefined(ctx.colors)) ctx.colors = false;
  if (isUndefined(ctx.customInspect)) ctx.customInspect = true;
  if (ctx.colors) ctx.stylize = stylizeWithColor;
  return formatValue(ctx, obj, ctx.depth);
}
exports.inspect = inspect;


// http://en.wikipedia.org/wiki/ANSI_escape_code#graphics
inspect.colors = {
  'bold' : [1, 22],
  'italic' : [3, 23],
  'underline' : [4, 24],
  'inverse' : [7, 27],
  'white' : [37, 39],
  'grey' : [90, 39],
  'black' : [30, 39],
  'blue' : [34, 39],
  'cyan' : [36, 39],
  'green' : [32, 39],
  'magenta' : [35, 39],
  'red' : [31, 39],
  'yellow' : [33, 39]
};

// Don't use 'blue' not visible on cmd.exe
inspect.styles = {
  'special': 'cyan',
  'number': 'yellow',
  'boolean': 'yellow',
  'undefined': 'grey',
  'null': 'bold',
  'string': 'green',
  'date': 'magenta',
  // "name": intentionally not styling
  'regexp': 'red'
};


function stylizeWithColor(str, styleType) {
  var style = inspect.styles[styleType];

  if (style) {
    return '\u001b[' + inspect.colors[style][0] + 'm' + str +
           '\u001b[' + inspect.colors[style][1] + 'm';
  } else {
    return str;
  }
}


function stylizeNoColor(str, styleType) {
  return str;
}


function arrayToHash(array) {
  var hash = {};

  array.forEach(function(val, idx) {
    hash[val] = true;
  });

  return hash;
}


function formatValue(ctx, value, recurseTimes) {
  // Provide a hook for user-specified inspect functions.
  // Check that value is an object with an inspect function on it
  if (ctx.customInspect &&
      value &&
      isFunction(value.inspect) &&
      // Filter out the util module, it's inspect function is special
      value.inspect !== exports.inspect &&
      // Also filter out any prototype objects using the circular check.
      !(value.constructor && value.constructor.prototype === value)) {
    var ret = value.inspect(recurseTimes, ctx);
    if (!isString(ret)) {
      ret = formatValue(ctx, ret, recurseTimes);
    }
    return ret;
  }

  // Primitive types cannot have properties
  var primitive = formatPrimitive(ctx, value);
  if (primitive) {
    return primitive;
  }

  // Look up the keys of the object.
  var keys = Object.keys(value);
  var visibleKeys = arrayToHash(keys);

  if (ctx.showHidden) {
    keys = Object.getOwnPropertyNames(value);
  }

  // IE doesn't make error fields non-enumerable
  // http://msdn.microsoft.com/en-us/library/ie/dww52sbt(v=vs.94).aspx
  if (isError(value)
      && (keys.indexOf('message') >= 0 || keys.indexOf('description') >= 0)) {
    return formatError(value);
  }

  // Some type of object without properties can be shortcutted.
  if (keys.length === 0) {
    if (isFunction(value)) {
      var name = value.name ? ': ' + value.name : '';
      return ctx.stylize('[Function' + name + ']', 'special');
    }
    if (isRegExp(value)) {
      return ctx.stylize(RegExp.prototype.toString.call(value), 'regexp');
    }
    if (isDate(value)) {
      return ctx.stylize(Date.prototype.toString.call(value), 'date');
    }
    if (isError(value)) {
      return formatError(value);
    }
  }

  var base = '', array = false, braces = ['{', '}'];

  // Make Array say that they are Array
  if (isArray(value)) {
    array = true;
    braces = ['[', ']'];
  }

  // Make functions say that they are functions
  if (isFunction(value)) {
    var n = value.name ? ': ' + value.name : '';
    base = ' [Function' + n + ']';
  }

  // Make RegExps say that they are RegExps
  if (isRegExp(value)) {
    base = ' ' + RegExp.prototype.toString.call(value);
  }

  // Make dates with properties first say the date
  if (isDate(value)) {
    base = ' ' + Date.prototype.toUTCString.call(value);
  }

  // Make error with message first say the error
  if (isError(value)) {
    base = ' ' + formatError(value);
  }

  if (keys.length === 0 && (!array || value.length == 0)) {
    return braces[0] + base + braces[1];
  }

  if (recurseTimes < 0) {
    if (isRegExp(value)) {
      return ctx.stylize(RegExp.prototype.toString.call(value), 'regexp');
    } else {
      return ctx.stylize('[Object]', 'special');
    }
  }

  ctx.seen.push(value);

  var output;
  if (array) {
    output = formatArray(ctx, value, recurseTimes, visibleKeys, keys);
  } else {
    output = keys.map(function(key) {
      return formatProperty(ctx, value, recurseTimes, visibleKeys, key, array);
    });
  }

  ctx.seen.pop();

  return reduceToSingleString(output, base, braces);
}


function formatPrimitive(ctx, value) {
  if (isUndefined(value))
    return ctx.stylize('undefined', 'undefined');
  if (isString(value)) {
    var simple = '\'' + JSON.stringify(value).replace(/^"|"$/g, '')
                                             .replace(/'/g, "\\'")
                                             .replace(/\\"/g, '"') + '\'';
    return ctx.stylize(simple, 'string');
  }
  if (isNumber(value))
    return ctx.stylize('' + value, 'number');
  if (isBoolean(value))
    return ctx.stylize('' + value, 'boolean');
  // For some reason typeof null is "object", so special case here.
  if (isNull(value))
    return ctx.stylize('null', 'null');
}


function formatError(value) {
  return '[' + Error.prototype.toString.call(value) + ']';
}


function formatArray(ctx, value, recurseTimes, visibleKeys, keys) {
  var output = [];
  for (var i = 0, l = value.length; i < l; ++i) {
    if (hasOwnProperty(value, String(i))) {
      output.push(formatProperty(ctx, value, recurseTimes, visibleKeys,
          String(i), true));
    } else {
      output.push('');
    }
  }
  keys.forEach(function(key) {
    if (!key.match(/^\d+$/)) {
      output.push(formatProperty(ctx, value, recurseTimes, visibleKeys,
          key, true));
    }
  });
  return output;
}


function formatProperty(ctx, value, recurseTimes, visibleKeys, key, array) {
  var name, str, desc;
  desc = Object.getOwnPropertyDescriptor(value, key) || { value: value[key] };
  if (desc.get) {
    if (desc.set) {
      str = ctx.stylize('[Getter/Setter]', 'special');
    } else {
      str = ctx.stylize('[Getter]', 'special');
    }
  } else {
    if (desc.set) {
      str = ctx.stylize('[Setter]', 'special');
    }
  }
  if (!hasOwnProperty(visibleKeys, key)) {
    name = '[' + key + ']';
  }
  if (!str) {
    if (ctx.seen.indexOf(desc.value) < 0) {
      if (isNull(recurseTimes)) {
        str = formatValue(ctx, desc.value, null);
      } else {
        str = formatValue(ctx, desc.value, recurseTimes - 1);
      }
      if (str.indexOf('\n') > -1) {
        if (array) {
          str = str.split('\n').map(function(line) {
            return '  ' + line;
          }).join('\n').substr(2);
        } else {
          str = '\n' + str.split('\n').map(function(line) {
            return '   ' + line;
          }).join('\n');
        }
      }
    } else {
      str = ctx.stylize('[Circular]', 'special');
    }
  }
  if (isUndefined(name)) {
    if (array && key.match(/^\d+$/)) {
      return str;
    }
    name = JSON.stringify('' + key);
    if (name.match(/^"([a-zA-Z_][a-zA-Z_0-9]*)"$/)) {
      name = name.substr(1, name.length - 2);
      name = ctx.stylize(name, 'name');
    } else {
      name = name.replace(/'/g, "\\'")
                 .replace(/\\"/g, '"')
                 .replace(/(^"|"$)/g, "'");
      name = ctx.stylize(name, 'string');
    }
  }

  return name + ': ' + str;
}


function reduceToSingleString(output, base, braces) {
  var numLinesEst = 0;
  var length = output.reduce(function(prev, cur) {
    numLinesEst++;
    if (cur.indexOf('\n') >= 0) numLinesEst++;
    return prev + cur.replace(/\u001b\[\d\d?m/g, '').length + 1;
  }, 0);

  if (length > 60) {
    return braces[0] +
           (base === '' ? '' : base + '\n ') +
           ' ' +
           output.join(',\n  ') +
           ' ' +
           braces[1];
  }

  return braces[0] + base + ' ' + output.join(', ') + ' ' + braces[1];
}


// NOTE: These type checking functions intentionally don't use `instanceof`
// because it is fragile and can be easily faked with `Object.create()`.
function isArray(ar) {
  return Array.isArray(ar);
}
exports.isArray = isArray;

function isBoolean(arg) {
  return typeof arg === 'boolean';
}
exports.isBoolean = isBoolean;

function isNull(arg) {
  return arg === null;
}
exports.isNull = isNull;

function isNullOrUndefined(arg) {
  return arg == null;
}
exports.isNullOrUndefined = isNullOrUndefined;

function isNumber(arg) {
  return typeof arg === 'number';
}
exports.isNumber = isNumber;

function isString(arg) {
  return typeof arg === 'string';
}
exports.isString = isString;

function isSymbol(arg) {
  return typeof arg === 'symbol';
}
exports.isSymbol = isSymbol;

function isUndefined(arg) {
  return arg === void 0;
}
exports.isUndefined = isUndefined;

function isRegExp(re) {
  return isObject(re) && objectToString(re) === '[object RegExp]';
}
exports.isRegExp = isRegExp;

function isObject(arg) {
  return typeof arg === 'object' && arg !== null;
}
exports.isObject = isObject;

function isDate(d) {
  return isObject(d) && objectToString(d) === '[object Date]';
}
exports.isDate = isDate;

function isError(e) {
  return isObject(e) &&
      (objectToString(e) === '[object Error]' || e instanceof Error);
}
exports.isError = isError;

function isFunction(arg) {
  return typeof arg === 'function';
}
exports.isFunction = isFunction;

function isPrimitive(arg) {
  return arg === null ||
         typeof arg === 'boolean' ||
         typeof arg === 'number' ||
         typeof arg === 'string' ||
         typeof arg === 'symbol' ||  // ES6 symbol
         typeof arg === 'undefined';
}
exports.isPrimitive = isPrimitive;

exports.isBuffer = require('./support/isBuffer');

function objectToString(o) {
  return Object.prototype.toString.call(o);
}


function pad(n) {
  return n < 10 ? '0' + n.toString(10) : n.toString(10);
}


var months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep',
              'Oct', 'Nov', 'Dec'];

// 26 Feb 16:19:34
function timestamp() {
  var d = new Date();
  var time = [pad(d.getHours()),
              pad(d.getMinutes()),
              pad(d.getSeconds())].join(':');
  return [d.getDate(), months[d.getMonth()], time].join(' ');
}


// log is just a thin wrapper to console.log that prepends a timestamp
exports.log = function() {
  console.log('%s - %s', timestamp(), exports.format.apply(exports, arguments));
};


/**
 * Inherit the prototype methods from one constructor into another.
 *
 * The Function.prototype.inherits from lang.js rewritten as a standalone
 * function (not on Function.prototype). NOTE: If this file is to be loaded
 * during bootstrapping this function needs to be rewritten using some native
 * functions as prototype setup using normal JavaScript does not work as
 * expected during bootstrapping (see mirror.js in r114903).
 *
 * @param {function} ctor Constructor function which needs to inherit the
 *     prototype.
 * @param {function} superCtor Constructor function to inherit prototype from.
 */
exports.inherits = require('inherits');

exports._extend = function(origin, add) {
  // Don't do anything if add isn't an object
  if (!add || !isObject(add)) return origin;

  var keys = Object.keys(add);
  var i = keys.length;
  while (i--) {
    origin[keys[i]] = add[keys[i]];
  }
  return origin;
};

function hasOwnProperty(obj, prop) {
  return Object.prototype.hasOwnProperty.call(obj, prop);
}

}).call(this,require('_process'),typeof global !== "undefined" ? global : typeof self !== "undefined" ? self : typeof window !== "undefined" ? window : {})
},{"./support/isBuffer":7,"_process":10,"inherits":6}],9:[function(require,module,exports){
(function (process){
// .dirname, .basename, and .extname methods are extracted from Node.js v8.11.1,
// backported and transplited with Babel, with backwards-compat fixes

// Copyright Joyent, Inc. and other Node contributors.
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to permit
// persons to whom the Software is furnished to do so, subject to the
// following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS
// OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN
// NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
// DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
// OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE
// USE OR OTHER DEALINGS IN THE SOFTWARE.

// resolves . and .. elements in a path array with directory names there
// must be no slashes, empty elements, or device names (c:\) in the array
// (so also no leading and trailing slashes - it does not distinguish
// relative and absolute paths)
function normalizeArray(parts, allowAboveRoot) {
  // if the path tries to go above the root, `up` ends up > 0
  var up = 0;
  for (var i = parts.length - 1; i >= 0; i--) {
    var last = parts[i];
    if (last === '.') {
      parts.splice(i, 1);
    } else if (last === '..') {
      parts.splice(i, 1);
      up++;
    } else if (up) {
      parts.splice(i, 1);
      up--;
    }
  }

  // if the path is allowed to go above the root, restore leading ..s
  if (allowAboveRoot) {
    for (; up--; up) {
      parts.unshift('..');
    }
  }

  return parts;
}

// path.resolve([from ...], to)
// posix version
exports.resolve = function() {
  var resolvedPath = '',
      resolvedAbsolute = false;

  for (var i = arguments.length - 1; i >= -1 && !resolvedAbsolute; i--) {
    var path = (i >= 0) ? arguments[i] : process.cwd();

    // Skip empty and invalid entries
    if (typeof path !== 'string') {
      throw new TypeError('Arguments to path.resolve must be strings');
    } else if (!path) {
      continue;
    }

    resolvedPath = path + '/' + resolvedPath;
    resolvedAbsolute = path.charAt(0) === '/';
  }

  // At this point the path should be resolved to a full absolute path, but
  // handle relative paths to be safe (might happen when process.cwd() fails)

  // Normalize the path
  resolvedPath = normalizeArray(filter(resolvedPath.split('/'), function(p) {
    return !!p;
  }), !resolvedAbsolute).join('/');

  return ((resolvedAbsolute ? '/' : '') + resolvedPath) || '.';
};

// path.normalize(path)
// posix version
exports.normalize = function(path) {
  var isAbsolute = exports.isAbsolute(path),
      trailingSlash = substr(path, -1) === '/';

  // Normalize the path
  path = normalizeArray(filter(path.split('/'), function(p) {
    return !!p;
  }), !isAbsolute).join('/');

  if (!path && !isAbsolute) {
    path = '.';
  }
  if (path && trailingSlash) {
    path += '/';
  }

  return (isAbsolute ? '/' : '') + path;
};

// posix version
exports.isAbsolute = function(path) {
  return path.charAt(0) === '/';
};

// posix version
exports.join = function() {
  var paths = Array.prototype.slice.call(arguments, 0);
  return exports.normalize(filter(paths, function(p, index) {
    if (typeof p !== 'string') {
      throw new TypeError('Arguments to path.join must be strings');
    }
    return p;
  }).join('/'));
};


// path.relative(from, to)
// posix version
exports.relative = function(from, to) {
  from = exports.resolve(from).substr(1);
  to = exports.resolve(to).substr(1);

  function trim(arr) {
    var start = 0;
    for (; start < arr.length; start++) {
      if (arr[start] !== '') break;
    }

    var end = arr.length - 1;
    for (; end >= 0; end--) {
      if (arr[end] !== '') break;
    }

    if (start > end) return [];
    return arr.slice(start, end - start + 1);
  }

  var fromParts = trim(from.split('/'));
  var toParts = trim(to.split('/'));

  var length = Math.min(fromParts.length, toParts.length);
  var samePartsLength = length;
  for (var i = 0; i < length; i++) {
    if (fromParts[i] !== toParts[i]) {
      samePartsLength = i;
      break;
    }
  }

  var outputParts = [];
  for (var i = samePartsLength; i < fromParts.length; i++) {
    outputParts.push('..');
  }

  outputParts = outputParts.concat(toParts.slice(samePartsLength));

  return outputParts.join('/');
};

exports.sep = '/';
exports.delimiter = ':';

exports.dirname = function (path) {
  if (typeof path !== 'string') path = path + '';
  if (path.length === 0) return '.';
  var code = path.charCodeAt(0);
  var hasRoot = code === 47 /*/*/;
  var end = -1;
  var matchedSlash = true;
  for (var i = path.length - 1; i >= 1; --i) {
    code = path.charCodeAt(i);
    if (code === 47 /*/*/) {
        if (!matchedSlash) {
          end = i;
          break;
        }
      } else {
      // We saw the first non-path separator
      matchedSlash = false;
    }
  }

  if (end === -1) return hasRoot ? '/' : '.';
  if (hasRoot && end === 1) {
    // return '//';
    // Backwards-compat fix:
    return '/';
  }
  return path.slice(0, end);
};

function basename(path) {
  if (typeof path !== 'string') path = path + '';

  var start = 0;
  var end = -1;
  var matchedSlash = true;
  var i;

  for (i = path.length - 1; i >= 0; --i) {
    if (path.charCodeAt(i) === 47 /*/*/) {
        // If we reached a path separator that was not part of a set of path
        // separators at the end of the string, stop now
        if (!matchedSlash) {
          start = i + 1;
          break;
        }
      } else if (end === -1) {
      // We saw the first non-path separator, mark this as the end of our
      // path component
      matchedSlash = false;
      end = i + 1;
    }
  }

  if (end === -1) return '';
  return path.slice(start, end);
}

// Uses a mixed approach for backwards-compatibility, as ext behavior changed
// in new Node.js versions, so only basename() above is backported here
exports.basename = function (path, ext) {
  var f = basename(path);
  if (ext && f.substr(-1 * ext.length) === ext) {
    f = f.substr(0, f.length - ext.length);
  }
  return f;
};

exports.extname = function (path) {
  if (typeof path !== 'string') path = path + '';
  var startDot = -1;
  var startPart = 0;
  var end = -1;
  var matchedSlash = true;
  // Track the state of characters (if any) we see before our first dot and
  // after any path separator we find
  var preDotState = 0;
  for (var i = path.length - 1; i >= 0; --i) {
    var code = path.charCodeAt(i);
    if (code === 47 /*/*/) {
        // If we reached a path separator that was not part of a set of path
        // separators at the end of the string, stop now
        if (!matchedSlash) {
          startPart = i + 1;
          break;
        }
        continue;
      }
    if (end === -1) {
      // We saw the first non-path separator, mark this as the end of our
      // extension
      matchedSlash = false;
      end = i + 1;
    }
    if (code === 46 /*.*/) {
        // If this is our first dot, mark it as the start of our extension
        if (startDot === -1)
          startDot = i;
        else if (preDotState !== 1)
          preDotState = 1;
    } else if (startDot !== -1) {
      // We saw a non-dot and non-path separator before our dot, so we should
      // have a good chance at having a non-empty extension
      preDotState = -1;
    }
  }

  if (startDot === -1 || end === -1 ||
      // We saw a non-dot character immediately before the dot
      preDotState === 0 ||
      // The (right-most) trimmed path component is exactly '..'
      preDotState === 1 && startDot === end - 1 && startDot === startPart + 1) {
    return '';
  }
  return path.slice(startDot, end);
};

function filter (xs, f) {
    if (xs.filter) return xs.filter(f);
    var res = [];
    for (var i = 0; i < xs.length; i++) {
        if (f(xs[i], i, xs)) res.push(xs[i]);
    }
    return res;
}

// String.prototype.substr - negative index don't work in IE8
var substr = 'ab'.substr(-1) === 'b'
    ? function (str, start, len) { return str.substr(start, len) }
    : function (str, start, len) {
        if (start < 0) start = str.length + start;
        return str.substr(start, len);
    }
;

}).call(this,require('_process'))
},{"_process":10}],10:[function(require,module,exports){
// shim for using process in browser
var process = module.exports = {};

// cached from whatever global is present so that test runners that stub it
// don't break things.  But we need to wrap it in a try catch in case it is
// wrapped in strict mode code which doesn't define any globals.  It's inside a
// function because try/catches deoptimize in certain engines.

var cachedSetTimeout;
var cachedClearTimeout;

function defaultSetTimout() {
    throw new Error('setTimeout has not been defined');
}
function defaultClearTimeout () {
    throw new Error('clearTimeout has not been defined');
}
(function () {
    try {
        if (typeof setTimeout === 'function') {
            cachedSetTimeout = setTimeout;
        } else {
            cachedSetTimeout = defaultSetTimout;
        }
    } catch (e) {
        cachedSetTimeout = defaultSetTimout;
    }
    try {
        if (typeof clearTimeout === 'function') {
            cachedClearTimeout = clearTimeout;
        } else {
            cachedClearTimeout = defaultClearTimeout;
        }
    } catch (e) {
        cachedClearTimeout = defaultClearTimeout;
    }
} ())
function runTimeout(fun) {
    if (cachedSetTimeout === setTimeout) {
        //normal enviroments in sane situations
        return setTimeout(fun, 0);
    }
    // if setTimeout wasn't available but was latter defined
    if ((cachedSetTimeout === defaultSetTimout || !cachedSetTimeout) && setTimeout) {
        cachedSetTimeout = setTimeout;
        return setTimeout(fun, 0);
    }
    try {
        // when when somebody has screwed with setTimeout but no I.E. maddness
        return cachedSetTimeout(fun, 0);
    } catch(e){
        try {
            // When we are in I.E. but the script has been evaled so I.E. doesn't trust the global object when called normally
            return cachedSetTimeout.call(null, fun, 0);
        } catch(e){
            // same as above but when it's a version of I.E. that must have the global object for 'this', hopfully our context correct otherwise it will throw a global error
            return cachedSetTimeout.call(this, fun, 0);
        }
    }


}
function runClearTimeout(marker) {
    if (cachedClearTimeout === clearTimeout) {
        //normal enviroments in sane situations
        return clearTimeout(marker);
    }
    // if clearTimeout wasn't available but was latter defined
    if ((cachedClearTimeout === defaultClearTimeout || !cachedClearTimeout) && clearTimeout) {
        cachedClearTimeout = clearTimeout;
        return clearTimeout(marker);
    }
    try {
        // when when somebody has screwed with setTimeout but no I.E. maddness
        return cachedClearTimeout(marker);
    } catch (e){
        try {
            // When we are in I.E. but the script has been evaled so I.E. doesn't  trust the global object when called normally
            return cachedClearTimeout.call(null, marker);
        } catch (e){
            // same as above but when it's a version of I.E. that must have the global object for 'this', hopfully our context correct otherwise it will throw a global error.
            // Some versions of I.E. have different rules for clearTimeout vs setTimeout
            return cachedClearTimeout.call(this, marker);
        }
    }



}
var queue = [];
var draining = false;
var currentQueue;
var queueIndex = -1;

function cleanUpNextTick() {
    if (!draining || !currentQueue) {
        return;
    }
    draining = false;
    if (currentQueue.length) {
        queue = currentQueue.concat(queue);
    } else {
        queueIndex = -1;
    }
    if (queue.length) {
        drainQueue();
    }
}

function drainQueue() {
    if (draining) {
        return;
    }
    var timeout = runTimeout(cleanUpNextTick);
    draining = true;

    var len = queue.length;
    while(len) {
        currentQueue = queue;
        queue = [];
        while (++queueIndex < len) {
            if (currentQueue) {
                currentQueue[queueIndex].run();
            }
        }
        queueIndex = -1;
        len = queue.length;
    }
    currentQueue = null;
    draining = false;
    runClearTimeout(timeout);
}

process.nextTick = function (fun) {
    var args = new Array(arguments.length - 1);
    if (arguments.length > 1) {
        for (var i = 1; i < arguments.length; i++) {
            args[i - 1] = arguments[i];
        }
    }
    queue.push(new Item(fun, args));
    if (queue.length === 1 && !draining) {
        runTimeout(drainQueue);
    }
};

// v8 likes predictible objects
function Item(fun, array) {
    this.fun = fun;
    this.array = array;
}
Item.prototype.run = function () {
    this.fun.apply(null, this.array);
};
process.title = 'browser';
process.browser = true;
process.env = {};
process.argv = [];
process.version = ''; // empty string to avoid regexp issues
process.versions = {};

function noop() {}

process.on = noop;
process.addListener = noop;
process.once = noop;
process.off = noop;
process.removeListener = noop;
process.removeAllListeners = noop;
process.emit = noop;
process.prependListener = noop;
process.prependOnceListener = noop;

process.listeners = function (name) { return [] }

process.binding = function (name) {
    throw new Error('process.binding is not supported');
};

process.cwd = function () { return '/' };
process.chdir = function (dir) {
    throw new Error('process.chdir is not supported');
};
process.umask = function() { return 0; };

},{}],11:[function(require,module,exports){
"use strict";

function ReaderWriterCloser() {
}

ReaderWriterCloser.prototype.read = function(buffer, offset, length, position) {
    throw new Error("Not implemented");
}

ReaderWriterCloser.prototype.readLines = function() {
    throw new Error("Not implemented");
}

ReaderWriterCloser.prototype.write = function(buffer, offset, length, position) {
    throw new Error("Not implemented");
}

ReaderWriterCloser.prototype.writeLines = function(data) {
    throw new Error("Not implemented");
}

ReaderWriterCloser.prototype.close = function() {
    throw new Error("Not implemented");
}

module.exports = ReaderWriterCloser

},{}],12:[function(require,module,exports){
'use strict';

var assert = require("assert");
var Permutation = require('./iterative-permutation');

const NAME_FORMAT = new RegExp("^[a-z_][0-9a-z_]*$");
const DIM_FORMAT = new RegExp("^[a-z_][0-9a-z_]*[\+\?\*]?$");

function SchemaException(message, path="") {
    this.message = message;
    this.path = path;
}

function Schema(nodes, categoryClasses, cyclic=false, undirected=false, fanin=false, srcDims=null) {
    
    if (nodes === null) {
        nodes = {};
    }
    if (categoryClasses === null) {
        categoryClasses = {};
    }

    // Simple type checks.
    if (typeof nodes !== "object") {
        throw new SchemaException("Schema nodes must be a key-value dictionary.");
    }
    if (typeof categoryClasses !== "object") {
        throw new SchemaException("Schema category classes must be a key-value dictionary.");
    }
    if (typeof cyclic !== "boolean") {
        throw new SchemaException("Schema cyclic flag must be a boolean.");
    }
    if (typeof undirected !== "boolean") {
        throw new SchemaException("Schema undirected flag must be a boolean.");
    }
    if (typeof fanin !== "boolean") {
        throw new SchemaException("Schema fan-in flag must be a boolean.");
    }

    // Quantity check.
    if (Object.keys(nodes).length < 1) {
        throw new SchemaException("Schema must have at least one node.");
    }

    // Type and name checks for nodes.
    for (let k in nodes) {
        assert(typeof k === "string");
        assert(nodes[k] instanceof Node);
        if (NAME_FORMAT.test(k) === false) {
            throw new SchemaException("Schema node names may contain lowercase letters, numbers and underscores. \
                They must start with a letter.", "nodes."+k);
        }
    }

    // Type and name checks for category classes.
    for (let k in categoryClasses) {
        assert(typeof k === "string");
        assert(categoryClasses[k] instanceof Class);
        if (NAME_FORMAT.test(k) === false) {
            throw new SchemaException("Schema category class names may contain lowercase letters, numbers and \
                underscores. They must start with a letter.", "classes."+k);
        }
    }

    // Reference checks.
    let orphanClasses = new Set(Object.keys(categoryClasses));
    for (let k in nodes) {
        // Check links.
        let linkCount = 0;
        for (let l in nodes[k].links) {
            if (!(l in nodes)) {
                throw new SchemaException("Node link points to unknown node.", "nodes."+k+".links."+l);
            }
            if (nodes[l].isSingleton) {
                throw new SchemaException("Node link points to a singleton node.", "nodes."+k+".links."+l);
            }
            if (undirected && !fanin) {
                if (nodes[k].links[l].dim[1] === "inf") {
                    throw new SchemaException("Nodes in undirected schemas with fan-in cannot have infinite outgoing links.", "nodes."+k+".links."+l);
                }
                linkCount += nodes[k].links[l].dim[1];
            }
        }

        // Check category field classes.
        for (let f in nodes[k].fields) {
            if (nodes[k].fields[f] instanceof Category) {
                if (!(nodes[k].fields[f].categoryClass in categoryClasses)) {
                    throw new SchemaException("Field category class undefined.", "nodes."+k+".fields."+f);
                }
                orphanClasses.delete(nodes[k].fields[f].categoryClass);
            }
        }

        if (undirected && !fanin && linkCount > 2) {
            throw new SchemaException("Nodes in undirected schemas with fan-in can have at most 2 outgoing links.", "nodes."+k);
        }
    }

    // Check if there are unreferenced classes.
    if (orphanClasses.size > 0) {
        const iterator1 = orphanClasses[Symbol.iterator]();
        throw new SchemaException("Every declared class must be referenced in a category.",
            "classes."+iterator1.next().value);
    }

    // Source dimensions check.
    if (srcDims !== null) {
        if (typeof srcDims !== "object") {
            throw new SchemaException("Source dimensions field must be a key-value dictionary.", "src-dims");
        }
        for (let k in srcDims) {
            assert(typeof k === "string")
            if (typeof srcDims[k] !== "string" && !Number.isInteger(srcDims[k])) {
                throw new SchemaException("Source dimension values must be strings or integers.", "src-dims."+k);
            }
        }       
    }
            
    this.nodes = nodes;
    this.categoryClasses = categoryClasses;
    this.cyclic = cyclic;
    this.undirected = undirected;
    this.fanin = fanin;
    this.srcDims = srcDims;

}

function load(input) {
    assert(typeof input === "object");

    let nodes = "nodes" in input ? input["nodes"] : null;
    let constraints = "ref-constraints" in input ? input["ref-constraints"] : {};
    let categoryClasses = "classes" in input ? input["classes"] : {};
    let srcDims = "src-dims" in input ? input["src-dims"] : null;

    if (nodes !== null && typeof nodes === "object") {
        for (let k in nodes) {
            try {
                nodes[k] = loadNode(nodes[k]);
            } catch(err) {
                if (err instanceof SchemaException) {
                    err.path = "nodes." + k;
                }
                throw err;
            }
        }
    }

    if (categoryClasses !== null && typeof categoryClasses === "object") {
        for (let k in categoryClasses) {
            try {
                categoryClasses[k] = loadClass(categoryClasses[k]);
            } catch(err) {
                if (err instanceof SchemaException) {
                    err.path = "classes." + k;
                }
                throw err;
            }
        }
    } else {
        throw new SchemaException("Category classes must be a key-value dictionary.", "classes");
    }

    let [cyclic, undirected, fanin] = [false, false ,false];
    if (constraints !== null && typeof constraints === "object" && (constraints instanceof Array) === false) {
        cyclic = "cyclic" in constraints ? constraints["cyclic"] : false;
        undirected = "undirected" in constraints ? constraints["undirected"] : false;
        fanin = "fan-in" in constraints ? constraints["fan-in"] : false;
    } else {
        throw new SchemaException("Reference constraints field must be a key-value dictionary.", "ref-constraints");
    }

    return new Schema(nodes, categoryClasses, cyclic, undirected, fanin, srcDims);
}

Schema.prototype.dump = function() {
    let result = {
        "ref-constraints" : {
            "cyclic" : this.cyclic,
            "undirected" : this.undirected,
            "fan-in" : this.fanin
        }
    };

    let nodes = {};
    for (k in this.nodes) {
        nodes[k] = dumpNode(this.nodes[k]);
    }
    result["nodes"] = nodes;

    if (Object.keys(this.categoryClasses).length > 0) {
        let categoryClasses = {};
        for (k in this.categoryClasses) {
            categoryClasses[k] = dumpClass(this.categoryClasses[k]);
        }
        result["classes"] = categoryClasses;
    }
    
    if (this.srcDims !== null) {
        result["src-dims"] = this.srcDims;
    }

    return result;
}

Schema.prototype.isVariable = function() {
    for (let node in this.nodes) {
        if (this.nodes[node].isVariable()) {
            return true;
        }
    }
    for (let categoryClass in this.categoryClasses) {
        if (this.categoryClasses[categoryClass].isVariable()) {
            return true;
        }
    }
    return false;
}

Schema.prototype.match = function(source, buildMatching=false) {

    assert(source instanceof Schema)

    let dimMap = {};
    let classNameMap = {};
    let nodeNameMap = {};
    let nodes = {};
    
    // Next we split the nodes into singletons and non-singletons.
    let selfSingletonNames = [];
    let selfNonsingletonNames = [];
    for (let k in this.nodes) {
        if (this.nodes[k].isSingleton) {
            selfSingletonNames.push(k);
        } else {
            selfNonsingletonNames.push(k);
        }
    }
    let sourceSingletonNames = [];
    let sourceNonsingletonNames = [];
    for (let k in source.nodes) {
        if (source.nodes[k].isSingleton) {
            sourceSingletonNames.push(k);
        } else {
            sourceNonsingletonNames.push(k);
        }
    }

    // We only compare referential constraints if there are non-singleton nodes.
    if (selfNonsingletonNames.length > 0) {
        if (source.cyclic && this.cyclic === false) {
            // A cyclic graph cannot be accepted by an acyclic destination.
            if (buildMatching) {
                return null;
            } else {
                return false;
            }
        }
        if (source.undirected === false && this.undirected) {
            // A directed graph cannot be accepted by an undirected destination.
            if (buildMatching) {
                return null;
            } else {
                return false;
            }
        }
        if (source.fanin && this.fanin === false) {
            // A graph that allows fan-in (multiple incoming pointers per node) cannot be accepted by
            // a destination that forbids it.
            if (buildMatching) {
                return null;
            } else {
                return false;
            }
        }
    }

    // Try all possible singleton node matchings. Individual matchings are not independent
    // because of various name matchings. Maybe this can be optimised.
    let match = sourceSingletonNames.length === 0;
    let singletonPerms = new Permutation(sourceSingletonNames);
    while(singletonPerms.hasNext()) {
        let sourceSingletonNamesPerm = singletonPerms.next();
        let dimMapUpdates = {};
        let classNameMapUpdates = {};
        let nodesIter = {};

        for (let i = 0; i < sourceSingletonNames.length; i++) {
            let selfName = selfSingletonNames[i];
            let sourceName = sourceSingletonNamesPerm[i];

            let [nodeMatch, dimMapUpdatesNew, classNameMapUpdatesNew] = matchNode(this.nodes[selfName],
                source.nodes[sourceName],
                Object.assign({}, dimMap, dimMapUpdates),
                Object.assign({}, classNameMap, classNameMapUpdates),
                nodeNameMap,
                this.categoryClasses,
                source.categoryClasses,
                buildMatching);
            
            if (buildMatching) {
                match = nodeMatch !== null;
            } else {
                match = nodeMatch;
            }
            
            if (match) {
                // If there was a match, we update the dimension and class name mappings.
                Object.assign(dimMapUpdates, dimMapUpdatesNew);
                Object.assign(classNameMapUpdates, classNameMapUpdatesNew);
                
                if (buildMatching) {
                    nodeMatch.srcName = sourceName;
                    nodesIter[selfName] = nodeMatch;
                }
            } else {
                // On the first failed match, we skip this permutation.
                break;
            }
        }

        // If we have found a matching, end the search.
        if (match) {
            // Add matched node names to name map.

            Object.assign(dimMap, dimMapUpdates);
            Object.assign(classNameMap, classNameMapUpdates);

            for (let i = 0; i < selfSingletonNames.length; i++) {
                nodeNameMap[selfSingletonNames[i]] = sourceSingletonNamesPerm[i];
            }

            if (buildMatching) {
                Object.assign(nodes, nodesIter);
            }
            break;
        }
    }
    
    // If no matching was found, we don't need to go on further.
    if (match === false) {
        if (buildMatching) {
            return null;
        } else {
            return false;
        }
    }

    // Try all possible non-singleton node matchings. Individual matchings are not independent
    // because of various name matchings. Maybe this can be optimised.
    match = sourceNonsingletonNames.length === 0;
    let nonsingletonPerms = new Permutation(sourceNonsingletonNames);
    while(nonsingletonPerms.hasNext()) {
        let sourceNonsingletonNamesPerm = nonsingletonPerms.next();

        let dimMapUpdates = Object.assign({}, dimMap);
        let classNameMapUpdates = Object.assign({}, classNameMap);

        let nodeNameMapCopy = Object.assign({}, nodeNameMap);
        for (let i = 0; i < selfNonsingletonNames.length; i++) {
            nodeNameMapCopy[selfNonsingletonNames[i]] = sourceNonsingletonNamesPerm[i];
        }
        
        let nodesIter = {};

        for (let i = 0; i < selfNonsingletonNames.length; i++) {
            let selfName = selfNonsingletonNames[i];
            let sourceName = sourceNonsingletonNamesPerm[i];

            let [nodeMatch, dimMapUpdatesNew, classNameMapUpdatesNew] = matchNode(this.nodes[selfName],
                source.nodes[sourceName],
                dimMapUpdates,
                classNameMapUpdates,
                nodeNameMapCopy,
                this.categoryClasses,
                source.categoryClasses,
                buildMatching);
            
            if (buildMatching) {
                match = nodeMatch !== null;
            } else {
                match = nodeMatch;
            }
            
            if (match) {
                // If there was a match, we update the dimension and class name mappings.
                Object.assign(dimMapUpdates, dimMapUpdatesNew);
                Object.assign(classNameMapUpdates, classNameMapUpdatesNew);

                if (buildMatching) {
                    nodeMatch.srcName = sourceName;
                    nodesIter[selfName] = nodeMatch;
                }
            } else {
                // On the first failed match, we skip this permutation.
                break;
            }
        }

        // If we have found a matching, end the search.
        if (match) {
            // Add matched node names to name map.

            Object.assign(dimMap, dimMapUpdates);
            Object.assign(classNameMap, classNameMapUpdates);
            Object.assign(nodeNameMap, nodeNameMapCopy);

            if (buildMatching) {
                Object.assign(nodes, nodesIter);
            }
            break;
        }
    }
    
    // If no matching was found, we don't need to go on further.
    if (match === false) {
        if (buildMatching) {
            return null;
        } else {
            return false;
        }
    }
    
    // If a matching was found, build a resulting matching node if needed.
    if (buildMatching) {
        let classes = {};
        for (let categoryClassName in this.categoryClasses) {
            let categoryClass = this.categoryClasses[categoryClassName];
            classes[categoryClassName] = new Class(categoryClass.dim, classNameMap[categoryClassName]);
        }

        return new Schema(nodes, classes, this.cyclic, this.undirected, this.fanin, dimMap);

    } else {
        // Otherwise just return a boolean True.
        return true;
    }

}

function Node(isSingleton=false, fields=null, links=null, srcName=null) {

    if (fields === null) {
        fields = {};
    }
    if (links === null){
        links = {};
    }

    // Simple type checks.
    if (typeof isSingleton !== "boolean") {
        throw new SchemaException("Node singleton flag must be a boolean.");
    }
    if (typeof fields !== "object") {
        throw new SchemaException("Node fields must be a key-value dictionary.");
    }
    if (typeof links !== "object") {
        throw new SchemaException("Node links must be a key-value dictionary.");
    }
    if (srcName !== null) {
        if (typeof srcName !== "string") {
            throw new SchemaException("Source name must be a string.");
        }
        if (NAME_FORMAT.test(srcName) === false) {
            throw new SchemaException("Source name may contain lowercase letters, numbers and underscores. They must start with a letter.");
        }
    }
    
    // We have special restrictions for singleton nodes. They must have a single field and no links.
    if (isSingleton) {
        if (Object.keys(fields).length != 1) {
            throw new SchemaException("Singleton nodes must have a single field.");
        }
        if (Object.keys(links).length != 0) {
            throw new SchemaException("Singleton nodes cannot have links.");
        }
    }
    
    // Quantity check.
    if (Object.keys(fields).length + Object.keys(links).length < 1) {
        throw new SchemaException("Node must have at least one field or link.");
    }

    // Type and name checks for fields.
    for (let k in fields) {
        assert(typeof k === "string");
        assert(fields[k] instanceof Field);
        if (NAME_FORMAT.test(k) === false) {
            throw new SchemaException("Node field names may contain lowercase letters, numbers and underscores. \
                They must start with a letter.", "fields."+k);
        }
    }
    
    // Type and name checks for links.
    for (let k in links) {
        assert(typeof k === "string");
        assert(links[k] instanceof Link);
        if (NAME_FORMAT.test(k) === false) {
            throw new SchemaException("Node link targets may contain lowercase letters, numbers and underscores. \
                They must start with a letter.", "links."+k);
        }
    }

    this.isSingleton = isSingleton;
    this.fields = fields;
    this.links = links;
    this.srcName = srcName;
}

Node.prototype.isVariable = function() {
    for (let field in this.fields) {
        if (this.fields[field].isVariable()) {
            return true;
        }
    }
    return false;
}

function dumpNode(self) {

    let result = { "singleton" : self.isSingleton };

    if (self.isSingleton) {
        let field = dumpField(Object.values(self.fields)[0]);
        for (let k in field) {
            result[k] = field[k];
        }
    } else {

        let fields = {};
        for (k in self.fields) {
            fields[k] = dumpField(self.fields[k]);
        }
        result["fields"] = fields;

        let links = {};
        for (k in self.links) {
            links[k] = dumpLink(self.links[k]);
        }
        result["links"] = links;
    }
    
    if (self.srcName !== null) {
        result["src-name"] = self.srcName;
    }
    
    return result;
}

function loadNode(input) {

    let isSingleton = "singleton" in input ? input["singleton"] : false;
    let links = "links" in input ? input["links"] : {};
    let fields = "fields" in input ? input["fields"] : {};
    let srcName = "src-name" in input ? input["src-name"] : null;

    if (typeof fields === "object") {
        if (isSingleton && Object.keys(fields).length == 0) {
            try {
                fields["field"] = loadField(input);
            } catch(err) {
                throw err;
            }
        } else {
            for (let k in fields) {
                try {
                    fields[k] = loadField(fields[k]);
                } catch(err) {
                    if (err instanceof SchemaException) {
                        err.path = "fields." + k;
                    }
                    throw err
                }
            }
        }
    }

    if (typeof links === "object") {
        for (let k in links) {
            try {
                links[k] = loadLink(links[k]);
            } catch(err) {
                if (err instanceof SchemaException) {
                    err.path = "links." + k;
                }
                throw err;
            }
        }
    }
    
    return new Node(isSingleton, fields, links, srcName);
}

function matchNode(self, source, dimMap, classNameMap, nodeNameMap, selfClasses, sourceClasses, buildMatching=false) {
    assert(source instanceof Node);
    assert(typeof dimMap === "object");
    assert(typeof classNameMap === "object");
    assert(typeof nodeNameMap === "object");

    let fieldNameMap = {};
    let dimMapUpdates = {};
    let classNameMapUpdates = {};

    let selfTensorNames = [];
    let selfCategoryNames = [];
    for (let k in self.fields) {
        if (self.fields[k] instanceof Tensor) {
            selfTensorNames.push(k);
        } else if (self.fields[k] instanceof Category) {
            selfCategoryNames.push(k);
        }
    }

    let sourceTensorNames = [];
    let sourceCategoryNames = [];
    for (let k in source.fields) {
        if (source.fields[k] instanceof Tensor) {
            sourceTensorNames.push(k);
        } else if (source.fields[k] instanceof Category) {
            sourceCategoryNames.push(k);
        }
    }

    // Simply dismiss in case the counts don't match.
    if ((selfTensorNames.length !== sourceTensorNames.length) ||
        (selfCategoryNames.length !== sourceCategoryNames.length) ||
        (Object.keys(self.links).length !== Object.keys(source.links).length)) {
            if (buildMatching) {
                return [null, {}, {}];
            } else {
                return [false, {}, {}];
            }
        }

    // We first try to match links as this the cheapest operation. We match based on
    // the node matching which must be given when this function is called.
    for (let nodeName in self.links) {
        let link = self.links[nodeName];
        let source_node_name = nodeNameMap[nodeName];
        if ((source_node_name in source.links) === false || matchLink(link, source.links[source_node_name]) === false) {
            if (buildMatching) {
                return [null, {}, {}];
            } else {
                return [false, {}, {}];
            }
        }
    }

    // Try all possible tensor matchings. Individual field matchings are not independent
    // because of dimension matchings. Maybe this can be optimised.
    let match = sourceTensorNames.length === 0;
    let tensorPerms = new Permutation(sourceTensorNames);
    while(tensorPerms.hasNext()) {
        let sourceTensorNamesPerm = tensorPerms.next();
        let dimMapUpdatesIter = {};

        for (let i = 0; i < selfTensorNames.length; i++) {
            let selfName = selfTensorNames[i];
            let sourceName = sourceTensorNamesPerm[i];
            
            let dimMapUpdatesNew;
            [match, dimMapUpdatesNew] = matchTensor(self.fields[selfName],
                source.fields[sourceName],
                Object.assign({}, dimMapUpdatesIter, dimMap));

            if (match) {
                // If there was a match, we update the dimension mappings.
                Object.assign(dimMapUpdatesIter, dimMapUpdatesNew);
            } else {
                // On the first failed match, we skip this permutation.
                break;
            }
        }

        // If we have found a matching, end the search.
        if (match) {
            // Add matched field names to name map.
            Object.assign(dimMapUpdates, dimMapUpdatesIter);
            for (let i = 0; i < selfTensorNames.length; i++) {
                fieldNameMap[selfTensorNames[i]] = sourceTensorNamesPerm[i];
            }
            break;
        }
    }
    
    // If no matching was found, we don't need to go on further.
    if (match === false) {
        if (buildMatching) {
            return [null, {}, {}];
        } else {
            return [false, {}, {}];
        }
    }

    // Try all possible category matchings. Individual field matchings are not independent
    // because of dimension matchings. Maybe this can be optimised.
    match = sourceCategoryNames.length == 0;
    let categoryPerms = new Permutation(sourceCategoryNames);
    while(categoryPerms.hasNext()) {
        let sourceCategoryNamesPerm = categoryPerms.next();
        let dimMapUpdatesIter = Object.assign({}, dimMapUpdates);
        let classNameMapUpdatesIter = {};

        for (let i = 0; i < selfCategoryNames.length; i++) {
            let selfName = selfCategoryNames[i];
            let sourceName = sourceCategoryNamesPerm[i];

            let dimMapUpdatesNew, classNameMapUpdatesNew;
            [match, dimMapUpdatesNew, classNameMapUpdatesNew] = matchCategory(self.fields[selfName],
                source.fields[sourceName],
                Object.assign({}, dimMap, dimMapUpdatesIter),
                Object.assign({}, classNameMap, classNameMapUpdatesIter),
                selfClasses,
                sourceClasses);

            if (match) {
                // If there was a match, we update the dimension and class name mappings.
                Object.assign(dimMapUpdatesIter, dimMapUpdatesNew);
                Object.assign(classNameMapUpdatesIter, classNameMapUpdatesNew);
            } else {
                // On the first failed match, we skip this permutation.
                break;
            }

            match = true;
        }

        // If we have found a matching, end the search.
        if (match) {
            // Add matched field names to name map.
            Object.assign(dimMapUpdates, dimMapUpdatesIter);
            Object.assign(classNameMapUpdates, classNameMapUpdatesIter);

            for (let i = 0; i < selfCategoryNames.length; i++) {
                fieldNameMap[selfCategoryNames[i]] = sourceCategoryNamesPerm[i];
            }
            break;
        }
    }
    
    // If no matching was found, we don't need to go on further.
    if (match === false) {
        if (buildMatching) {
            return [null, {}, {}];
        } else {
            return [false, {}, {}];
        }
    }
    
    // If a matching was found, build a resulting matching node if needed.
    if (buildMatching) {

        let links = {};
        for (let k in self.links) {
            links[k] = loadLink(dumpLink(self.links[k]));
        }

        let fields = {};
        for (let k in self.fields) {
            fields[k] = loadField(dumpField(self.fields[k]));
        }

        for (let field_name in fields) {
            let field = fields[field_name];
            field.srcName = fieldNameMap[field_name];
            if (field.fieldType === "tensor") {
                // Copy the dimension array.
                field.srcDim = source.fields[field.srcName].dim.slice();
            }
        }

        return [new Node(self.isSingleton, fields, links), dimMapUpdates, classNameMapUpdates];

    } else {
        // Otherwise, just return a boolean along with name updates.
        return [true, dimMapUpdates, classNameMapUpdates];
    }
}

function Link(dim) {

    // If the link dimension is a list, then we expect two values. One for the upper and one for the lowr bound.
    if (Array.isArray(dim)) {
        if (dim.length != 2) {
            throw new SchemaException("Link dimension must be a list of two elements representing the upper and lower bound.");
        }

        if (Number.isInteger(dim[0]) === false || dim[0] < 0) {
            throw new SchemaException("Link lower bound must be a non-negative integer.");
        }

        if ((Number.isInteger(dim[1]) && dim[1] <= 0) || (Number.isInteger(dim[1]) ===  false && dim[1] !== "inf")) {
            throw new SchemaException("Link upper bound must be a positive integer or 'inf'.");
        }

        if (Number.isInteger(dim[1]) && dim[0] > dim[1]) {
            throw new SchemaException("Link lower bound cannot be greater than the upper bound.");
        }
        
        this.dim = dim;

    // We allow a link dimension to be an integer, in which case we set it as both the upper and lower bound.
    } else if (Number.isInteger(dim)) {
        if (dim < 1) {
            throw new SchemaException("Link dimension must be a positive integer.");
        }
        this.dim = [dim, dim];
        
    } else {
        throw new SchemaException("Link dimension must be either a positive integer or a two-element list.");
    }
}

function dumpLink(self) {
    return self.dim;
}

function loadLink(input) {
    return new Link(input);
}

function matchLink(self, source) {
    assert(source instanceof Link);

    if (self.dim[0] > source.dim[0]) {
        return false;
    } else {
        if (self.dim[1] === 'inf') {
            return true;
        } else if (source.dim[1] === 'inf' || self.dim[1] < source.dim[1]) {
            return false;
        }
    }

    return true;
}

function Field(fieldType, srcName=null) {
    assert(["tensor", "category"].indexOf(fieldType) >= 0);

    if (srcName !== null) {
        if (typeof srcName !== "string") {
            throw new SchemaException("Source name must be a string.");
        }
        if (NAME_FORMAT.test(srcName) === false) {
            throw new SchemaException("Source name may contain lowercase letters, numbers and underscores. They must start with a letter.");
        }
    }

    this.fieldType = fieldType;
    this.srcName = srcName;
}

function dumpField(self) {
    let result = {};
    switch (self.fieldType) {
        case "tensor":
            result = dumpTensor(self);
            break;
        case "category":
            result = dumpCategory(self);
            break;
    }
    result["type"] = self.fieldType;
    if (self.srcName !== null) {
        result["src-name"] = self.srcName;
    }
    return result;
}

function loadField(input) {

    let fieldType = "type" in input ? input["type"] : null;
    if (fieldType === null) {
        throw new SchemaException("Field must have a 'type' field.");
    }

    switch (fieldType) {
        case "tensor":
            return loadTensor(input);
        case "category":
            return loadCategory(input);
        default:
            throw new SchemaException("Unknown field type '" + fieldType + "'.");
    }
}


function Tensor(dim, srcName=null, srcDim=null) {
    Field.call(this, "tensor", srcName);

    // Simple type and value checks.
    if (Array.isArray(dim) === false) {
        throw new SchemaException("Tensor dim field must be a list of dimension definitions.");
    }
    if (dim.length < 1) {
        throw new SchemaException("Tensor must have at least one dimension.");
    }
    
    // Type and value checks for each dimension.
    for (let i in dim) {
        if (Number.isInteger(dim[i]) === false && typeof dim[i] !== "string") {
            throw new SchemaException("Tensor dim fields must all be integers or strings.");
        }
        if (Number.isInteger(dim[i]) && dim[i] < 1) {
            throw new SchemaException("Tensor dim fields that are integer must be positive numbers.");
        }
        if (typeof dim[i] === "string" && DIM_FORMAT.test(dim[i]) === false) {
            throw new SchemaException("Tensor dim fields that are strings may contain only lowercase \
                letters, numbers and underscores. They must start with a letter. \
                They may be suffixed by wildcard characters '?', '+' and '*' to denote variable count dimensions.");
        }   
    }
    
    // Type and value checks for each source dimension.
    if (srcDim !== null) {
        if (Array.isArray(dim) === false) {
            throw new SchemaException("Tensor source dim field must be a list of dimension definitions.");
        }
        for (let i in srcDim) {
            if (Number.isInteger(srcDim[i]) === false && typeof srcDim[i] !== "string") {
                throw new SchemaException("Tensor source dim fields must all be integers or strings.");
            }
            if (Number.isInteger(srcDim[i]) && srcDim[i] < 1) {
                throw new SchemaException("Tensor source dim fields that are integer must be positive numbers.");
            }
            if (typeof srcDim[i] === "string" && DIM_FORMAT.test(srcDim[i]) === false) {
                throw new SchemaException("Tensor source dim fields that are strings may contain only lowercase \
                    letters, numbers and underscores. They must start with a letter. \
                    They may be suffixed by wildcard characters '?', '+' and '*' to denote variable count dimensions.");
            }
        }
    }
    
    // Make sure that we have at most one variable count dimension.
    let foundWildcard = false;
    for (let i in dim) {
        if (typeof dim[i] === "string" && ["?", "+", "*"].indexOf(dim[i][dim[i].length - 1]) >= 0) {
            if (foundWildcard) {
                throw new SchemaException("Tensor can have at most one variable count dimension.");
            }
            foundWildcard = true;
        }
    }
    
    // If we have only one dimension, make sure it's not a variable count dimension which
    // allows zero dimensions.
    if (dim.length === 1 && typeof dim[0] === "string" && ["?", "*"].indexOf(dim[0][dim[0].length - 1]) >= 0) {
        throw new SchemaException("Tensors cannot have zero dimensions. \
            Having only one dimension suffixed with '?' or '*' permits this.");
    }

    this.dim = dim;
    this.srcDim = srcDim;
}

Tensor.prototype = Object.create(Field.prototype);
Tensor.prototype.constructor = Tensor;

Tensor.prototype.isVariable = function() {
    for (let i = 0; i < this.dim.length; i++) {
        if (Number.isInteger(this.dim[i]) === false) {
            return true;
        }
    }
    return false;
}

function dumpTensor(self) {
    let result = {};
    //let result = dumpField(self)
    result["dim"] = self.dim;
    if (self.srcDim !== null) {
        result["src-dim"] = self.srcDim;
    }
    return result;
}

function loadTensor(input) {
    let dim = "dim" in input ? input["dim"] : null;
    let srcName = "src-name" in input ? input["src-name"] : null;
    let srcDim = "src-dim" in input ? input["src-dim"] : null;

    if (dim === null) {
        throw new SchemaException("Tensor must have a 'dim' field.");
    }
    
    return new Tensor(dim, srcName, srcDim);
}

function matchTensor(self, source, dimMap) {
    assert(typeof dimMap === "object");
    assert(source instanceof Tensor);

    let [match, dimMapUpdate] = matchDimList(self.dim, source.dim, dimMap);
    return [match, dimMapUpdate];
}

function matchDimList(listA, listB, dimMap=null) {
    if (dimMap === null) {
        dimMap = {};
    }

    // Get first dimension and modifier of list A if possible.
    let [dimA, modA] = [null, null];
    if (listA.length > 0) {
        dimA = listA[0];
        if (typeof dimA === "string" && ['?', '+', '*'].indexOf(dimA[dimA.length - 1]) >= 0) {
            [dimA, modA] = [dimA.slice(0, -1), dimA.slice(-1)];
        }
    }
    
    // Get first dimension and modifier of list B if possible.
    let [dimB, modB] = [null, null];
    if (listB.length > 0) {
        dimB = listB[0];
        if (typeof dimB === "string" && ['?', '+', '*'].indexOf(dimB[dimB.length - 1]) >= 0) {
            [dimB, modB] = [dimB.slice(0, -1), dimB.slice(-1)];
        }
    }
    
    // If both lists are empty we simply return True.
    if (dimA === null && dimB === null) {
        return [true, {}];
    }
        
    // Handle the case when only list A is empty.
    if (dimA === null) {
        // If list A is empty, we can continue only if modifier B can be skipped.
        if (['?', '*'].indexOf(modB) >= 0) {
            return matchDimList(listA, listB.slice(1), dimMap);
        } else {
            return [false, {}];
        }
    }
    
    // Handle the case when only list B is empty.
    if (dimB === null) {
        // If list B is empty, we can continue only if modifier A can be skipped.
        if (['?', '*'].indexOf(modA) >= 0) {
            return matchDimList(listA.slice(1), listB, dimMap);
        } else {
            return [false, {}];
        }
    }
    
    // Check whether we can match the first dimensions.
    let match = typeof dimA === "string" && ((dimA in dimMap) === false || dimMap[dimA] === dimB) || Number.isInteger(dimA) && dimA === dimB;
    
    // If we can match we can try to move on.
    if (match) {
        let dimMapUpdate = {};
        if (typeof dimA === "string") {
            dimMapUpdate[dimA] = dimB;
        }
        let [recMatch, recDimMapUpdate] = matchDimList(listA.slice(1), listB.slice(1), Object.assign({}, dimMap, dimMapUpdate));
        if (recMatch) {
            return [true, Object.assign(recDimMapUpdate, dimMapUpdate)];
        }
    }
    
    // We can match and move dim A to match current B with more dims.
    if (match && ['+', '*'].indexOf(modB) >= 0) {
        let dimMapUpdate = {};
        if (typeof dimA === "string") {
            dimMapUpdate[dimA] = dimB;
        }
        let [recMatch, recDimMapUpdate] = matchDimList(listA.slice(1), listB, Object.assign({}, dimMap, dimMapUpdate));
        if (recMatch) {
            return [true, Object.assign(recDimMapUpdate, dimMapUpdate)];
        }
    }
    
    // We can skip dim A if it is skippable.
    if (['?', '*'].indexOf(modA) >= 0) {
        let [recMatch, recDimMapUpdate] = matchDimList(listA.slice(1), listB, dimMap);
        if (recMatch) {
            return [true, recDimMapUpdate];
        }
    }
    
    // We can match and move dim B to match current A with more dims.
    if (match && ['+', '*'].indexOf(modA) >= 0) {
        let dimMapUpdate = {};
        if (typeof dimA === "string") {
            dimMapUpdate[dimA] = dimB;
        }
        let [recMatch, recDimMapUpdate] = matchDimList(listA, listB.slice(1), Object.assign({}, dimMap, dimMapUpdate));
        if (recMatch) {
            return [true, Object.assign(recDimMapUpdate, dimMapUpdate)];
        }
    }

    // We can skip dim B if it is skippable.
    if (['?', '*'].indexOf(modB) >= 0) {
        let [recMatch, recDimMapUpdate] = matchDimList(listA, listB.slice(1), dimMap);
        if (recMatch) {
            return [true, recDimMapUpdate];
        }
    }

    return [false, {}];
}

function Category(categoryClass, srcName=null) {
    Field.call(this, "category", srcName);

    // Simple type and value checks.
    if (typeof categoryClass !== "string") {
        throw new SchemaException("Category class must be a string.");
    }
    if (NAME_FORMAT.test(categoryClass) === false) {
        throw new SchemaException("Category class may contain lowercase letters, numbers and underscores. They must start with a letter.");
    }

    this.categoryClass = categoryClass;
}

Category.prototype = Object.create(Field.prototype);
Category.prototype.constructor = Category;

Category.prototype.isVariable = function() {
    return false;
}

function dumpCategory(self) {
    let result = {};
    //let result = dumpField(self)
    result["class"] = self.categoryClass;
    return result;
}

function loadCategory(input) {
    
    let categoryClass = "class" in input ? input["class"] : null;
    let srcName = "src-name" in input ? input["src-name"] : null;

    if (categoryClass === null) {
        throw new SchemaException("Category must have a 'class' field.");
    }
    
    return new Category(categoryClass, srcName);
}

function matchCategory(self, source, dimMap, classNameMap, selfClasses, sourceClasses) {
    assert(typeof dimMap === "object");
    assert(typeof classNameMap === "object");
    assert(source instanceof Category);

    // If the class has already been mapped then we simply compare.
    if (self.categoryClass in classNameMap) {
        return [source.categoryClass === classNameMap[self.categoryClass], {}, {}];

    } else {
        let selfClass = selfClasses[self.categoryClass];
        let sourceClass = sourceClasses[source.categoryClass];

        let [match, dimMapUpdate] = matchClass(selfClass, sourceClass, dimMap);

        if (match) {
            let classNameMapUpdate = {};
            classNameMapUpdate[self.categoryClass] = source.categoryClass;
            return [true, dimMapUpdate, classNameMapUpdate];

        } else {
            return [false, {}, {}];
        }
    }
}

function Class(dim, srcName=null) {

    // Simple type and value checks.
    if (Number.isInteger(dim) === false && typeof dim !== "string") {
        throw new SchemaException("Class dimension must be an integer or a string.");
    }
    if (Number.isInteger(dim) && dim < 1) {
        throw new SchemaException("Class dimension must be a positive integer.");
    }
    if (typeof dim === "string" && NAME_FORMAT.test(dim) === false) {
        throw new SchemaException("Class dimension can contain lowercase letters, numbers and underscores. They must start with a letter.");
    }
    if (srcName !== null) {
        if (typeof srcName !== "string") {
            throw new SchemaException("Source name must be a string.");
        }
        if (NAME_FORMAT.test(srcName) === false) {
            throw new SchemaException("Source name may contain lowercase letters, numbers and underscores. They must start with a letter.");
        }
    }
        
    this.dim = dim;
    this.srcName = srcName;
}

Class.prototype.isVariable = function() {
    return Number.isInteger(this.dim) === false;
}

function dumpClass(self) {
    let result = {"dim" : self.dim}
    if (self.srcName !== null) {
        result["src-name"] = self.srcName;
    }
    return result;
}

function loadClass(input) {
    assert(typeof input === "object");

    let dim = "dim" in input ? input["dim"] : null;
    let srcName = "src-name" in input ? input["src-name"] : null;

    if (dim === null) {
        throw new SchemaException("Class must have a 'dim' field.");
    }

    return new Class(dim, srcName);
}

function matchClass(self, source, dimMap) {

    assert(typeof dimMap === "object");
    assert(source instanceof Class);

    if (Number.isInteger(self.dim)) {
        return [self.dim == source.dim, {}];

    } else {
        let dim = dimMap[self.dim] || null;

        if (dim === null) {
            let dimMapUpdate = {};
            dimMapUpdate[self.dim] = source.dim;
            return [true, dimMapUpdate];

        } else {
            return [dim === source.dim, {}];
        }
    }
}

module.exports = {
    "Schema" : Schema,
    "Node" : Node,
    "Link" : Link,
    "Tensor" : Tensor,
    "Category" : Category,
    "Class" : Class,
    "SchemaException": SchemaException,
    "load" : load,
};

},{"./iterative-permutation":3,"assert":5}]},{},[2]);
