
import argparse
import numpy as np
import json
import os
import random
import re
import string
import sys

import easemlschema.schema as sch


class DatasetException(Exception):

    def __init__(self, message, path=""):
        self.message = message
        self.path = path


def default_opener(root, rel_path, file_type, read_only=True, binary=False):

    path = os.path.join(root, rel_path)

    if file_type == "directory":

        if not read_only:
            if not os.path.exists(path):
                os.makedirs(path)

        return os.listdir(path)

    else:

        mode = "r" if read_only else "w"
        mode += "b" if binary else ""

        return open(path, mode)


def random_string(size, chars=string.ascii_lowercase + string.digits):
    return "".join(random.choice(chars) for _ in range(size))


class File:

    def __init__(self, name, file_type, subtype="default"):
        assert isinstance(name, str)
        assert file_type in FILE_TYPES

        self.name = name
        self.file_type = file_type
        self.subtype = "default"


class Directory(File):

    def __init__(self, name, children=None):
        if children is None:
            children = {}
        super(Directory, self).__init__(name, "directory")
        self.children = children

    @staticmethod
    def _load(root, rel_path, name, opener,
              metadata_only=False, subtype="default"):
        path = os.path.join(rel_path, name) if len(name) > 0 else rel_path
        children = Directory._load_children(root, path, opener, metadata_only)
        return Directory(name, children)

    @staticmethod
    def _load_children(root, path, opener, metadata_only=False):

        dirlist = opener(root, path, "directory")

        children = {}

        for n in dirlist:
            child = None

            for file_type, ext_dict in TYPE_EXTENSIONS.items():
                for subtype, ext in ext_dict.items():
                    if n.endswith(ext):
                        n = n[:-len(ext)]
                        child = FILE_TYPES[file_type]._load(
                            root, path, n, opener, metadata_only, subtype)
                        break

            if child is None:
                child = Directory._load(
                    root, path, n, opener, metadata_only, "default")

            children[n] = child

        return children

    def _dump(self, root, rel_path, name, opener):
        path = os.path.join(rel_path, name)
        opener(root, path, "directory", read_only=False)
        self._dump_children(root, path, opener)

    def _dump_children(self, root, path, opener):
        for child_name, child in self.children.items():
            child._dump(root, path, child_name, opener)

    def get_child_by_type(self, file_type):
        for child in self.children.values():
            if child.file_type == file_type:
                return child
            if child.file_type == "directory":
                result = child.get_child_by_type(file_type)
                if result is not None:
                    return result
        return None

    def get_file_subtype(self, file_type):
        child = self.get_child_by_type(file_type)
        if child is not None:
            return child.subtype
        else:
            return None

    def set_file_subtype(self, file_type, subtype):
        for child in self.children.values():
            if child.file_type == file_type:
                child.subtype = subtype
            if child.file_type == "directory":
                child.set_file_subtype(file_type, subtype)


