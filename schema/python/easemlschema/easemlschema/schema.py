import argparse
import copy
import itertools
import json
import re
import sys


NAME_FORMAT = re.compile(r"^[a-z_][0-9a-z_]*\Z", re.ASCII)
DIM_FORMAT = re.compile(r"^[a-z_][0-9a-z_]*[+?*]?\Z", re.ASCII)


class SchemaException(Exception):

    def __init__(self, message, path=""):
        self.message = message
        self.path = path


class Schema:

    def __init__(
            self,
            nodes=None,
            category_classes=None,
            cyclic=False,
            undirected=False,
            fanin=False,
            src_dims=None):

        if nodes is None:
            nodes = {}
        if category_classes is None:
            category_classes = {}

        # Simple type checks.
        if not isinstance(nodes, dict):
            raise SchemaException(
                "Schema nodes must be a key-value dictionary.")
        if not isinstance(category_classes, dict):
            raise SchemaException(
                "Schema category classes must be a key-value dictionary.")
        if not isinstance(cyclic, bool):
            raise SchemaException("Schema cyclic flag must be a boolean.")
        if not isinstance(undirected, bool):
            raise SchemaException("Schema undirected flag must be a boolean.")
        if not isinstance(fanin, bool):
            raise SchemaException("Schema fan-in flag must be a boolean.")

        # Quantity check.
        if len(nodes) < 1:
            raise SchemaException("Schema must have at least one node.")

        # Type and name checks for nodes.
        for k in nodes:
            assert(isinstance(k, str))
            assert(isinstance(nodes[k], Node))
            if re.match(NAME_FORMAT, k) is None:
                raise SchemaException(
                    "Schema node names may contain lowercase letters, numbers and underscores. "
                    "They must start with a letter.", "nodes." + k)

        # Type and name checks for category classes.
        for k in category_classes:
            assert(isinstance(k, str))
            assert(isinstance(category_classes[k], Class))
            if re.match(NAME_FORMAT, k) is None:
                raise SchemaException(
                    "Schema category class names may contain lowercase letters, numbers and "
                    "underscores. They must start with a letter.", "classes." + k)

        # Reference checks.
        orphan_classes = set(category_classes.keys())
        for k in nodes:
            # Check links.
            link_count = 0
            for l in nodes[k].links:
                if l not in nodes:
                    raise SchemaException(
                        "Node link points to unknown node.",
                        "nodes." + k + ".links." + l)
                if nodes[l].is_singleton:
                    raise SchemaException(
                        "Node link points to a singleton node.",
                        "nodes." + k + ".links." + l)
                if undirected and not fanin:
                    if nodes[k].links[l].dim[1] == "inf":
                        raise SchemaException(
                            "Nodes in undirected schemas with fan-in cannot have infinite outgoing links.",
                            "nodes." + k + ".links." + l)
                    link_count += nodes[k].links[l].dim[1]

            # Check category field classes.
            for f in nodes[k].fields:
                if isinstance(nodes[k].fields[f], Category):
                    if nodes[k].fields[f].category_class not in category_classes:
                        raise SchemaException(
                            "Field category class undefined.",
                            "nodes." + k + ".fields." + f)
                    orphan_classes.discard(nodes[k].fields[f].category_class)

            if undirected and not fanin and link_count > 2:
                raise SchemaException(
                    "Nodes in undirected schemas with fan-in can have at most 2 outgoing links.",
                    "nodes." + k)

        # Check if there are unreferenced classes.
        if len(orphan_classes) > 0:
            raise SchemaException(
                "Every declared class must be referenced in a category.",
                "classes." + orphan_classes.pop())

        # Source dimensions check.
        if src_dims is not None:
            if not isinstance(src_dims, dict):
                raise SchemaException(
                    "Source dimensions field must be a key-value dictionary.", "src-dims")
            for k, v in src_dims.items():
                assert(isinstance(k, str))
                if not isinstance(v, str) and not isinstance(v, int):
                    raise SchemaException(
                        "Source dimension values must be strings or integers.",
                        "src-dims." + k)

        self.nodes = nodes
        self.category_classes = category_classes
        self.cyclic = cyclic
        self.undirected = undirected
        self.fanin = fanin
        self.src_dims = src_dims

    def dump(self):

        result = {
            "ref-constraints": {
                "cyclic": self.cyclic,
                "undirected": self.undirected,
                "fan-in": self.fanin
            }
        }

        result["nodes"] = dict([(k, v._dump())
                                for (k, v) in self.nodes.items()])

        if len(self.category_classes) > 0:
            result["classes"] = dict([(k, v._dump())
                                      for (k, v) in self.category_classes.items()])

        if self.src_dims is not None:
            result["src-dims"] = self.src_dims

        return result

    @staticmethod
    def load(input):
        assert(isinstance(input, dict))

        nodes = input.get("nodes", None)
        constraints = input.get("ref-constraints", {})
        category_classes = input.get("classes", {})
        src_dims = input.get("src-dims", None)

        if isinstance(nodes, dict):
            for k in nodes:
                try:
                    nodes[k] = Node._load(nodes[k])
                except SchemaException as se:
                    se.path = "nodes." + k
                    raise se

        if isinstance(category_classes, dict):
            for k in category_classes:
                try:
                    category_classes[k] = Class._load(category_classes[k])
                except SchemaException as se:
                    se.path = "classes." + k
                    raise se
        else:
            raise SchemaException(
                "Category classes must be a key-value dictionary.", "classes")

        if isinstance(constraints, dict):
            cyclic = constraints.get("cyclic", False)
            undirected = constraints.get("undirected", False)
            fanin = constraints.get("fan-in", False)
        else:
            raise SchemaException(
                "Reference constraints field must be a key-value dictionary.",
                "ref-constraints")

        return Schema(
            nodes,
            category_classes,
            cyclic,
            undirected,
            fanin,
            src_dims)

    def match(self, source, build_matching=False):
        assert(isinstance(source, Schema))

        dim_map = {}
        class_name_map = {}
        node_name_map = {}
        nodes = {}

        # Start by matching constraints as they are the cheapest.

        # Next we split the nodes into singletons and non-singletons.
        self_singleton_names = [
            name for name in self.nodes if self.nodes[name].is_singleton]
        self_nonsingleton_names = [
            name for name in self.nodes if self.nodes[name].is_singleton is False]
        source_singleton_names = [
            name for name in source.nodes if source.nodes[name].is_singleton]
        source_nonsingleton_names = [
            name for name in source.nodes if source.nodes[name].is_singleton is False]

        # We only compare referential constraints if there are non-singleton
        # nodes.
        if len(self_nonsingleton_names) > 0:
            if source.cyclic and not self.cyclic:
                # A cyclic graph cannot be accepted by an acyclic destination.
                return None if build_matching else False
            if not source.undirected and self.undirected:
                # A directed graph cannot be accepted by an undirected
                # destination.
                return None if build_matching else False
            if source.fanin and not self.fanin:
                # A graph that allows fan-in (multiple incoming pointers per node) cannot be accepted by
                # a destination that forbids it.
                return None if build_matching else False

        # Try all possible singleton node matchings. Individual matchings are not independent
        # because of various name matchings. Maybe this can be optimised.
        match = len(source_singleton_names) == 0
        for source_singleton_names_perm in itertools.permutations(
                source_singleton_names):
            dim_map_updates = {}
            class_name_map_updates = {}
            nodes_iter = {}
            for (
                    self_name,
                    source_name) in zip(
                    self_singleton_names,
                    source_singleton_names_perm):

                node_match, dim_map_updates_new, class_name_map_updates_new = self.nodes[self_name]._match(
                    source.nodes[source_name],
                    {**dim_map, **dim_map_updates},
                    {**class_name_map, **class_name_map_updates},
                    node_name_map,
                    self.category_classes,
                    source.category_classes,
                    build_matching)

                match = (
                    node_match is not None) if build_matching else node_match

                if match:
                    # If there was a match, we update the dimension and class
                    # name mappings.
                    dim_map_updates.update(dim_map_updates_new)
                    class_name_map_updates.update(class_name_map_updates_new)
                    if build_matching:
                        node_match.src_name = source_name
                        nodes_iter[self_name] = node_match
                else:
                    # On the first failed match, we skip this permutation.
                    break

            # If we have found a matching, end the search.
            if match:
                # Add matched node names to name map.
                dim_map.update(dim_map_updates)
                class_name_map.update(class_name_map_updates)
                node_name_map = dict(
                    zip(self_singleton_names, source_singleton_names_perm))
                if build_matching:
                    nodes.update(nodes_iter)
                break

        # If no matching was found, we don't need to go on further.
        if not match:
            return None if build_matching else False

        # Try all possible non-singleton node matchings. Individual matchings are not independent
        # because of various name matchings. Maybe this can be optimised.
        match = len(source_nonsingleton_names) == 0
        for source_nonsingleton_names_perm in itertools.permutations(
                source_nonsingleton_names):
            dim_map_updates = copy.deepcopy(dim_map)
            class_name_map_updates = copy.deepcopy(class_name_map)
            node_name_map_copy = {**node_name_map,
                                  **dict(zip(self_nonsingleton_names,
                                             source_nonsingleton_names_perm))}
            nodes_iter = {}
            for (
                    self_name,
                    source_name) in zip(
                    self_nonsingleton_names,
                    source_nonsingleton_names_perm):

                node_match, dim_map_updates_new, class_name_map_updates_new = self.nodes[self_name]._match(
                    source.nodes[source_name],
                    dim_map_updates,
                    class_name_map_updates,
                    node_name_map_copy,
                    self.category_classes,
                    source.category_classes,
                    build_matching)

                match = (
                    node_match is not None) if build_matching else node_match

                if match:
                    # If there was a match, we update the dimension and class
                    # name mappings.
                    dim_map_updates.update(dim_map_updates_new)
                    class_name_map_updates.update(class_name_map_updates_new)
                    if build_matching:
                        node_match.src_name = source_name
                        nodes_iter[self_name] = node_match
                else:
                    # On the first failed match, we skip this permutation.
                    break

            # If we have found a matching, end the search.
            if match:
                # Add matched node names to name map.
                dim_map.update(dim_map_updates)
                class_name_map.update(class_name_map_updates)
                node_name_map.update(node_name_map_copy)
                if build_matching:
                    nodes.update(nodes_iter)
                break

        # If no matching was found, we don't need to go on further.
        if not match:
            return None if build_matching else False

        # If a matching was found, build a resulting matching node if needed.
        if build_matching:
            classes = {}
            for category_class_name, category_class in self.category_classes.items():
                classes[category_class_name] = Class(
                    category_class.dim, class_name_map[category_class_name])

            return Schema(
                nodes,
                classes,
                self.cyclic,
                self.undirected,
                self.fanin,
                dim_map)

        else:
            # Otherwise just return a boolean True.
            return True

    def is_variable(self):
        for node in self.nodes.values():
            if node.is_variable():
                return True
        for category_class in self.category_classes.values():
            if category_class.is_variable():
                return True
        return False


