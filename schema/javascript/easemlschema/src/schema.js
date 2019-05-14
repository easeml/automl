'use strict'

import assert from 'assert'
import Permutation from './util/iterative-permutation'

const NAME_FORMAT = new RegExp('^[a-z_][0-9a-z_]*$')
const DIM_FORMAT = new RegExp('^[a-z_][0-9a-z_]*[+?*]?$')

function SchemaException (message, path = '') {
  this.message = message
  this.path = path
}

function Schema (nodes, categoryClasses, cyclic = false, undirected = false, fanin = false, srcDims = null) {
  if (nodes === null) {
    nodes = {}
  }
  if (categoryClasses === null) {
    categoryClasses = {}
  }

  // Simple type checks.
  if (typeof nodes !== 'object') {
    throw new SchemaException('Schema nodes must be a key-value dictionary.')
  }
  if (typeof categoryClasses !== 'object') {
    throw new SchemaException('Schema category classes must be a key-value dictionary.')
  }
  if (typeof cyclic !== 'boolean') {
    throw new SchemaException('Schema cyclic flag must be a boolean.')
  }
  if (typeof undirected !== 'boolean') {
    throw new SchemaException('Schema undirected flag must be a boolean.')
  }
  if (typeof fanin !== 'boolean') {
    throw new SchemaException('Schema fan-in flag must be a boolean.')
  }

  // Quantity check.
  if (Object.keys(nodes).length < 1) {
    throw new SchemaException('Schema must have at least one node.')
  }

  // Type and name checks for nodes.
  for (let k in nodes) {
    assert(typeof k === 'string')
    assert(nodes[k] instanceof Node)
    if (NAME_FORMAT.test(k) === false) {
      throw new SchemaException('Schema node names may contain lowercase letters, numbers and underscores. ' +
                                'They must start with a letter.', 'nodes.' + k)
    }
  }

  // Type and name checks for category classes.
  for (let k in categoryClasses) {
    assert(typeof k === 'string')
    assert(categoryClasses[k] instanceof Class)
    if (NAME_FORMAT.test(k) === false) {
      throw new SchemaException('Schema category class names may contain lowercase letters, numbers and ' +
                                'underscores. They must start with a letter.', 'classes.' + k)
    }
  }

  // Reference checks.
  let orphanClasses = new Set(Object.keys(categoryClasses))
  for (let k in nodes) {
    // Check links.
    let linkCount = 0
    for (let l in nodes[k].links) {
      if (!(l in nodes)) {
        throw new SchemaException('Node link points to unknown node.', 'nodes.' + k + '.links.' + l)
      }
      if (nodes[l].isSingleton) {
        throw new SchemaException('Node link points to a singleton node.', 'nodes.' + k + '.links.' + l)
      }
      if (undirected && !fanin) {
        if (nodes[k].links[l].dim[1] === 'inf') {
          throw new SchemaException('Nodes in undirected schemas with fan-in cannot have infinite outgoing links.', 'nodes.' + k + '.links.' + l)
        }
        linkCount += nodes[k].links[l].dim[1]
      }
    }

    // Check category field classes.
    for (let f in nodes[k].fields) {
      if (nodes[k].fields[f] instanceof Category) {
        if (!(nodes[k].fields[f].categoryClass in categoryClasses)) {
          throw new SchemaException('Field category class undefined.', 'nodes.' + k + '.fields.' + f)
        }
        orphanClasses.delete(nodes[k].fields[f].categoryClass)
      }
    }

    if (undirected && !fanin && linkCount > 2) {
      throw new SchemaException('Nodes in undirected schemas with fan-in can have at most 2 outgoing links.', 'nodes.' + k)
    }
  }

  // Check if there are unreferenced classes.
  if (orphanClasses.size > 0) {
    const iterator1 = orphanClasses[Symbol.iterator]()
    throw new SchemaException('Every declared class must be referenced in a category.',
      'classes.' + iterator1.next().value)
  }

  // Source dimensions check.
  if (srcDims !== null) {
    if (typeof srcDims !== 'object') {
      throw new SchemaException('Source dimensions field must be a key-value dictionary.', 'src-dims')
    }
    for (let k in srcDims) {
      assert(typeof k === 'string')
      if (typeof srcDims[k] !== 'string' && !Number.isInteger(srcDims[k])) {
        throw new SchemaException('Source dimension values must be strings or integers.', 'src-dims.' + k)
      }
    }
  }

  this.nodes = nodes
  this.categoryClasses = categoryClasses
  this.cyclic = cyclic
  this.undirected = undirected
  this.fanin = fanin
  this.srcDims = srcDims
}