class Dataset(Directory):

    def __init__(self, root, children):
        assert(isinstance(root, str))
        self.root = root
        super(Dataset, self).__init__(root, children)

    def infer_schema(self):

        category_classes = {}
        category_class_sets = {}
        sch_category_classes = {}
        samples = {}
        for child_name, child in self.children.items():

            if child.file_type == "directory":
                # A directory corresponds to a data sample.
                samples[child_name] = child
            elif child.file_type == "class":
                # Collect class file and get its dimensions.
                category_classes[child_name] = child
                sch_category_classes[child_name] = sch.Class(
                    len(child.categories))
                category_class_sets[child_name] = set(child.categories)
            else:
                # We forbid any other file type in the root directory. Maybe we
                # should just ignore.
                raise DatasetException("Files of type '%s' are unexpected in dataset root." %
                                       child.file_type, "/".join(["", child_name]))

        sch_nodes = {}
        first_sample = True
        links_file_found = False
        sch_cyclic = False
        sch_fanin = False
        sch_undirected = True

        # Go through all data samples.
        for sample_name, sample in samples.items():

            # Sort all sample children according to their type.
            sample_children = dict([(k, {}) for k in FILE_TYPES])
            sample_nodes = set()
            for child_name, child in sample.children.items():
                assert(child.file_type in FILE_TYPES)

                if child.file_type == "links":
                    links_file_found = True
                else:
                    sample_nodes.add(child_name)

                sample_children[child.file_type][child_name] = child

            # Ensure that either all samples have links files or none of them
            # have.
            if (len(sample_children["links"]) > 0) != links_file_found:
                raise DatasetException(
                    "Links file not found in all data samples.", "/".join(["", sample_name]))

            # Ensure all samples have the same node names.
            schema_nodes_set = set(sch_nodes.keys())
            if not first_sample and schema_nodes_set != sample_nodes:
                if schema_nodes_set.issuperset(sample_nodes):
                    child_name = next(iter(schema_nodes_set - sample_nodes))
                    raise DatasetException(
                        "Item expected but not found.", "/".join(["", sample_name, child_name]))
                elif sample_nodes.issuperset(schema_nodes_set):
                    child_name = next(iter(sample_nodes - schema_nodes_set))
                    raise DatasetException(
                        "Item found but not expected.", "/".join(["", sample_name, child_name]))

            # Handle tensor singleton nodes.
            for child_name, child in sample_children["tensor"].items():
                if first_sample:
                    field = sch.Tensor(child.dimensions)
                    sch_nodes[child_name] = sch.Node(
                        is_singleton=True, fields={"field": field})
                else:
                    # Verify that the node is the same.
                    node = sch_nodes[child_name]
                    if node.is_singleton is False or len(
                            node.fields) > 1 or node.fields["field"].field_type != "tensor":
                        raise DatasetException(
                            "Node '%s' not the same type in all samples." % child_name, "/".join(["", sample_name]))
                    elif node.fields["field"].dim != child.dimensions:
                        raise DatasetException(
                            "Tensor dimensions mismatch.", "/".join(["", sample_name, child_name]))

            # Handle category singleton nodes.
            for child_name, child in sample_children["category"].items():

                # Infer class by finding first class to which the node belongs.
                category_class = None
                for class_name, category_set in category_class_sets.items():
                    if child.belongs_to_set(category_set):
                        category_class = class_name
                        break
                if category_class is None:
                    raise DatasetException(
                        "Category file does not match any class.", "/".join(["", sample_name, child_name]))

                if first_sample:
                    field = sch.Category(category_class)
                    sch_nodes[child_name] = sch.Node(
                        is_singleton=True, fields={"field": field})
                else:
                    # Verify that the node is the same.
                    node = sch_nodes[child_name]
                    if node.is_singleton is False or len(
                            node.fields) > 1 or node.fields["field"].field_type != "category":
                        raise DatasetException(
                            "Node '%s' not the same type in all samples." % child_name, "/".join(["", sample_name]))
                    elif node.fields["field"].category_class != category_class:
                        raise DatasetException(
                            "Category class mismatch.", "/".join(["", sample_name, child_name]))

            # This counts how many instances each node has. It is used to
            # validate link targets.
            node_instance_count = {}

            # Handle regular non-singleton nodes.
            for child_name, child in sample_children["directory"].items():
                if first_sample:
                    fields = {}
                else:
                    node = sch_nodes[child_name]
                    fields = node.fields
                    if node.is_singleton:
                        raise DatasetException(
                            "Node '%s' not the same type in all samples." % child_name, "/".join(["", sample_name]))

                # Ensure nodes in all samples have the same children.
                fields_set, children_set = set(
                    fields.keys()), set(child.children.keys())
                if not first_sample and fields_set != children_set:
                    if fields_set.issuperset(children_set):
                        node_child_name = next(iter(fields_set - children_set))
                        raise DatasetException(
                            "Item expected but not found.", "/".join(["", sample_name, child_name, node_child_name]))
                    elif children_set.issuperset(fields_set):
                        node_child_name = next(iter(children_set - fields_set))
                        raise DatasetException(
                            "Item found but not expected.", "/".join(["", sample_name, child_name, node_child_name]))

                # Go through all fields of the non-singleton node.
                for node_child_name, node_child in child.children.items():

                    if node_child.file_type == "tensor":

                        # Verify that all node fields have the same number of
                        # instances.
                        count = node_child.dimensions[0]
                        if node_instance_count.setdefault(
                                child_name, count) != count:
                            raise DatasetException("Tensor instance count mismatch.",
                                                   "/".join(["", sample_name, child_name, node_child_name]))

                        if first_sample:
                            fields[node_child_name] = sch.Tensor(
                                node_child.dimensions[1:])
                        else:
                            # Verify that the node is the same.
                            field = fields[node_child_name]
                            if field.field_type != "tensor":
                                raise DatasetException("Node '%s' not the same type in all samples." %
                                                       child_name, "/".join(["", sample_name, child_name, node_child_name]))
                            if field.dim != node_child.dimensions[1:]:
                                raise DatasetException("Tensor dimensions mismatch.",
                                                       "/".join(["", sample_name, child_name, node_child_name]))

                    elif node_child.file_type == "category":

                        # Infer class by finding first class to which the node
                        # belongs.
                        category_class = None
                        for class_name, category_set in category_class_sets.items():
                            if node_child.belongs_to_set(category_set):
                                category_class = class_name
                                break
                        if category_class is None:
                            raise DatasetException("Category file does not match any class.", "/".join(
                                ["", sample_name, child_name, node_child_name]))

                        # Verify that all node fields have the same number of
                        # instances.
                        count = len(node_child.categories)
                        if node_instance_count.setdefault(
                                child_name, count) != count:
                            raise DatasetException("Category instance count mismatch.",
                                                   "/".join(["", sample_name, child_name, node_child_name]))

                        if first_sample:
                            fields[node_child_name] = sch.Category(
                                category_class)
                        else:
                            # Verify that the node is the same.
                            field = fields[node_child_name]
                            if field.field_type != "category":
                                raise DatasetException("Node '%s' not the same type in all samples." %
                                                       child_name, "/".join(["", sample_name, child_name, node_child_name]))
                            if field.category_class != category_class:
                                raise DatasetException("Category class mismatch.",
                                                       "/".join(["", sample_name, child_name, node_child_name]))

                    else:
                        # We forbid any other file type in the node directory.
                        # Maybe we should just ignore.
                        raise DatasetException("Files of type '%s' are unexpected in node directory." %
                                               node_child.file_type, "/".join(["", sample_name, child_name, node_child_name]))

                # Create the node with all found fields. We will add links
                # later.
                sch_nodes[child_name] = sch.Node(
                    is_singleton=False, fields=fields)

            # Handle links files. We allow at most one links file.
            if len(sample_children["links"]) > 1:
                raise DatasetException(
                    "At most one links file per data sample is allowed.", "/".join(["", sample_name]))

            # If a links file is missing, we might have implicit links.
            if len(sample_children["links"]) == 0:

                # If we have non-signleton nodes, then we assume a single directed chain.
                # To construct a graph without links, there must be an empty
                # links file.
                for node_name, node in sch_nodes.items():
                    if not node.is_singleton:
                        node.links[node_name] = sch.Link(1)
                        sch_undirected = False

            else:

                links = list(sample_children["links"].values())[0]

                if len(node_instance_count) == 0:
                    raise DatasetException(
                        "Link file found but no non-singleton nodes.", "/".join(["", sample_name]))

                # Check link counts.
                link_counts = links.get_link_destination_counts()
                for ((src_node_name, _, dst_node_name),
                     count) in link_counts.items():
                    src_node = sch_nodes.get(src_node_name, None)
                    if src_node is None:
                        raise DatasetException(
                            "Link references unknown node '%s'." % src_node_name, "/".join(["", sample_name]))
                    if src_node.is_singleton:
                        raise DatasetException(
                            "Link references singleton node '%s'." % src_node_name, "/".join(["", sample_name]))

                    dst_node = sch_nodes.get(dst_node_name, None)
                    if dst_node is None:
                        raise DatasetException(
                            "Link references unknown node '%s'." % dst_node_name, "/".join(["", sample_name]))
                    if dst_node.is_singleton:
                        raise DatasetException(
                            "Link references singleton node '%s'." % dst_node_name, "/".join(["", sample_name]))

                    # All link counts are merged together over the entire
                    # sample.
                    link = src_node.links.setdefault(
                        dst_node_name, sch.Link(count))
                    link.dim[0] = min(count, link.dim[0])
                    link.dim[1] = max(count, link.dim[1])

                # Check if any link index overflows the number of node
                # instances.
                max_node_indices = links.get_max_indices_per_node()
                for node_name, count in max_node_indices.items():
                    if count >= node_instance_count[node_name]:
                        raise DatasetException("Found link index %d to node with %d instances." % (
                            count, node_instance_count[node_name]), "/".join(["", sample_name]))

                # Get referential constraints if needed.
                if sch_undirected:
                    # If the undirected constraint has been violated
                    # at least once, there is no point to check more.
                    sch_undirected = links.is_undirected()
                if not sch_fanin:
                    # If a fan-in has been detected at any point, there is no
                    # point to check more.
                    sch_fanin = links.is_fanin(sch_undirected)
                if not sch_cyclic:
                    # If cycles have been detected at any point, the links are
                    # not acyclic.
                    sch_cyclic = links.is_cyclic(sch_undirected)

            first_sample = False

        # Build and return schema result.
        return sch.Schema(sch_nodes, sch_category_classes,
                          sch_cyclic, sch_undirected, sch_fanin)

    @staticmethod
    def generate_from_schema(
            root, schema, num_samples=10, num_node_instances=10):
        assert(schema.is_variable() is False)

        # Generate classes.
        classes = {}
        for class_name, category_class in schema.category_classes.items():
            assert(isinstance(category_class.dim, int))
            categories = [random_string(16) for _ in range(category_class.dim)]
            classes[class_name] = Class(class_name, categories)

        # Generate samples.
        samples = {}
        for _ in range(num_samples):

            sample_name = random_string(16)
            nodes = {}

            # Generate nodes.
            for node_name, node in schema.nodes.items():

                if node.is_singleton:

                    field = next(iter(node.fields.values()))
                    assert(field.field_type in ["tensor", "category"])

                    # Generate singleton tensor.
                    if field.field_type == "tensor":
                        assert(all([isinstance(x, int) for x in field.dim]))
                        data = np.random.rand(*field.dim)
                        nodes[node_name] = Tensor(node_name, field.dim, data)

                    # Generate singleton category.
                    elif field.field_type == "category":
                        categories = [random.choice(
                            classes[field.category_class].categories)]
                        nodes[node_name] = Category(node_name, categories)

                else:

                    node_children = {}

                    for field_name, field in node.fields.items():
                        assert(field.field_type in ["tensor", "category"])

                        # Generate non-singleton tensor.
                        if field.field_type == "tensor":
                            assert(all([isinstance(x, int)
                                        for x in field.dim]))
                            dim = [num_node_instances] + field.dim
                            data = np.random.rand(*dim)
                            node_children[field_name] = Tensor(
                                field_name, dim, data)

                        # Generate non-singleton category.
                        elif field.field_type == "category":
                            categories = [random.choice(
                                classes[field.category_class].categories) for _ in range(num_node_instances)]
                            node_children[field_name] = Category(
                                field_name, categories)

                    # Generate the actual node directory.
                    nodes[node_name] = Directory(node_name, node_children)

            # Generate links.
            links = set()
            all_instances = {}
            count_in, count_out = {}, {}
            for node_name, node in schema.nodes.items():
                if not node.is_singleton:
                    all_instances[node_name] = random.sample(
                        range(num_node_instances), k=num_node_instances)
                    for i in range(num_node_instances):
                        count_in[(node_name, i)] = 0
                        count_out[(node_name, i)] = 0
            max_idx_in = {}

            # TODO: Fix this. Take advantage of SOURCE and SINK.
            for node, instances in all_instances.items():
                for i in range(len(instances)):
                    for target, link in schema.nodes[node].links.items():
                        l_bound = link.dim[0]
                        u_bound = min(
                            link.dim[1],
                            num_node_instances) if link.dim[1] != "inf" else num_node_instances
                        assert(l_bound <= u_bound)
                        count = random.randrange(
                            l_bound, u_bound + 1) - count_out[(node, i)]

                        # If there are no links to create, simply skip.
                        if count <= 0:
                            continue

                        candidates = list(range(len(all_instances[target])))
                        if not schema.cyclic:
                            if schema.undirected:
                                candidates = [
                                    x for x in candidates if x != i and count_in[(target, x)] == 0]
                            elif target == node:
                                candidates = [x for x in candidates if x > i]
                            else:
                                idx = max_idx_in.get((node, i, target), -1)
                                candidates = [x for x in candidates if x > idx]

                        if not schema.fanin:
                            max_count = 2 if schema.undirected else 1
                            candidates = [
                                x for x in candidates if count_in[(target, x)] < max_count]

                        # assert(len(candidates) >= count)
                        for j in candidates[:count]:
                            count_out[(node, i)] += 1
                            count_in[(target, j)] += 1
                            links.add(Link(node, i, target, j))

                            idx = max_idx_in.get((node, i, target), 0)
                            max_idx_in[(node, i, target)] = max(idx, j)

                            if schema.undirected:
                                count_out[(target, j)] += 1
                                count_in[(node, i)] += 1
                                links.add(Link(target, j, node, i))

                                idx = max_idx_in.get((target, j, node), 0)
                                max_idx_in[(target, j, node)] = max(idx, i)

            # Conect nodes without an incoming link to the SOURCE and
            # without an outgoing one to the SINK.
            # for ((node_name, i), count) in count_in.items():
            #     if count == 0:
            #         links.add(Link(NODE_SOURCE, None, node_name, i))
            # for ((node_name, i), count) in count_out.items():
            #     if count == 0:
            #         links.add(Link(node_name, i, NODE_SINK, None))

            # Create a links instance if there were non-singleton nodes.
            if len(all_instances) > 0:
                nodes["links"] = Links("links", links)

            # Generate the actual sample directory.
            samples[sample_name] = Directory(sample_name, nodes)

        # Generate the dataset.
        dataset = Dataset(root, {**samples, **classes})

        return dataset

    @staticmethod
    def load(root, metadata_only=False, opener=None):
        if opener is None:
            opener = default_opener
        # TODO: This won't work. The directory name cannot be "root" as it will
        # be appended to the path.
        children = Directory._load_children(root, "", opener, metadata_only)
        return Dataset(root, children)

    def dump(self, root, opener=None):
        if opener is None:
            opener = default_opener
        # TODO: This won't work. The directory name cannot be "root" as it will
        # be appended to the path.
        self._dump(root, "", "", opener)


