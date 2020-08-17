# -*- coding: utf-8 -*-

# Learn more: https://github.com/kennethreitz/setup.py

from setuptools import setup, find_packages  # type: ignore
from easeml_renkuplugin import __version__

with open("README.md", "r") as fh:
    README = fh.read()

# The main source of truth for install requirements of this project is the requirements.txt file.
with open("requirements.txt", "r") as f:
    REQUIREMENTS = f.readlines()

setup(
    name='easemlclient-renku',
    version=__version__+'.dev.2',
    description='Plug-in that interfaces between easeml and renku',
    long_description=README,
    long_description_content_type="text/markdown",
    author='Bojan Karlas, Leonel Aguilar',
    author_email='bojan.karlas@gmail.com, leonel.aguilar.m@gmail.com',
    url='https://github.com/DS3Lab/easeml',
    license='MIT',
    install_requires=REQUIREMENTS,
    packages=find_packages(),
    include_package_data=True,
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent"
    ],
    entry_points={"easemlclient.hook.tree": ["renku = easeml_renkuplugin.easeml.plugin"],

                  "renku": ["easeml = easeml_renkuplugin.renku.plugin"],
                  },
)