function load (input) {
  assert(typeof input === 'object')

  let nodes = 'nodes' in input ? input['nodes'] : null
  let constraints = 'ref-constraints' in input ? input['ref-constraints'] : {}
  let categoryClasses = 'classes' in input ? input['classes'] : {}
  let srcDims = 'src-dims' in input ? input['src-dims'] : null

  if (nodes !== null && typeof nodes === 'object') {
    for (let k in nodes) {
      try {
        nodes[k] = loadNode(nodes[k])
      } catch (err) {
        if (err instanceof SchemaException) {
          err.path = 'nodes.' + k
        }
        throw err
      }
    }
  }

  if (categoryClasses !== null && typeof categoryClasses === 'object') {
    for (let k in categoryClasses) {
      try {
        categoryClasses[k] = loadClass(categoryClasses[k])
      } catch (err) {
        if (err instanceof SchemaException) {
          err.path = 'classes.' + k
        }
        throw err
      }
    }
  } else {
    throw new SchemaException('Category classes must be a key-value dictionary.', 'classes')
  }

  let [cyclic, undirected, fanin] = [false, false, false]
  if (constraints !== null && typeof constraints === 'object' && (constraints instanceof Array) === false) {
    cyclic = 'cyclic' in constraints ? constraints['cyclic'] : false
    undirected = 'undirected' in constraints ? constraints['undirected'] : false
    fanin = 'fan-in' in constraints ? constraints['fan-in'] : false
  } else {
    throw new SchemaException('Reference constraints field must be a key-value dictionary.', 'ref-constraints')
  }

  return new Schema(nodes, categoryClasses, cyclic, undirected, fanin, srcDims)
}

Schema.prototype.dump = function () {
  let result = {
    'ref-constraints': {
      'cyclic': this.cyclic,
      'undirected': this.undirected,
      'fan-in': this.fanin
    }
  }

  let nodes = {}
  for (let k in this.nodes) {
    nodes[k] = dumpNode(this.nodes[k])
  }
  result['nodes'] = nodes

  if (Object.keys(this.categoryClasses).length > 0) {
    let categoryClasses = {}
    for (let k in this.categoryClasses) {
      categoryClasses[k] = dumpClass(this.categoryClasses[k])
    }
    result['classes'] = categoryClasses
  }

  if (this.srcDims !== null) {
    result['src-dims'] = this.srcDims
  }

  return result
}

Schema.prototype.isVariable = function () {
  for (let node in this.nodes) {
    if (this.nodes[node].isVariable()) {
      return true
    }
  }
  for (let categoryClass in this.categoryClasses) {
    if (this.categoryClasses[categoryClass].isVariable()) {
      return true
    }
  }
  return false
}