class Tensor(File):

    def __init__(self, name, dimensions, data=None, subtype="default"):
        assert(subtype in ["default", "csv"])
        super(Tensor, self).__init__(name, "tensor", subtype)
        assert(isinstance(dimensions, list) or isinstance(dimensions, tuple))
        self.dimensions = list(dimensions)
        self.data = data

    @staticmethod
    def _load(root, rel_path, name, opener,
              metadata_only=False, subtype="default"):
        path = os.path.join(rel_path, name)
        with opener(root, path + TYPE_EXTENSIONS["tensor"][subtype], "tensor", read_only=True, binary=True) as f:

            if subtype == "default":
                if metadata_only:
                    major, minor = np.lib.format.read_magic(f)
                    read_header = getattr(
                        np.lib.format, "read_array_header_%d_%d" %
                        (major, minor))
                    shape, _, dtype = read_header(f)
                    data = None

                else:
                    data = np.load(f)
                    shape = data.shape
                    dtype = data.dtype

            elif subtype == "csv":
                data = np.loadtxt(f, delimiter=",")
                shape = data.shape
                dtype = data.dtype

        if dtype != np.dtype("float_"):
            raise DatasetException("Tensor datatype must be float64.", path)

        return Tensor(name, shape, data, subtype)

    def _dump(self, root, rel_path, name, opener):

        path = os.path.join(rel_path, name)
        if self.data is None:
            raise DatasetException("Cannot write tensor without data.", path)

        with opener(root, path + TYPE_EXTENSIONS["tensor"][self.subtype], "tensor", read_only=False, binary=True) as f:
            if self.subtype == "default":
                np.save(f, self.data, allow_pickle=False)
            elif self.subtype == "csv":
                np.savetxt(f, self.data, delimiter=",")