class Node:

    def __init__(
            self,
            is_singleton=False,
            fields=None,
            links=None,
            src_name=None):

        if fields is None:
            fields = {}
        if links is None:
            links = {}

        # Simple type checks.
        if not isinstance(is_singleton, bool):
            raise SchemaException("Node singleton flag must be a boolean.")
        if not isinstance(fields, dict):
            raise SchemaException(
                "Node fields must be a key-value dictionary.")
        if not isinstance(links, dict):
            raise SchemaException("Node links must be a key-value dictionary.")
        if src_name is not None:
            if not isinstance(src_name, str):
                raise SchemaException("Source name must be a string.")
            elif re.match(NAME_FORMAT, src_name) is None:
                raise SchemaException(
                    "Source name may contain lowercase letters, numbers and underscores. They must start with a letter.")

        # We have special restrictions for singleton nodes. They must have a
        # single field and no links.
        if is_singleton:
            if len(fields) != 1:
                raise SchemaException(
                    "Singleton nodes must have a single field.")
            if len(links) != 0:
                raise SchemaException("Singleton nodes cannot have links.")

        # Quantity check.
        if len(fields) + len(links) < 1:
            raise SchemaException("Node must have at least one field or link.")

        # Type and name checks for fields.
        for k in fields:
            assert(isinstance(k, str))
            assert(isinstance(fields[k], Field))
            if re.match(NAME_FORMAT, k) is None:
                raise SchemaException(
                    "Node field names may contain lowercase letters, numbers and underscores. "
                    "They must start with a letter.", "fields." + k)

        # Type and name checks for links.
        for k in links:
            assert(isinstance(k, str))
            assert(isinstance(links[k], Link))
            if re.match(NAME_FORMAT, k) is None:
                raise SchemaException(
                    "Node link targets may contain lowercase letters, numbers and underscores. "
                    "They must start with a letter.", "links." + k)

        self.is_singleton = is_singleton
        self.fields = fields
        self.links = links
        self.src_name = src_name

    def _dump(self):

        result = {"singleton": self.is_singleton}

        if self.is_singleton:
            field = next(iter(self.fields.values()))._dump()
            for k, v in field.items():
                result[k] = v

        else:
            result["fields"] = dict([(k, v._dump())
                                     for (k, v) in self.fields.items()])
            result["links"] = dict([(k, v._dump())
                                    for (k, v) in self.links.items()])

        if self.src_name is not None:
            result["src-name"] = self.src_name

        return result

    @staticmethod
    def _load(input):
        is_singleton = input.get("singleton", False)
        links = input.get("links", {})
        fields = input.get("fields", {})
        src_name = input.get("src-name", None)

        if isinstance(fields, dict):
            if is_singleton and len(fields) == 0:
                try:
                    fields["field"] = Field._load(input)
                except SchemaException as se:
                    raise se
            else:
                for k in fields:
                    try:
                        fields[k] = Field._load(fields[k])
                    except SchemaException as se:
                        se.path = "fields." + k
                        raise se

        if isinstance(links, dict):
            for k in links:
                try:
                    links[k] = Link._load(links[k])
                except SchemaException as se:
                    se.path = "links." + k
                    raise se

        return Node(is_singleton, fields, links, src_name)

    def _match(
            self,
            source,
            dim_map,
            class_name_map,
            node_name_map,
            self_classes,
            source_classes,
            build_matching=False):
        assert(isinstance(source, Node))
        assert(isinstance(dim_map, dict))
        assert(isinstance(class_name_map, dict))
        assert(isinstance(node_name_map, dict))

        field_name_map = {}
        dim_map_updates = {}
        class_name_map_updates = {}

        self_tensor_names = [
            name for name in self.fields if isinstance(
                self.fields[name], Tensor)]
        self_category_names = [
            name for name in self.fields if isinstance(
                self.fields[name], Category)]
        source_tensor_names = [
            name for name in source.fields if isinstance(
                source.fields[name], Tensor)]
        source_category_names = [
            name for name in source.fields if isinstance(
                source.fields[name], Category)]

        # Simply dismiss in case the counts don't match.
        if len(self_tensor_names) != len(source_tensor_names) or \
                len(self_category_names) != len(source_category_names) or \
                len(self.links) != len(source.links):
            return (None, {}, {}) if build_matching else (False, {}, {})

        # We first try to match links as this the cheapest operation. We match based on
        # the node matching which must be given when this function is called.
        for (node_name, link) in self.links.items():
            source_node_name = node_name_map[node_name]
            if source_node_name not in source.links or not link._match(
                    source.links[source_node_name]):
                return (None, {}, {}) if build_matching else (False, {}, {})

        # Try all possible tensor matchings. Individual field matchings are not independent
        # because of dimension matchings. Maybe this can be optimised.
        match = len(source_tensor_names) == 0
        for source_tensor_names_perm in itertools.permutations(
                source_tensor_names):
            dim_map_updates_iter = {}
            for (self_name, source_name) in zip(self_tensor_names, source_tensor_names_perm):

                match, dim_map_updates_new = self.fields[self_name]._match(
                    source.fields[source_name],
                    {**dim_map, **dim_map_updates_iter})

                if match:
                    # If there was a match, we update the dimension mappings.
                    dim_map_updates_iter.update(dim_map_updates_new)
                else:
                    # On the first failed match, we skip this permutation.
                    break

            # If we have found a matching, end the search.
            if match:
                # Add matched field names to name map.
                dim_map_updates.update(dim_map_updates_iter)
                field_name_map.update(
                    dict(zip(self_tensor_names, source_tensor_names_perm)))
                break

        # If no matching was found, we don't need to go on further.
        if not match:
            return (None, {}, {}) if build_matching else (False, {}, {})

        # Try all possible category matchings. Individual field matchings are not independent
        # because of dimension matchings. Maybe this can be optimised.
        match = len(source_category_names) == 0
        for source_category_names_perm in itertools.permutations(
                source_category_names):
            dim_map_updates_iter = copy.deepcopy(dim_map_updates)
            class_name_map_updates_iter = {}

            for (self_name, source_name) in zip(self_category_names, source_category_names_perm):

                match, dim_map_updates_new, class_name_map_updates_new = self.fields[self_name]._match(
                    source.fields[source_name],
                    {**dim_map, **dim_map_updates_iter},
                    {**class_name_map, **class_name_map_updates_iter},
                    self_classes,
                    source_classes)

                if match:
                    # If there was a match, we update the dimension and class
                    # name mappings.
                    dim_map_updates_iter.update(dim_map_updates_new)
                    class_name_map_updates_iter.update(
                        class_name_map_updates_new)
                else:
                    # On the first failed match, we skip this permutation.
                    break

                match = True

            # If we have found a matching, end the search.
            if match:
                # Add matched field names to name map.
                dim_map_updates.update(dim_map_updates_iter)
                class_name_map_updates.update(class_name_map_updates_iter)
                field_name_map.update(
                    dict(zip(self_category_names, source_category_names_perm)))
                break

        # If no matching was found, we don't need to go on further.
        if not match:
            return (None, {}, {}) if build_matching else (False, {}, {})

        # If a matching was found, build a resulting matching node if needed.
        if build_matching:
            links = copy.deepcopy(self.links)
            fields = copy.deepcopy(self.fields)
            for field_name, field in fields.items():
                field.src_name = field_name_map[field_name]
                if field.field_type == "tensor":
                    field.src_dim = copy.copy(
                        source.fields[field.src_name].dim)

            return Node(self.is_singleton, fields,
                        links), dim_map_updates, class_name_map_updates

        else:
            # Otherwise, just return a boolean along with name updates.
            return True, dim_map_updates, class_name_map_updates

    def is_variable(self):
        for field in self.fields.values():
            if field.field_type == "tensor" and field.is_variable():
                return True
        return False