Schema.prototype.match = function (source, buildMatching = false) {
  assert(source instanceof Schema)

  let dimMap = {}
  let classNameMap = {}
  let nodeNameMap = {}
  let nodes = {}

  // Next we split the nodes into singletons and non-singletons.
  let selfSingletonNames = []
  let selfNonsingletonNames = []
  for (let k in this.nodes) {
    if (this.nodes[k].isSingleton) {
      selfSingletonNames.push(k)
    } else {
      selfNonsingletonNames.push(k)
    }
  }
  let sourceSingletonNames = []
  let sourceNonsingletonNames = []
  for (let k in source.nodes) {
    if (source.nodes[k].isSingleton) {
      sourceSingletonNames.push(k)
    } else {
      sourceNonsingletonNames.push(k)
    }
  }

  // We only compare referential constraints if there are non-singleton nodes.
  if (selfNonsingletonNames.length > 0) {
    if (source.cyclic && this.cyclic === false) {
      // A cyclic graph cannot be accepted by an acyclic destination.
      if (buildMatching) {
        return null
      } else {
        return false
      }
    }
    if (source.undirected === false && this.undirected) {
      // A directed graph cannot be accepted by an undirected destination.
      if (buildMatching) {
        return null
      } else {
        return false
      }
    }
    if (source.fanin && this.fanin === false) {
      // A graph that allows fan-in (multiple incoming pointers per node) cannot be accepted by
      // a destination that forbids it.
      if (buildMatching) {
        return null
      } else {
        return false
      }
    }
  }

  // Try all possible singleton node matchings. Individual matchings are not independent
  // because of various name matchings. Maybe this can be optimised.
  let match = sourceSingletonNames.length === 0
  let singletonPerms = new Permutation(sourceSingletonNames)
  while (singletonPerms.hasNext()) {
    let sourceSingletonNamesPerm = singletonPerms.next()
    let dimMapUpdates = {}
    let classNameMapUpdates = {}
    let nodesIter = {}

    for (let i = 0; i < sourceSingletonNames.length; i++) {
      let selfName = selfSingletonNames[i]
      let sourceName = sourceSingletonNamesPerm[i]

      let [nodeMatch, dimMapUpdatesNew, classNameMapUpdatesNew] = matchNode(this.nodes[selfName],
        source.nodes[sourceName],
        Object.assign({}, dimMap, dimMapUpdates),
        Object.assign({}, classNameMap, classNameMapUpdates),
        nodeNameMap,
        this.categoryClasses,
        source.categoryClasses,
        buildMatching)

      if (buildMatching) {
        match = nodeMatch !== null
      } else {
        match = nodeMatch
      }

      if (match) {
        // If there was a match, we update the dimension and class name mappings.
        Object.assign(dimMapUpdates, dimMapUpdatesNew)
        Object.assign(classNameMapUpdates, classNameMapUpdatesNew)

        if (buildMatching) {
          nodeMatch.srcName = sourceName
          nodesIter[selfName] = nodeMatch
        }
      } else {
        // On the first failed match, we skip this permutation.
        break
      }
    }

    // If we have found a matching, end the search.
    if (match) {
      // Add matched node names to name map.

      Object.assign(dimMap, dimMapUpdates)
      Object.assign(classNameMap, classNameMapUpdates)

      for (let i = 0; i < selfSingletonNames.length; i++) {
        nodeNameMap[selfSingletonNames[i]] = sourceSingletonNamesPerm[i]
      }

      if (buildMatching) {
        Object.assign(nodes, nodesIter)
      }
      break
    }
  }

  // If no matching was found, we don't need to go on further.
  if (match === false) {
    if (buildMatching) {
      return null
    } else {
      return false
    }
  }

  // Try all possible non-singleton node matchings. Individual matchings are not independent
  // because of various name matchings. Maybe this can be optimised.
  match = sourceNonsingletonNames.length === 0
  let nonsingletonPerms = new Permutation(sourceNonsingletonNames)
  while (nonsingletonPerms.hasNext()) {
    let sourceNonsingletonNamesPerm = nonsingletonPerms.next()

    let dimMapUpdates = Object.assign({}, dimMap)
    let classNameMapUpdates = Object.assign({}, classNameMap)

    let nodeNameMapCopy = Object.assign({}, nodeNameMap)
    for (let i = 0; i < selfNonsingletonNames.length; i++) {
      nodeNameMapCopy[selfNonsingletonNames[i]] = sourceNonsingletonNamesPerm[i]
    }

    let nodesIter = {}

    for (let i = 0; i < selfNonsingletonNames.length; i++) {
      let selfName = selfNonsingletonNames[i]
      let sourceName = sourceNonsingletonNamesPerm[i]

      let [nodeMatch, dimMapUpdatesNew, classNameMapUpdatesNew] = matchNode(this.nodes[selfName],
        source.nodes[sourceName],
        dimMapUpdates,
        classNameMapUpdates,
        nodeNameMapCopy,
        this.categoryClasses,
        source.categoryClasses,
        buildMatching)

      if (buildMatching) {
        match = nodeMatch !== null
      } else {
        match = nodeMatch
      }

      if (match) {
        // If there was a match, we update the dimension and class name mappings.
        Object.assign(dimMapUpdates, dimMapUpdatesNew)
        Object.assign(classNameMapUpdates, classNameMapUpdatesNew)

        if (buildMatching) {
          nodeMatch.srcName = sourceName
          nodesIter[selfName] = nodeMatch
        }
      } else {
        // On the first failed match, we skip this permutation.
        break
      }
    }

    // If we have found a matching, end the search.
    if (match) {
      // Add matched node names to name map.

      Object.assign(dimMap, dimMapUpdates)
      Object.assign(classNameMap, classNameMapUpdates)
      Object.assign(nodeNameMap, nodeNameMapCopy)

      if (buildMatching) {
        Object.assign(nodes, nodesIter)
      }
      break
    }
  }

  // If no matching was found, we don't need to go on further.
  if (match === false) {
    if (buildMatching) {
      return null
    } else {
      return false
    }
  }

  // If a matching was found, build a resulting matching node if needed.
  if (buildMatching) {
    let classes = {}
    for (let categoryClassName in this.categoryClasses) {
      let categoryClass = this.categoryClasses[categoryClassName]
      classes[categoryClassName] = new Class(categoryClass.dim, classNameMap[categoryClassName])
    }

    return new Schema(nodes, classes, this.cyclic, this.undirected, this.fanin, dimMap)
  } else {
    // Otherwise just return a boolean True.
    return true
  }
}