class Category(File):

    def __init__(self, name, categories):
        super(Category, self).__init__(name, "category")
        assert(isinstance(categories, list))
        self.categories = categories

    @staticmethod
    def _load(root, rel_path, name, opener,
              metadata_only=False, subtype="default"):
        path = os.path.join(rel_path, name)
        with opener(root, path + TYPE_EXTENSIONS["category"][subtype], "category", read_only=True, binary=False) as f:
            lines = [x.strip() for x in f.readlines()]

        return Category(name, lines)

    def _dump(self, root, rel_path, name, opener):
        path = os.path.join(rel_path, name)
        with opener(root, path + TYPE_EXTENSIONS["category"][self.subtype], "category", read_only=False, binary=False) as f:
            lines = [x + "\n" for x in self.categories]
            f.writelines(lines)

    def get_orphans(self, category_class):
        assert(isinstance(category_class, Class))
        return set(self.categories) - set(category_class.categories)

    def belongs_to_set(self, category_set):
        assert(isinstance(category_set, set))
        for category in self.categories:
            if category not in category_set:
                return False
        return True


class Links(File):

    def __init__(self, name, links):
        super(Links, self).__init__(name, "links")
        assert(isinstance(links, set))
        assert(all([isinstance(link, Link) for link in links]))
        self.links = links

    @staticmethod
    def _load(root, rel_path, name, opener,
              metadata_only=False, subtype="default"):
        path = os.path.join(rel_path, name)
        with opener(root, path + TYPE_EXTENSIONS["links"][subtype], "links", read_only=True, binary=False) as f:
            lines = [x.strip() for x in f.readlines()]

        try:
            links = set([Link.load(x) for x in lines if len(x) > 0])
        except DatasetException as de:
            de.path = path
            raise de

        return Links(name, links)

    def _dump(self, root, rel_path, name, opener):
        path = os.path.join(rel_path, name)
        with opener(root, path + TYPE_EXTENSIONS["links"][self.subtype], "links", read_only=False, binary=False) as f:
            lines = [x.dump() + "\n" for x in self.links]
            f.writelines(lines)

    def adjacency_map(self, nodes):
        assert(isinstance(nodes, set))
        assert(all([isinstance(node, tuple) and len(node) == 2 and isinstance(
            node[0], str) and (node[1] is None) for node in nodes]))

        # Build the adjacency map for the given set of nodes.
        adjacency = dict([(node, []) for node in nodes])
        for link in self.links:
            adjacency[link.src()].append(link.dst())
            nodes.discard(link.dst())

        # Make sure that the implicit source node points to all nodes that have no incoming links
        # and that all nodes without an outgoing link point to the sink.
        for node, adj in adjacency.items():
            if node != NODE_SOURCE and len(adj) == 0:
                adj.append(NODE_SINK)
        adjacency[NODE_SOURCE] = list(nodes)

        return adjacency

    def is_fanin(self, undirected):
        dst_nodes = {}
        for link in self.links:
            # If a node has more than one incoming link, this constitutes a fan-in in a directed graph.
            # In an undirected graph, a fan-in happens when a node is connected
            # to more than 2 nodes.
            count = dst_nodes.get((link.dst_node, link.dst_index), 0)
            if count >= (2 if undirected else 1):
                return True
            dst_nodes[(link.dst_node, link.dst_index)] = count + 1
        return False

    def is_undirected(self):
        for link in self.links:
            # If for every link from A to B, we don't find
            # a link from B to A, then the graph is not undirected.
            if link.get_reverse() not in self.links:
                return False
        return True

    def is_cyclic(self, undirected):
        assert(isinstance(undirected, bool))

        # This is a set of unvisited nodes.
        nodes = set()

        # Build adjacency list for each node.
        adjacency = {}
        for link in self.links:
            nodes.add(link.src())
            nodes.add(link.dst())
            adjacency.setdefault(link.src(), []).append(link.dst())

        # The algorithm differs for undirected and directed graphs.
        if undirected:
            while len(nodes) > 0:

                # Get arbitrary unvisited node.
                x = nodes.pop()

                # For undirected graphs, we need to remember the parent x.
                adj = [(x, y) for y in adjacency[x]]
                stack = adj

                while len(stack) > 0:

                    # Get the node and its parent.
                    parent, x = stack.pop()

                    # A node is missing from nodes only if it is visited.
                    if x not in nodes:
                        return True
                    else:
                        nodes.remove(x)

                    # In undirected graphs, all edges are bidirectional. We
                    # don't count this as a cycle.
                    adj = [(x, y) for y in adjacency[x] if y != parent]
                    stack.extend(adj)

        else:
            while len(nodes) > 0:

                # We keep a set of nodes that are ancestors in the DFS tree.
                ancestors = set()

                # Get arbitrary unvisited node and push it to the stack.
                x = next(iter(nodes))
                stack = [x]

                while len(stack) > 0:

                    # We encounter each node twice.
                    x = stack[-1]

                    # If it is not in ancestors, then this is a first
                    # encounter.
                    if x not in ancestors:

                        # Mark node as visited and add it to active ancestors.
                        nodes.discard(x)
                        ancestors.add(x)
                        adj = adjacency.get(x, [])

                        # A cycle is detected if we find a back edge (i.e. edge
                        # pointing to an ancestor).
                        for y in adj:
                            if y in ancestors:
                                return True

                        # Add all adjacent nodes to the stack.
                        stack.extend([a for a in adj if a in nodes])

                    else:

                        # Since we encountered x the second time, we can pop it
                        # from the stack and remove it from active ancestors.
                        stack.pop()
                        ancestors.remove(x)

        # No cycle was found.
        return False

    def get_link_destination_counts(self):
        counter = {}
        for link in self.links:
            key = (link.src_node, link.src_index, link.dst_node)
            count = counter.get(key, 0)
            counter[key] = count + 1
        return counter

    def get_max_indices_per_node(self):
        max_node_indices = {}
        for link in self.links:
            max_index = max_node_indices.get(link.src_node, 0)
            max_node_indices[link.src_node] = max(max_index, link.src_index)
            max_index = max_node_indices.get(link.dst_node, 0)
            max_node_indices[link.dst_node] = max(max_index, link.dst_index)
        return max_node_indices