class Link:

    def __init__(self, dim):

        # If the link dimension is a list, then we expect two values. One for
        # the upper and one for the lowr bound.
        if isinstance(dim, list):
            if len(dim) != 2:
                raise SchemaException(
                    "Link dimension must be a list of two elements representing the upper and lower bound.")

            if (not isinstance(dim[0], int)) or dim[0] < 0:
                raise SchemaException(
                    "Link lower bound must be a non-negative integer.")

            if (not isinstance(dim[1], int) and dim[1] != 'inf') or (
                    isinstance(dim[1], int) and dim[1] <= 0):
                raise SchemaException(
                    "Link upper bound must be a positive integer or 'inf'.")

            if isinstance(dim[1], int) and dim[0] > dim[1]:
                raise SchemaException(
                    "Link lower bound cannot be greater than the upper bound.")

            self.dim = dim

        # We allow a link dimension to be an integer, in which case we set it
        # as both the upper and lower bound.
        elif isinstance(dim, int):
            if dim < 1:
                raise SchemaException(
                    "Link dimension must be a positive integer.")

            self.dim = [dim, dim]

        else:
            raise SchemaException(
                "Link dimension must be either a positive integer or a two-element list.")

    def _dump(self):
        return self.dim

    @staticmethod
    def _load(input):
        return Link(input)

    def _match(self, source):
        assert(isinstance(source, Link))

        if self.dim[0] > source.dim[0]:
            return False
        else:
            if self.dim[1] == 'inf':
                return True
            elif source.dim[1] == 'inf' or self.dim[1] < source.dim[1]:
                return False

        return True