function Node (isSingleton = false, fields = null, links = null, srcName = null) {
  if (fields === null) {
    fields = {}
  }
  if (links === null) {
    links = {}
  }

  // Simple type checks.
  if (typeof isSingleton !== 'boolean') {
    throw new SchemaException('Node singleton flag must be a boolean.')
  }
  if (typeof fields !== 'object') {
    throw new SchemaException('Node fields must be a key-value dictionary.')
  }
  if (typeof links !== 'object') {
    throw new SchemaException('Node links must be a key-value dictionary.')
  }
  if (srcName !== null) {
    if (typeof srcName !== 'string') {
      throw new SchemaException('Source name must be a string.')
    }
    if (NAME_FORMAT.test(srcName) === false) {
      throw new SchemaException('Source name may contain lowercase letters, numbers and underscores. They must start with a letter.')
    }
  }

  // We have special restrictions for singleton nodes. They must have a single field and no links.
  if (isSingleton) {
    if (Object.keys(fields).length !== 1) {
      throw new SchemaException('Singleton nodes must have a single field.')
    }
    if (Object.keys(links).length !== 0) {
      throw new SchemaException('Singleton nodes cannot have links.')
    }
  }

  // Quantity check.
  if (Object.keys(fields).length + Object.keys(links).length < 1) {
    throw new SchemaException('Node must have at least one field or link.')
  }

  // Type and name checks for fields.
  for (let k in fields) {
    assert(typeof k === 'string')
    assert(fields[k] instanceof Field)
    if (NAME_FORMAT.test(k) === false) {
      throw new SchemaException('Node field names may contain lowercase letters, numbers and underscores. ' +
                                'They must start with a letter.', 'fields.' + k)
    }
  }

  // Type and name checks for links.
  for (let k in links) {
    assert(typeof k === 'string')
    assert(links[k] instanceof Link)
    if (NAME_FORMAT.test(k) === false) {
      throw new SchemaException('Node link targets may contain lowercase letters, numbers and underscores. ' +
                                'They must start with a letter.', 'links.' + k)
    }
  }

  this.isSingleton = isSingleton
  this.fields = fields
  this.links = links
  this.srcName = srcName
}

Node.prototype.isVariable = function () {
  for (let field in this.fields) {
    if (this.fields[field].isVariable()) {
      return true
    }
  }
  return false
}

function dumpNode (self) {
  let result = { 'singleton': self.isSingleton }

  if (self.isSingleton) {
    let field = dumpField(Object.values(self.fields)[0])
    for (let k in field) {
      result[k] = field[k]
    }
  } else {
    let fields = {}
    for (let k in self.fields) {
      fields[k] = dumpField(self.fields[k])
    }
    result['fields'] = fields

    let links = {}
    for (let k in self.links) {
      links[k] = dumpLink(self.links[k])
    }
    result['links'] = links
  }

  if (self.srcName !== null) {
    result['src-name'] = self.srcName
  }

  return result
}

function loadNode (input) {
  let isSingleton = 'singleton' in input ? input['singleton'] : false
  let links = 'links' in input ? input['links'] : {}
  let fields = 'fields' in input ? input['fields'] : {}
  let srcName = 'src-name' in input ? input['src-name'] : null

  if (typeof fields === 'object') {
    if (isSingleton && Object.keys(fields).length === 0) {
      try {
        fields['field'] = loadField(input)
      } catch (err) {
        throw err
      }
    } else {
      for (let k in fields) {
        try {
          fields[k] = loadField(fields[k])
        } catch (err) {
          if (err instanceof SchemaException) {
            err.path = 'fields.' + k
          }
          throw err
        }
      }
    }
  }

  if (typeof links === 'object') {
    for (let k in links) {
      try {
        links[k] = loadLink(links[k])
      } catch (err) {
        if (err instanceof SchemaException) {
          err.path = 'links.' + k
        }
        throw err
      }
    }
  }

  return new Node(isSingleton, fields, links, srcName)
}

