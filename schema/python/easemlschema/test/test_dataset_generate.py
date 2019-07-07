import easemlschema.schema as sch
import easemlschema.dataset as ds
import json
import os
import sys
import unittest

sys.path.append('../.')


class TestExamples(unittest.TestCase):
    pass


def test_generator(example_file, positive_example=True):

    if positive_example:
        def test(self):

            with open(example_file, 'r') as datafile:
                src = json.load(datafile)

            schema = sch.Schema.load(src)
            self.assertIsInstance(schema, sch.Schema)

            dataset = ds.Dataset.generate_from_schema("_", schema)

            inferred_schema = dataset.infer_schema()

            match = schema.match(inferred_schema, build_matching=False)
            self.assertTrue(match)

    return test


sch_validate_path = os.path.abspath(
    os.path.join(
        os.path.realpath(__file__),
        "../../../../test-examples/dataset/generate"))

# Correct schemas.
valid_schemas = []
for root, dirs, files in os.walk(os.path.join(sch_validate_path, "positive")):
    for f in files:
        if f.endswith(".json"):
            valid_schemas.append(os.path.join(root, f))

# Incorrect schemas.
invalid_schemas = []
for root, dirs, files in os.walk(os.path.join(sch_validate_path, "negative")):
    for f in files:
        if f.endswith(".json"):
            invalid_schemas.append(os.path.join(root, f))

for example in valid_schemas + invalid_schemas:
    test_name = "test_" + \
        "_".join(example[:-len(".json")].replace("-", os.path.sep).split(os.path.sep)[3:])
    positive_example = example in valid_schemas
    test = test_generator(example, positive_example)
    setattr(TestExamples, test_name, test)

if __name__ == '__main__':
    unittest.main()