class Link:

    def __init__(self, src_node, src_index, dst_node, dst_index):
        assert(isinstance(src_node, str))
        assert(isinstance(dst_node, str))
        assert((src_node == NODE_SOURCE and src_index is None)
               or isinstance(src_index, int) and src_index >= 0)
        assert((dst_node == NODE_SINK and dst_index is None)
               or isinstance(dst_index, int) and dst_index >= 0)

        self.src_node = src_node
        self.src_index = src_index
        self.dst_node = dst_node
        self.dst_index = dst_index

    def __hash__(self):
        return hash((self.src_node, self.src_index,
                     self.dst_node, self.dst_index))

    def __eq__(self, other):
        assert(isinstance(other, Link))

        result = self.src_node == other.src_node and self.src_index == other.src_index and \
            self.dst_node == other.dst_node and self.dst_index == other.dst_index

        return result

    @staticmethod
    def load(input):
        assert(isinstance(input, str))
        if LINK_FORMAT.match(input) is None:
            raise DatasetException(
                "Link must have a source and a destination separated by whitespace.")

        src, dst = input.split()
        src = src.split("/")
        dst = dst.split("/")

        src_node, src_index = src[0], int(
            src[1]) if len(src) > 1 else (src, None)
        dst_node, dst_index = dst[0], int(
            dst[1]) if len(dst) > 1 else (dst, None)

        return Link(src_node, src_index, dst_node, dst_index)

    def dump(self):
        src = "%s/%d" % (self.src_node,
                         self.src_index) if self.src_index is not None else self.src_node
        dst = "%s/%d" % (self.dst_node,
                         self.dst_index) if self.dst_index is not None else self.dst_node
        return "%s %s" % (src, dst)

    def get_reverse(self):
        return Link(self.dst_node, self.dst_index,
                    self.src_node, self.src_index)

    def src(self):
        return (self.src_node, self.src_index)

    def dst(self):
        return (self.dst_node, self.dst_index)