function matchNode (self, source, dimMap, classNameMap, nodeNameMap, selfClasses, sourceClasses, buildMatching = false) {
  assert(source instanceof Node)
  assert(typeof dimMap === 'object')
  assert(typeof classNameMap === 'object')
  assert(typeof nodeNameMap === 'object')

  let fieldNameMap = {}
  let dimMapUpdates = {}
  let classNameMapUpdates = {}

  let selfTensorNames = []
  let selfCategoryNames = []
  for (let k in self.fields) {
    if (self.fields[k] instanceof Tensor) {
      selfTensorNames.push(k)
    } else if (self.fields[k] instanceof Category) {
      selfCategoryNames.push(k)
    }
  }

  let sourceTensorNames = []
  let sourceCategoryNames = []
  for (let k in source.fields) {
    if (source.fields[k] instanceof Tensor) {
      sourceTensorNames.push(k)
    } else if (source.fields[k] instanceof Category) {
      sourceCategoryNames.push(k)
    }
  }

  // Simply dismiss in case the counts don't match.
  if ((selfTensorNames.length !== sourceTensorNames.length) ||
        (selfCategoryNames.length !== sourceCategoryNames.length) ||
        (Object.keys(self.links).length !== Object.keys(source.links).length)) {
    if (buildMatching) {
      return [null, {}, {}]
    } else {
      return [false, {}, {}]
    }
  }

  // We first try to match links as this the cheapest operation. We match based on
  // the node matching which must be given when this function is called.
  for (let nodeName in self.links) {
    let link = self.links[nodeName]
    let sourceNodeName = nodeNameMap[nodeName]
    if ((sourceNodeName in source.links) === false || matchLink(link, source.links[sourceNodeName]) === false) {
      if (buildMatching) {
        return [null, {}, {}]
      } else {
        return [false, {}, {}]
      }
    }
  }

  // Try all possible tensor matchings. Individual field matchings are not independent
  // because of dimension matchings. Maybe this can be optimised.
  let match = sourceTensorNames.length === 0
  let tensorPerms = new Permutation(sourceTensorNames)
  while (tensorPerms.hasNext()) {
    let sourceTensorNamesPerm = tensorPerms.next()
    let dimMapUpdatesIter = {}

    for (let i = 0; i < selfTensorNames.length; i++) {
      let selfName = selfTensorNames[i]
      let sourceName = sourceTensorNamesPerm[i]

      let dimMapUpdatesNew;
      [match, dimMapUpdatesNew] = matchTensor(self.fields[selfName],
        source.fields[sourceName],
        Object.assign({}, dimMapUpdatesIter, dimMap))

      if (match) {
        // If there was a match, we update the dimension mappings.
        Object.assign(dimMapUpdatesIter, dimMapUpdatesNew)
      } else {
        // On the first failed match, we skip this permutation.
        break
      }
    }

    // If we have found a matching, end the search.
    if (match) {
      // Add matched field names to name map.
      Object.assign(dimMapUpdates, dimMapUpdatesIter)
      for (let i = 0; i < selfTensorNames.length; i++) {
        fieldNameMap[selfTensorNames[i]] = sourceTensorNamesPerm[i]
      }
      break
    }
  }

  // If no matching was found, we don't need to go on further.
  if (match === false) {
    if (buildMatching) {
      return [null, {}, {}]
    } else {
      return [false, {}, {}]
    }
  }

  // Try all possible category matchings. Individual field matchings are not independent
  // because of dimension matchings. Maybe this can be optimised.
  match = sourceCategoryNames.length === 0
  let categoryPerms = new Permutation(sourceCategoryNames)
  while (categoryPerms.hasNext()) {
    let sourceCategoryNamesPerm = categoryPerms.next()
    let dimMapUpdatesIter = Object.assign({}, dimMapUpdates)
    let classNameMapUpdatesIter = {}

    for (let i = 0; i < selfCategoryNames.length; i++) {
      let selfName = selfCategoryNames[i]
      let sourceName = sourceCategoryNamesPerm[i]

      let dimMapUpdatesNew, classNameMapUpdatesNew;
      [match, dimMapUpdatesNew, classNameMapUpdatesNew] = matchCategory(self.fields[selfName],
        source.fields[sourceName],
        Object.assign({}, dimMap, dimMapUpdatesIter),
        Object.assign({}, classNameMap, classNameMapUpdatesIter),
        selfClasses,
        sourceClasses)

      if (match) {
        // If there was a match, we update the dimension and class name mappings.
        Object.assign(dimMapUpdatesIter, dimMapUpdatesNew)
        Object.assign(classNameMapUpdatesIter, classNameMapUpdatesNew)
      } else {
        // On the first failed match, we skip this permutation.
        break
      }

      match = true
    }

    // If we have found a matching, end the search.
    if (match) {
      // Add matched field names to name map.
      Object.assign(dimMapUpdates, dimMapUpdatesIter)
      Object.assign(classNameMapUpdates, classNameMapUpdatesIter)

      for (let i = 0; i < selfCategoryNames.length; i++) {
        fieldNameMap[selfCategoryNames[i]] = sourceCategoryNamesPerm[i]
      }
      break
    }
  }

  // If no matching was found, we don't need to go on further.
  if (match === false) {
    if (buildMatching) {
      return [null, {}, {}]
    } else {
      return [false, {}, {}]
    }
  }

  // If a matching was found, build a resulting matching node if needed.
  if (buildMatching) {
    let links = {}
    for (let k in self.links) {
      links[k] = loadLink(dumpLink(self.links[k]))
    }

    let fields = {}
    for (let k in self.fields) {
      fields[k] = loadField(dumpField(self.fields[k]))
    }

    for (let fieldName in fields) {
      let field = fields[fieldName]
      field.srcName = fieldNameMap[fieldName]
      if (field.fieldType === 'tensor') {
        // Copy the dimension array.
        field.srcDim = source.fields[field.srcName].dim.slice()
      }
    }

    return [new Node(self.isSingleton, fields, links), dimMapUpdates, classNameMapUpdates]
  } else {
    // Otherwise, just return a boolean along with name updates.
    return [true, dimMapUpdates, classNameMapUpdates]
  }
}