class Field:

    def __init__(self, field_type, src_name=None):
        assert(field_type in ["tensor", "category"])
        assert(src_name is None or isinstance(src_name, str))

        self.field_type = field_type
        self.src_name = src_name

    def _dump(self):
        result = {"type": self.field_type}
        if self.src_name is not None:
            result["src-name"] = self.src_name
        return result

    @staticmethod
    def _load(input):

        try:
            field_type = input["type"]
        except KeyError:
            raise SchemaException("Field must have a 'type' field.")

        if field_type == "tensor":
            return Tensor._load(input)
        elif field_type == "category":
            return Category._load(input)
        else:
            raise SchemaException("Unknown field type '%s'." % field_type)


class Tensor(Field):

    def __init__(self, dim, src_name=None, src_dim=None):

        # Simple type and value checks.
        if not isinstance(dim, list):
            raise SchemaException(
                "Tensor dim field must be a list of dimension definitions.")
        if len(dim) < 1:
            raise SchemaException("Tensor must have at least one dimension.")
        if src_name is not None:
            if not isinstance(src_name, str):
                raise SchemaException("Source name must be a string.")
            elif re.match(NAME_FORMAT, src_name) is None:
                raise SchemaException(
                    "Source name may contain lowercase letters, numbers and underscores. They must start with a letter.")

        # Type and value checks for each dimension.
        for d in dim:
            if not (isinstance(d, int) or isinstance(d, str)):
                raise SchemaException(
                    "Tensor dim fields must all be integers or strings.")
            elif isinstance(d, int) and d < 1:
                raise SchemaException(
                    "Tensor dim fields that are integer must be positive numbers.")
            elif isinstance(d, str) and re.match(DIM_FORMAT, d) is None:
                raise SchemaException(
                    "Tensor dim fields that are strings may contain only lowercase "
                    "letters, numbers and underscores. They must start with a letter. "
                    "They may be suffixed by wildcard characters '?', '+' and '*' to denote variable count dimensions.")

        # Type and value checks for each source dimension.
        if src_dim is not None:
            if not isinstance(src_dim, list):
                raise SchemaException(
                    "Tensor source dim field must be a list of dimension definitions.")
            for d in src_dim:
                if not (isinstance(d, int) or isinstance(d, str)):
                    raise SchemaException(
                        "Tensor source dim fields must all be integers or strings.")
                elif isinstance(d, int) and d < 1:
                    raise SchemaException(
                        "Tensor source dim fields that are integer must be positive numbers.")
                elif isinstance(d, str) and re.match(DIM_FORMAT, d) is None:
                    raise SchemaException(
                        "Tensor source dim fields that are strings may contain only lowercase "
                        "letters, numbers and underscores. They must start with a letter. "
                        "They may be suffixed by wildcard characters '?', '+' and '*' to denote variable count dimensions.")

        # Make sure that we have at most one variable count dimension.
        if sum([isinstance(d, str) and d[-1] in ["?", "+", "*"]
                for d in dim]) > 1:
            raise SchemaException(
                "Tensor can have at most one variable count dimension.")

        # If we have only one dimension, make sure it's not a variable count dimension which
        # allows zero dimensions.
        if len(dim) == 1 and isinstance(
                dim[0], str) and dim[0][-1] in ["?", "*"]:
            raise SchemaException(
                "Tensors cannot have zero dimensions. "
                "Having only one dimension suffixed with '?' or '*' permits this.")

        super(Tensor, self).__init__("tensor", src_name)
        self.dim = dim
        self.src_dim = src_dim

    def _dump(self):
        result = super(Tensor, self)._dump()
        result["dim"] = self.dim
        if self.src_dim is not None:
            result["src-dim"] = self.src_dim
        return result

    @staticmethod
    def _load(input):
        dim = input.get("dim", None)
        src_name = input.get("src-name", None)
        src_dim = input.get("src-dim", None)

        if dim is None:
            raise SchemaException("Tensor must have a 'dim' field.")

        return Tensor(dim, src_name, src_dim)

    def _match(self, source, dim_map):
        assert(isinstance(dim_map, dict))
        assert(isinstance(source, Tensor))

        match, dim_map_update = match_dim_list(self.dim, source.dim, dim_map)
        return match, dim_map_update

    def is_variable(self):
        for dim in self.dim:
            if isinstance(dim, str):
                return True
        return False