class Class(File):

    def __init__(self, name, categories):
        super(Class, self).__init__(name, "class")
        assert(isinstance(categories, list))
        self.categories = categories

    @staticmethod
    def _load(root, rel_path, name, opener,
              metadata_only=False, subtype="default"):
        path = os.path.join(rel_path, name)
        with opener(root, path + TYPE_EXTENSIONS["class"][subtype], "class", read_only=True, binary=False) as f:
            lines = [x.strip() for x in f.readlines()]

        # Get rid of empty lines.
        lines = [x for x in lines if len(x) > 0]

        # We don't check the line format. We assume each one contains a string
        # which represents the class. We only check the uniqueness.
        categories = set(lines)
        if len(categories) != len(lines):
            raise DatasetException(
                "Class file contains duplicate entries.", path)

        return Class(name, lines)

    def _dump(self, root, rel_path, name, opener):
        path = os.path.join(rel_path, name)
        with opener(root, path + TYPE_EXTENSIONS["class"][self.subtype], "class", read_only=False, binary=False) as f:
            lines = [x + "\n" for x in self.categories]
            f.writelines(lines)


FILE_TYPES = {
    "directory": Directory,
    "tensor": Tensor,
    "category": Category,
    "links": Links,
    "class": Class
}


TYPE_EXTENSIONS = {
    "tensor": {"default": ".ten.npy", "csv": ".ten.csv"},
    "category": {"default": ".cat.txt"},
    "class": {"default": ".class.txt"},
    "links": {"default": ".links.csv"}
}


