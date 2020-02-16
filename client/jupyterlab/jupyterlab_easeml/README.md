# Easeml JupyterLab extension

Provides the widgets necessary to access easeml webui from the jupyter lab environment

## Prerequisites

* JupyterLab

## Development

For a development install (requires npm version 4 or later), do the following in this repository directory:

```bash
npm install
npm run build
jupyter labextension install .
```

To rebuild the package and the easeml jupyterlab extension:

```bash
npm run build
jupyter lab build
```