def match_dim_list(list_a, list_b, dim_map=None):
    if dim_map is None:
        dim_map = {}

    # Get first dimension and modifier of list A if possible.
    dim_a, mod_a = None, None
    if len(list_a) > 0:
        dim_a = list_a[0]
        if isinstance(dim_a, str) and dim_a[-1] in ['?', '+', '*']:
            mod_a = dim_a[-1]
            dim_a = dim_a[:-1]

    # Get first dimension and modifier of list B if possible.
    dim_b, mod_b = None, None
    if len(list_b) > 0:
        dim_b = list_b[0]
        if isinstance(dim_b, str) and dim_b[-1] in ['?', '+', '*']:
            mod_b = dim_b[-1]
            dim_b = dim_b[:-1]

    # If both lists are empty we simply return True.
    if dim_a is None and dim_b is None:
        return True, {}

    # Handle the case when only list A is empty.
    if dim_a is None:
        # If list A is empty, we can continue only if modifier B can be
        # skipped.
        if mod_b in ['?', '*']:
            return match_dim_list(list_a, list_b[1:], dim_map)
        else:
            return False, {}

    # Handle the case when only list B is empty.
    if dim_b is None:
        # If list B is empty, we can continue only if modifier A can be
        # skipped.
        if mod_a in ['?', '*']:
            return match_dim_list(list_a[1:], list_b, dim_map)
        else:
            return False, {}

    # Check whether we can match the first dimensions.
    match = isinstance(
        dim_a, str) and (
        dim_a not in dim_map or dim_map[dim_a] == dim_b) or isinstance(
            dim_a, int) and dim_a == dim_b

    # If we can match we can try to move on.
    if match:
        dim_map_update = {} if isinstance(dim_a, int) else {dim_a: dim_b}
        rec_match, rec_dim_map_update = match_dim_list(
            list_a[1:], list_b[1:], {**dim_map, **dim_map_update})
        if rec_match:
            return True, {**dim_map, **dim_map_update, **rec_dim_map_update}

    # We can match and move dim A to match current B with more dims.
    if match and mod_b in ['+', '*']:
        dim_map_update = {} if isinstance(dim_a, int) else {dim_a: dim_b}
        rec_match, rec_dim_map_update = match_dim_list(
            list_a[1:], list_b, {**dim_map, **dim_map_update})
        if rec_match:
            return True, {**dim_map, **dim_map_update, **rec_dim_map_update}

    # We can skip dim A if it is skippable.
    if mod_a in ['?', '*']:
        rec_match, rec_dim_map_update = match_dim_list(
            list_a[1:], list_b, dim_map)
        if rec_match:
            return True, {**dim_map, **rec_dim_map_update}

    # We can match and move dim B to match current A with more dims.
    if match and mod_a in ['+', '*']:
        dim_map_update = {} if isinstance(dim_a, int) else {dim_a: dim_b}
        rec_match, rec_dim_map_update = match_dim_list(
            list_a, list_b[1:], {**dim_map, **dim_map_update})
        if rec_match:
            return True, {**dim_map, **dim_map_update, **rec_dim_map_update}

    # We can skip dim B if it is skippable.
    if mod_b in ['?', '*']:
        rec_match, rec_dim_map_update = match_dim_list(
            list_a, list_b[1:], dim_map)
        if rec_match:
            return True, {**dim_map, **rec_dim_map_update}

    return False, {}