LINK_FORMAT = re.compile(
    r"^\s*[a-z_]*[0-9a-z_]*(/[0-9]+)?\s+[a-z_]*[0-9a-z_]*(/[0-9]+)?\s*\Z",
    re.ASCII)
NODE_SOURCE = "SOURCE"
NODE_SINK = "SINK"


def validate(args):  # pragma: no cover

    try:
        dataset = Dataset.load(args.root, metadata_only=True)
    except DatasetException as de:
        print("Dataset loading error:")
        print("  Path:      ", de.path)
        print("  Message:   ", de.message)
        sys.exit(-1)

    try:
        schema = dataset.infer_schema()
    except DatasetException as de:
        print("Dataset schema inference error:")
        print("  Path:      ", de.path)
        print("  Message:   ", de.message)
        sys.exit(-1)

    print(json.dumps(schema.dump(), indent=2))
    sys.exit(0)


def match(args):  # pragma: no cover

    # Load dataset and infer its schema.
    try:
        dataset = Dataset.load(args.root, metadata_only=True)
    except DatasetException as de:
        print("Dataset loading error:")
        print("  Path:      ", de.path)
        print("  Message:   ", de.message)
        sys.exit(-1)

    try:
        src_schema = dataset.infer_schema()
    except DatasetException as de:
        print("Dataset schema inference error:")
        print("  Path:      ", de.path)
        print("  Message:   ", de.message)
        sys.exit(-1)

    with open(args.schema) as f:
        dst = json.load(f)

    # Load destination schema.
    try:
        dst_schema = sch.Schema.load(dst)
    except sch.SchemaException as se:
        print("Destination schema validation error:")
        print("  Path:      ", se.path)
        print("  Message:   ", se.message)
        sys.exit(-1)

    # Match dataset (source) schema with destination schema.
    match = dst_schema.match(src_schema, build_matching=True)

    if match:
        print(json.dumps(match, indent=2))
        sys.exit(0)
    else:
        print("Schema match failed.")
        sys.exit(-1)


