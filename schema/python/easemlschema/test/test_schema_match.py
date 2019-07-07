import easemlschema.schema as sch
import json
import os
import sys
import unittest

sys.path.append('../.')


class TestExamples(unittest.TestCase):
    pass


def test_generator(example_file_src, positive_example=True):

    example_file_dst = example_file_src[:-len("src.json")] + "dst.json"

    with open(example_file_dst, 'r') as datafile:
        dst = json.load(datafile)
    with open(example_file_src, 'r') as datafile:
        src = json.load(datafile)

    if positive_example:
        def test(self):
            dst_schema = sch.Schema.load(dst)
            src_schema = sch.Schema.load(src)

            match = dst_schema.match(src_schema, build_matching=False)
            self.assertTrue(match)

            match = dst_schema.match(src_schema, build_matching=True)
            self.assertIsNotNone(match)
            self.assertIsInstance(match, sch.Schema)

            expected_constant = match.src_dims is None or len(
                match.src_dims) == 0
            actual_constant = not match.is_variable()
            self.assertEqual(actual_constant, expected_constant)

    else:
        def test(self):
            dst_schema = sch.Schema.load(dst)
            src_schema = sch.Schema.load(src)

            match = dst_schema.match(src_schema, build_matching=False)
            self.assertFalse(match)

            match = dst_schema.match(src_schema, build_matching=True)
            self.assertIsNone(match)

    return test


sch_validate_path = os.path.abspath(
    os.path.join(
        os.path.realpath(__file__),
        "../../../../test-examples/schema/match"))

# Correct schemas.
valid_schemas = []
for root, dirs, files in os.walk(os.path.join(sch_validate_path, "positive")):
    for f in files:
        if f.endswith("src.json"):
            valid_schemas.append(os.path.join(root, f))

# Incorrect schemas.
invalid_schemas = []
for root, dirs, files in os.walk(os.path.join(sch_validate_path, "negative")):
    for f in files:
        if f.endswith("src.json"):
            invalid_schemas.append(os.path.join(root, f))

for example in valid_schemas + invalid_schemas:
    test_name = "test_" + \
        "_".join(example[:-len("-src.json")].replace("-", os.path.sep).split(os.path.sep)[3:])
    positive_example = example in valid_schemas
    test = test_generator(example, positive_example)
    setattr(TestExamples, test_name, test)

if __name__ == '__main__':
    unittest.main()