class Category(Field):

    def __init__(self, category_class, src_name=None):

        # Simple type and value checks.
        if not isinstance(category_class, str):
            raise SchemaException("Category class must be a string.")
        elif re.match(NAME_FORMAT, category_class) is None:
            raise SchemaException(
                "Category class may contain lowercase letters, numbers and underscores. They must start with a letter.")
        if src_name is not None:
            if not isinstance(src_name, str):
                raise SchemaException("Source name must be a string.")
            elif re.match(NAME_FORMAT, src_name) is None:
                raise SchemaException(
                    "Source name may contain lowercase letters, numbers and underscores. They must start with a letter.")

        super(Category, self).__init__("category", src_name)
        self.category_class = category_class

    def _dump(self):
        result = super(Category, self)._dump()
        result["class"] = self.category_class
        return result

    @staticmethod
    def _load(input):
        category_class = input.get("class", None)
        src_name = input.get("src-name", None)

        if category_class is None:
            raise SchemaException("Category must have a 'class' field.")

        return Category(category_class, src_name)

    def _match(
            self,
            source,
            dim_map,
            class_name_map,
            self_classes,
            source_classes):
        assert(isinstance(dim_map, dict))
        assert(isinstance(class_name_map, dict))
        assert(isinstance(source, Category))

        # If the class has already been mapped then we simply compare.
        if self.category_class in class_name_map:
            return source.category_class == class_name_map[self.category_class], {
            }, {}

        else:
            self_class = self_classes[self.category_class]
            source_class = source_classes[source.category_class]

            match, dim_map_update = self_class._match(source_class, dim_map)
            if match:
                return True, dim_map_update, {
                    self.category_class: source.category_class}
            else:
                return False, {}, {}