def generate(args):  # pragma: no cover

    # Load schema.
    with open(args.schema) as f:
        src = json.load(f)
    try:
        schema = sch.Schema.load(src)
    except sch.SchemaException as se:
        print("Schema validation error:")
        print("  Path:      ", se.path)
        print("  Message:   ", se.message)
        sys.exit(-1)

    # Generate dataset and store it.
    dataset = Dataset.generate_from_schema(args.root, schema)
    dataset.dump(args.root)

    sys.exit(0)


if __name__ == "__main__":  # pragma: no cover

    description = "Operations with ease.ml datasets."

    parser = argparse.ArgumentParser(description=description)
    subparsers = parser.add_subparsers(
        description="Please choose one of the following commands to run.")

    validate_parser = subparsers.add_parser(
        "validate",
        help="Check if the given dataset is valid. The inferred schema will be returned.")
    validate_parser.add_argument("root", type=str,
                                 help="Root directory of the dataset.")
    validate_parser.set_defaults(func=validate)

    match_parser = subparsers.add_parser(
        "match", help="Check if a dataset matches with a given schema.")
    match_parser.add_argument("root", type=str,
                              help="Root directory of the dataset.")
    match_parser.add_argument(
        "schema",
        type=str,
        help="JSON file containing the schema description.")
    match_parser.set_defaults(func=match)

    generate_parser = subparsers.add_parser(
        "generate", help="Generate a random dateset from a given schema.")
    generate_parser.add_argument(
        "root",
        type=str,
        help="Root directory of the new dataset to be generated.")
    generate_parser.add_argument(
        "schema",
        type=str,
        help="JSON file containing the schema description.")

    generate_parser.set_defaults(func=generate)

    args = parser.parse_args()
    if hasattr(args, "func"):
        args.func(args)
    else:
        parser.print_help()
