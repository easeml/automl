#!/usr/bin/env python

import os
import shutil

dirs = ["prep", "raw"]

for d in dirs:
    if os.path.isdir(os.path.join(os.getcwd(), d)):
        shutil.rmtree(os.path.join(os.getcwd(), d))

print("Done.")