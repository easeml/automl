"""This module contains tools to build and manipulate modules.
"""

import docker

from io import BytesIO

client = docker.from_env()

def build_module() -> None:
    pass

def upload_module(name: str) -> None:
    image = client.images.get(name)
    generator = image.save()
    memfile = BytesIO()
    for chunk in generator:
        memfile.write(chunk)
    memfile.seek(0)
    # Upload memfile.
