#!/usr/bin/env python

import glob
import itertools
import os
import pandas as pd
import random
import shutil
import string
import tarfile

print("Reading raw data.")

files = glob.glob("raw/signs/*/*.sign")
print("Found %d files." % len(files))

all_labels = set()
speaker_labels = dict()
speaker_data = dict()

for f in files:
    speaker = f.split("/")[2][:-1]
    label = f.split("/")[3].split(".")[0][:-1]
    try:
        data = pd.read_csv(f, header=None, usecols=[0,1,2,3,6,7,8,9])
    except:
        continue
    
    speaker_data.setdefault(speaker, []).append({"input" : data, "output" : label})
    speaker_labels.setdefault(speaker, set()).add(label)
    all_labels.add(label)

speakers = {"train" : ["andrew", "john", "stephen"], "val" : ["adam", "waleed"]}

labels = {}
data = {}
for subset in speakers:
    labels[subset] = set().union(*[speaker_labels[s] for s in speakers[subset]])
    data[subset] = list(itertools.chain.from_iterable([speaker_data[s] for s in speakers[subset]]))

print("Building dataset.")

target_dir = os.path.join(os.getcwd(), "prep")
dataset_name = "signs"
input_name = "stream"
feature_name = "data"
output_name = "sign"

if os.path.isdir(os.path.join(target_dir, dataset_name)):
    shutil.rmtree(os.path.join(target_dir, dataset_name))

for subset in speakers:
    
    os.makedirs(os.path.join(target_dir, dataset_name, subset))
    
    for sample in data[subset]:
        # Generate sample name.
        sample_name = ''.join(random.choice(string.ascii_lowercase + string.digits) for _ in range(10))
        
        # Write input CSV file.
        input_dir = os.path.join(target_dir, dataset_name, subset, "input", sample_name, input_name)
        os.makedirs(input_dir)
        sample["input"].to_csv(os.path.join(input_dir, feature_name + ".ten.csv"), header=False, index=False)
        
        # Write output CSV file.
        output_dir = os.path.join(target_dir, dataset_name, subset, "output", sample_name)
        os.makedirs(output_dir)
        with open(os.path.join(output_dir, output_name + ".cat.txt"), "w") as f:
            f.write(sample["output"])
    
    # Write labels to output set.
    output_dir = os.path.join(target_dir, dataset_name, subset, "output")
    with open(os.path.join(output_dir, output_name + ".class.txt"), "w") as f:
        for label in sorted(all_labels):
            f.write(label + "\n")

print("Writing TAR file.")

if not os.path.isdir(os.path.join(os.getcwd(), "final")):
    os.makedirs(os.path.join(os.getcwd(), "final"))

with tarfile.open(os.path.join(os.getcwd(), "final", dataset_name + ".tar"), "w") as tar:
    tar.add(os.path.join(os.getcwd(), "README.md"), arcname="README.md")
    for subset in speakers:
        dirname = os.path.join(target_dir, dataset_name, subset)
        tar.add(dirname, arcname=os.path.basename(dirname))

print("Done.")