function Link (dim) {
  // If the link dimension is a list, then we expect two values. One for the upper and one for the lowr bound.
  if (Array.isArray(dim)) {
    if (dim.length !== 2) {
      throw new SchemaException('Link dimension must be a list of two elements representing the upper and lower bound.')
    }

    if (Number.isInteger(dim[0]) === false || dim[0] < 0) {
      throw new SchemaException('Link lower bound must be a non-negative integer.')
    }

    if ((Number.isInteger(dim[1]) && dim[1] <= 0) || (Number.isInteger(dim[1]) === false && dim[1] !== 'inf')) {
      throw new SchemaException("Link upper bound must be a positive integer or 'inf'.")
    }

    if (Number.isInteger(dim[1]) && dim[0] > dim[1]) {
      throw new SchemaException('Link lower bound cannot be greater than the upper bound.')
    }

    this.dim = dim

    // We allow a link dimension to be an integer, in which case we set it as both the upper and lower bound.
  } else if (Number.isInteger(dim)) {
    if (dim < 1) {
      throw new SchemaException('Link dimension must be a positive integer.')
    }
    this.dim = [dim, dim]
  } else {
    throw new SchemaException('Link dimension must be either a positive integer or a two-element list.')
  }
}

function dumpLink (self) {
  return self.dim
}

function loadLink (input) {
  return new Link(input)
}

function matchLink (self, source) {
  assert(source instanceof Link)

  if (self.dim[0] > source.dim[0]) {
    return false
  } else {
    if (self.dim[1] === 'inf') {
      return true
    } else if (source.dim[1] === 'inf' || self.dim[1] < source.dim[1]) {
      return false
    }
  }

  return true
}

function Field (fieldType, srcName = null) {
  assert(['tensor', 'category'].indexOf(fieldType) >= 0)

  if (srcName !== null) {
    if (typeof srcName !== 'string') {
      throw new SchemaException('Source name must be a string.')
    }
    if (NAME_FORMAT.test(srcName) === false) {
      throw new SchemaException('Source name may contain lowercase letters, numbers and underscores. They must start with a letter.')
    }
  }

  this.fieldType = fieldType
  this.srcName = srcName
}

function dumpField (self) {
  let result = {}
  switch (self.fieldType) {
    case 'tensor':
      result = dumpTensor(self)
      break
    case 'category':
      result = dumpCategory(self)
      break
  }
  result['type'] = self.fieldType
  if (self.srcName !== null) {
    result['src-name'] = self.srcName
  }
  return result
}

function loadField (input) {
  let fieldType = 'type' in input ? input['type'] : null
  if (fieldType === null) {
    throw new SchemaException("Field must have a 'type' field.")
  }

  switch (fieldType) {
    case 'tensor':
      return loadTensor(input)
    case 'category':
      return loadCategory(input)
    default:
      throw new SchemaException("Unknown field type '" + fieldType + "'.")
  }
}

function Tensor (dim, srcName = null, srcDim = null) {
  Field.call(this, 'tensor', srcName)

  // Simple type and value checks.
  if (Array.isArray(dim) === false) {
    throw new SchemaException('Tensor dim field must be a list of dimension definitions.')
  }
  if (dim.length < 1) {
    throw new SchemaException('Tensor must have at least one dimension.')
  }

  // Type and value checks for each dimension.
  for (let i in dim) {
    if (Number.isInteger(dim[i]) === false && typeof dim[i] !== 'string') {
      throw new SchemaException('Tensor dim fields must all be integers or strings.')
    }
    if (Number.isInteger(dim[i]) && dim[i] < 1) {
      throw new SchemaException('Tensor dim fields that are integer must be positive numbers.')
    }
    if (typeof dim[i] === 'string' && DIM_FORMAT.test(dim[i]) === false) {
      throw new SchemaException('Tensor dim fields that are strings may contain only lowercase ' +
                                'letters, numbers and underscores. They must start with a letter. ' +
                                'They may be suffixed by wildcard characters \'?\', \'+\' and \'*\' to ' +
                                'denote variable count dimensions.')
    }
  }

  // Type and value checks for each source dimension.
  if (srcDim !== null) {
    if (Array.isArray(dim) === false) {
      throw new SchemaException('Tensor source dim field must be a list of dimension definitions.')
    }
    for (let i in srcDim) {
      if (Number.isInteger(srcDim[i]) === false && typeof srcDim[i] !== 'string') {
        throw new SchemaException('Tensor source dim fields must all be integers or strings.')
      }
      if (Number.isInteger(srcDim[i]) && srcDim[i] < 1) {
        throw new SchemaException('Tensor source dim fields that are integer must be positive numbers.')
      }
      if (typeof srcDim[i] === 'string' && DIM_FORMAT.test(srcDim[i]) === false) {
        throw new SchemaException('Tensor source dim fields that are strings may contain only lowercase ' +
                                  'letters, numbers and underscores. They must start with a letter. ' +
                                  'They may be suffixed by wildcard characters \'?\', \'+\' and \'*\' to ' +
                                  'denote variable count dimensions.')
      }
    }
  }

  // Make sure that we have at most one variable count dimension.
  let foundWildcard = false
  for (let i in dim) {
    if (typeof dim[i] === 'string' && ['?', '+', '*'].indexOf(dim[i][dim[i].length - 1]) >= 0) {
      if (foundWildcard) {
        throw new SchemaException('Tensor can have at most one variable count dimension.')
      }
      foundWildcard = true
    }
  }

  // If we have only one dimension, make sure it's not a variable count dimension which
  // allows zero dimensions.
  if (dim.length === 1 && typeof dim[0] === 'string' && ['?', '*'].indexOf(dim[0][dim[0].length - 1]) >= 0) {
    throw new SchemaException('Tensors cannot have zero dimensions. ' +
                              'Having only one dimension suffixed with \'?\' or \'*\' permits this.')
  }

  this.dim = dim
  this.srcDim = srcDim
}

