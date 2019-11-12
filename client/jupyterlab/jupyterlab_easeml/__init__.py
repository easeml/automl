__version__ = '0.0.1'
from .extension import load_jupyter_server_extension  # noqa

def _jupyter_server_extension_paths():
    return [{
        "module": "jupyterlab_easeml"
    }]