class Class:

    def __init__(self, dim, src_name=None):

        # Simple type and value checks.
        if not isinstance(dim, int) and not isinstance(dim, str):
            raise SchemaException(
                "Class dimension must be an integer or a string.")
        if isinstance(dim, int) and dim < 1:
            raise SchemaException(
                "Class dimension must be a positive integer.")
        if isinstance(dim, str) and re.match(NAME_FORMAT, dim) is None:
            raise SchemaException(
                "Class dimension can contain lowercase letters, numbers and underscores. They must start with a letter.")
        if src_name is not None:
            if not isinstance(src_name, str):
                raise SchemaException("Source name must be a string.")
            elif re.match(NAME_FORMAT, src_name) is None:
                raise SchemaException(
                    "Source name may contain lowercase letters, numbers and underscores. They must start with a letter.")

        self.dim = dim
        self.src_name = src_name

    def _dump(self):
        result = {"dim": self.dim}
        if self.src_name is not None:
            result["src-name"] = self.src_name
        return result

    @staticmethod
    def _load(input):
        assert(isinstance(input, dict))

        dim = input.get("dim", None)
        src_name = input.get("src-name", None)

        if dim is None:
            raise SchemaException("Class must have a 'dim' field.")

        return Class(dim, src_name)

    def _match(self, source, dim_map):
        assert(isinstance(dim_map, dict))
        assert(isinstance(source, Class))

        if isinstance(self.dim, int):
            return self.dim == source.dim, {}
        else:
            dim = dim_map.get(self.dim, None)
            if dim is None:
                return True, {self.dim: source.dim}
            else:
                return dim == source.dim, {}

    def is_variable(self):
        return isinstance(self.dim, str)