Tensor.prototype = Object.create(Field.prototype)
Tensor.prototype.constructor = Tensor

Tensor.prototype.isVariable = function () {
  for (let i = 0; i < this.dim.length; i++) {
    if (Number.isInteger(this.dim[i]) === false) {
      return true
    }
  }
  return false
}

function dumpTensor (self) {
  let result = {}
  // let result = dumpField(self)
  result['dim'] = self.dim
  if (self.srcDim !== null) {
    result['src-dim'] = self.srcDim
  }
  return result
}

function loadTensor (input) {
  let dim = 'dim' in input ? input['dim'] : null
  let srcName = 'src-name' in input ? input['src-name'] : null
  let srcDim = 'src-dim' in input ? input['src-dim'] : null

  if (dim === null) {
    throw new SchemaException("Tensor must have a 'dim' field.")
  }

  return new Tensor(dim, srcName, srcDim)
}

function matchTensor (self, source, dimMap) {
  assert(typeof dimMap === 'object')
  assert(source instanceof Tensor)

  let [match, dimMapUpdate] = matchDimList(self.dim, source.dim, dimMap)
  return [match, dimMapUpdate]
}

function matchDimList (listA, listB, dimMap = null) {
  if (dimMap === null) {
    dimMap = {}
  }

  // Get first dimension and modifier of list A if possible.
  let [dimA, modA] = [null, null]
  if (listA.length > 0) {
    dimA = listA[0]
    if (typeof dimA === 'string' && ['?', '+', '*'].indexOf(dimA[dimA.length - 1]) >= 0) {
      [dimA, modA] = [dimA.slice(0, -1), dimA.slice(-1)]
    }
  }

  // Get first dimension and modifier of list B if possible.
  let [dimB, modB] = [null, null]
  if (listB.length > 0) {
    dimB = listB[0]
    if (typeof dimB === 'string' && ['?', '+', '*'].indexOf(dimB[dimB.length - 1]) >= 0) {
      [dimB, modB] = [dimB.slice(0, -1), dimB.slice(-1)]
    }
  }

  // If both lists are empty we simply return True.
  if (dimA === null && dimB === null) {
    return [true, {}]
  }

  // Handle the case when only list A is empty.
  if (dimA === null) {
    // If list A is empty, we can continue only if modifier B can be skipped.
    if (['?', '*'].indexOf(modB) >= 0) {
      return matchDimList(listA, listB.slice(1), dimMap)
    } else {
      return [false, {}]
    }
  }

  // Handle the case when only list B is empty.
  if (dimB === null) {
    // If list B is empty, we can continue only if modifier A can be skipped.
    if (['?', '*'].indexOf(modA) >= 0) {
      return matchDimList(listA.slice(1), listB, dimMap)
    } else {
      return [false, {}]
    }
  }

  // Check whether we can match the first dimensions.
  let match = (typeof dimA === 'string' && ((dimA in dimMap) === false || dimMap[dimA] === dimB)) || (Number.isInteger(dimA) && dimA === dimB)

  // If we can match we can try to move on.
  if (match) {
    let dimMapUpdate = {}
    if (typeof dimA === 'string') {
      dimMapUpdate[dimA] = dimB
    }
    let [recMatch, recDimMapUpdate] = matchDimList(listA.slice(1), listB.slice(1), Object.assign({}, dimMap, dimMapUpdate))
    if (recMatch) {
      return [true, Object.assign(recDimMapUpdate, dimMapUpdate)]
    }
  }

  // We can match and move dim A to match current B with more dims.
  if (match && ['+', '*'].indexOf(modB) >= 0) {
    let dimMapUpdate = {}
    if (typeof dimA === 'string') {
      dimMapUpdate[dimA] = dimB
    }
    let [recMatch, recDimMapUpdate] = matchDimList(listA.slice(1), listB, Object.assign({}, dimMap, dimMapUpdate))
    if (recMatch) {
      return [true, Object.assign(recDimMapUpdate, dimMapUpdate)]
    }
  }

  // We can skip dim A if it is skippable.
  if (['?', '*'].indexOf(modA) >= 0) {
    let [recMatch, recDimMapUpdate] = matchDimList(listA.slice(1), listB, dimMap)
    if (recMatch) {
      return [true, recDimMapUpdate]
    }
  }

  // We can match and move dim B to match current A with more dims.
  if (match && ['+', '*'].indexOf(modA) >= 0) {
    let dimMapUpdate = {}
    if (typeof dimA === 'string') {
      dimMapUpdate[dimA] = dimB
    }
    let [recMatch, recDimMapUpdate] = matchDimList(listA, listB.slice(1), Object.assign({}, dimMap, dimMapUpdate))
    if (recMatch) {
      return [true, Object.assign(recDimMapUpdate, dimMapUpdate)]
    }
  }

  // We can skip dim B if it is skippable.
  if (['?', '*'].indexOf(modB) >= 0) {
    let [recMatch, recDimMapUpdate] = matchDimList(listA, listB.slice(1), dimMap)
    if (recMatch) {
      return [true, recDimMapUpdate]
    }
  }

  return [false, {}]
}

