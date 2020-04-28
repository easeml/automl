# Ease.ml JupyterLab extension

Provides the widgets necessary to access ease.ml webui from the jupyter lab environment

## Prerequisites

* JupyterLab

## Installation

* Jupyter lab: Settings > Enable Extension Manager
* Extension Manager: Search @easeml/jupyterlab_easeml

## Development

For a development install (requires npm version 4 or later), do the following in this repository directory:

```bash
npm install
npm run build
jupyter labextension install .
```

To rebuild the package and the ease.ml jupyterlab extension:

```bash
npm run build
jupyter lab build
```

