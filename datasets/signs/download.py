#!/usr/bin/env python

import gzip
import io
import requests
import tarfile

print("Downloading and extracting.")
zip_file_url = "https://archive.ics.uci.edu/ml/machine-learning-databases/auslan-mld/allsigns.tar.gz"
r = requests.get(zip_file_url, stream=True)
data = gzip.GzipFile(fileobj=io.BytesIO(r.content))
z = tarfile.TarFile(fileobj=data)
z.extractall(path="raw")

print("Done.")