function Category (categoryClass, srcName = null) {
  Field.call(this, 'category', srcName)

  // Simple type and value checks.
  if (typeof categoryClass !== 'string') {
    throw new SchemaException('Category class must be a string.')
  }
  if (NAME_FORMAT.test(categoryClass) === false) {
    throw new SchemaException('Category class may contain lowercase letters, numbers and underscores. They must start with a letter.')
  }

  this.categoryClass = categoryClass
}

Category.prototype = Object.create(Field.prototype)
Category.prototype.constructor = Category

Category.prototype.isVariable = function () {
  return false
}

function dumpCategory (self) {
  let result = {}
  // let result = dumpField(self)
  result['class'] = self.categoryClass
  return result
}

function loadCategory (input) {
  let categoryClass = 'class' in input ? input['class'] : null
  let srcName = 'src-name' in input ? input['src-name'] : null

  if (categoryClass === null) {
    throw new SchemaException("Category must have a 'class' field.")
  }

  return new Category(categoryClass, srcName)
}

function matchCategory (self, source, dimMap, classNameMap, selfClasses, sourceClasses) {
  assert(typeof dimMap === 'object')
  assert(typeof classNameMap === 'object')
  assert(source instanceof Category)

  // If the class has already been mapped then we simply compare.
  if (self.categoryClass in classNameMap) {
    return [source.categoryClass === classNameMap[self.categoryClass], {}, {}]
  } else {
    let selfClass = selfClasses[self.categoryClass]
    let sourceClass = sourceClasses[source.categoryClass]

    let [match, dimMapUpdate] = matchClass(selfClass, sourceClass, dimMap)

    if (match) {
      let classNameMapUpdate = {}
      classNameMapUpdate[self.categoryClass] = source.categoryClass
      return [true, dimMapUpdate, classNameMapUpdate]
    } else {
      return [false, {}, {}]
    }
  }
}

function Class (dim, srcName = null) {
  // Simple type and value checks.
  if (Number.isInteger(dim) === false && typeof dim !== 'string') {
    throw new SchemaException('Class dimension must be an integer or a string.')
  }
  if (Number.isInteger(dim) && dim < 1) {
    throw new SchemaException('Class dimension must be a positive integer.')
  }
  if (typeof dim === 'string' && NAME_FORMAT.test(dim) === false) {
    throw new SchemaException('Class dimension can contain lowercase letters, numbers and underscores. They must start with a letter.')
  }
  if (srcName !== null) {
    if (typeof srcName !== 'string') {
      throw new SchemaException('Source name must be a string.')
    }
    if (NAME_FORMAT.test(srcName) === false) {
      throw new SchemaException('Source name may contain lowercase letters, numbers and underscores. They must start with a letter.')
    }
  }

  this.dim = dim
  this.srcName = srcName
}

Class.prototype.isVariable = function () {
  return Number.isInteger(this.dim) === false
}

function dumpClass (self) {
  let result = { 'dim': self.dim }
  if (self.srcName !== null) {
    result['src-name'] = self.srcName
  }
  return result
}

function loadClass (input) {
  assert(typeof input === 'object')

  let dim = 'dim' in input ? input['dim'] : null
  let srcName = 'src-name' in input ? input['src-name'] : null

  if (dim === null) {
    throw new SchemaException("Class must have a 'dim' field.")
  }

  return new Class(dim, srcName)
}

function matchClass (self, source, dimMap) {
  assert(typeof dimMap === 'object')
  assert(source instanceof Class)

  if (Number.isInteger(self.dim)) {
    return [self.dim === source.dim, {}]
  } else {
    let dim = dimMap[self.dim] || null

    if (dim === null) {
      let dimMapUpdate = {}
      dimMapUpdate[self.dim] = source.dim
      return [true, dimMapUpdate]
    } else {
      return [dim === source.dim, {}]
    }
  }
}

export default {
  'Schema': Schema,
  'Node': Node,
  'Link': Link,
  'Tensor': Tensor,
  'Category': Category,
  'Class': Class,
  'SchemaException': SchemaException,
  'load': load
}
