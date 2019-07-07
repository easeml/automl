import easemlschema.schema as sch
import easemlschema.dataset as ds
import json
import os
import sys
import unittest

sys.path.append('../.')


class TestExamples(unittest.TestCase):
    pass


def test_generator(example_dataset, positive_example=True):

    if positive_example:
        def test(self):

            with open(example_dataset + ".json", 'r') as datafile:
                src = json.load(datafile)

            dst_schema = sch.Schema.load(src)

            dataset = ds.Dataset.load(example_dataset)
            src_schema = dataset.infer_schema()

            match = dst_schema.match(src_schema, build_matching=False)
            self.assertTrue(match)

    else:
        def test(self):

            passed = False
            try:
                dataset = ds.Dataset.load(example_dataset, metadata_only=True)
                dataset.infer_schema()
            except ds.DatasetException:
                passed = True
                pass

            self.assertTrue(passed, "Operation was expected to fail.")

    return test


sch_validate_path = os.path.abspath(
    os.path.join(
        os.path.realpath(__file__),
        "../../../../test-examples/dataset/infer"))

# Correct schemas.
valid_schemas = []
positive_root = os.path.join(sch_validate_path, "positive")
for d in os.listdir(positive_root):
    if os.path.isdir(os.path.join(positive_root, d)):
        valid_schemas.append(os.path.join(positive_root, d))

# Incorrect schemas.
invalid_schemas = []
negative_root = os.path.join(sch_validate_path, "negative")
for d in os.listdir(negative_root):
    if os.path.isdir(os.path.join(negative_root, d)):
        invalid_schemas.append(os.path.join(negative_root, d))

for example in valid_schemas + invalid_schemas:
    test_name = "test_" + \
        "_".join(example.replace("-", os.path.sep).split(os.path.sep)[3:])
    positive_example = example in valid_schemas
    test = test_generator(example, positive_example)
    setattr(TestExamples, test_name, test)

if __name__ == '__main__':
    unittest.main()
