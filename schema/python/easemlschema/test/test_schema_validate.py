import easemlschema.schema as sch
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

            sch_dump = schema.dump()
            dumped_schema = sch.Schema.load(sch_dump)
            self.assertIsInstance(dumped_schema, sch.Schema)

    else:
        def test(self):
            with open(example_file, 'r') as datafile:
                src = json.load(datafile)

            passed = False
            try:
                sch.Schema.load(src)
            except sch.SchemaException:
                passed = True
                pass

            self.assertTrue(passed, "Operation was expected to fail.")

    return test


sch_validate_path = os.path.abspath(
    os.path.join(
        os.path.realpath(__file__),
        "../../../../test-examples/schema/validate"))

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