def validate(args):  # pragma: no cover
    with open(args.src) as f:
        src = json.load(f)

    try:
        Schema.load(src)
    except SchemaException as se:
        print("Source schema validation error:")
        print("  Path:      ", se.path)
        print("  Message:   ", se.message)
        sys.exit(-1)

    sys.exit(0)


def match(args):  # pragma: no cover

    with open(args.src) as f:
        src = json.load(f)

    try:
        src_schema = Schema.load(src)
    except SchemaException as se:
        print("Source schema validation error:")
        print("  Path:      ", se.path)
        print("  Message:   ", se.message)
        sys.exit(-1)

    with open(args.dst) as f:
        dst = json.load(f)

    try:
        dst_schema = Schema.load(dst)
    except SchemaException as se:
        print("Destination schema validation error:")
        print("  Path:      ", se.path)
        print("  Message:   ", se.message)
        sys.exit(-1)

    match = dst_schema.match(src_schema, build_matching=True)

    if match is not None:
        print(json.dumps(match.dump(), indent=2))
        sys.exit(0)
    else:
        print("Schema match failed.")
        sys.exit(-1)


if __name__ == "__main__":  # pragma: no cover

    description = "Operations with ease.ml schemas."

    parser = argparse.ArgumentParser(description=description)
    subparsers = parser.add_subparsers(
        description="Please choose one of the following commands to run.")

    validate_parser = subparsers.add_parser(
        "validate", help="Check if the given schema is valid.")
    validate_parser.add_argument(
        "src",
        type=str,
        help="JSON file containing the schema description.")
    validate_parser.set_defaults(func=validate)

    match_parser = subparsers.add_parser(
        "match",
        help="Check if the given source schema can be accepted by a given destination schema.")
    match_parser.add_argument(
        "src",
        type=str,
        help="JSON file containing the source schema description.")
    match_parser.add_argument(
        "dst",
        type=str,
        help="JSON file containing the destination schema description.")
    match_parser.set_defaults(func=match)

    args = parser.parse_args()
    if hasattr(args, "func"):
        args.func(args)
    else:
        parser.print_help()
