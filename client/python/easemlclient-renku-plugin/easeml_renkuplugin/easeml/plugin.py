import easemlclient.commands
from easemlclient.model import Dataset, Task, TaskStatus
from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType
from typing import List, Dict, Any
import os
import tempfile
import json
import tarfile
import io
import shutil


@easemlclient.commands.hookimpl
def easemlclient_add_pre_action(config: dict, connection: Connection) -> dict:
    return config


@easemlclient.commands.hookimpl
def easemlclient_add_post_action(config: dict, connection: Connection, response: Dict[str, ApiType]) -> Dict[str, Any]:
    print("@@@ easemlclient POST HOOK")
    hook_response = {'name': __name__}
    sys_temp = tempfile.gettempdir()
    easeml_temp_folder = os.path.join(sys_temp, 'easeml_tmp')

    if not os.path.exists(easeml_temp_folder):
        os.makedirs(easeml_temp_folder)
    else:
        shutil.rmtree(easeml_temp_folder)
        os.makedirs(easeml_temp_folder)
    for key, r in response.items():
        if issubclass(type(r), ApiType):
            if type(r) is Task:
                print("Found task {}, retrieving metadata".format(r.id))
                if r.status == TaskStatus.COMPLETED:
                    metadata = r.get_metadata(connection)
                    hook_response['metadata'] = metadata
                    metadata_io = io.BytesIO(metadata)
                    with tarfile.open("metadata", mode='r', fileobj=metadata_io) as tar:
                        for member_info in tar.getmembers():
                            member_info.name = r.id.replace("/", "_") + "_" + member_info.name
                            tar.extract(member_info, path=easeml_temp_folder)
                else:
                    print("Task ID: {} not ready to extract its metadata, status: {}".format(r.id, r.status))
            fd, path = tempfile.mkstemp(prefix="easeml_", suffix=".json", dir=easeml_temp_folder)
            try:
                with os.fdopen(fd, 'w') as tmp:
                    # do stuff with temp file
                    json.dump(r._dict, tmp)
            except Exception as error:
                print("Error while writing tempory data: {}".format(fd))
        elif r is None:
            continue
        else:
            msg = "Unhandled return type: val {}, type {}".format(r, type(r))
            print(msg)
            raise Exception(msg)
    return hook_response
