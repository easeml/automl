import renku  # type: ignore
import os
import tempfile
import shutil
from renku.core.models.cwl.annotation import Annotation  # type: ignore
from uuid import uuid1
import json

def load_json(file_path: str) -> dict:
    with open(file_path) as json_file:
        data = json.load(json_file)
    return data

@renku.core.plugins.hookimpl
def pre_run(tool):
    """Plugin Hook that gets called at the start of a ``renku run`` call.
    :param run: A ``WorkflowTool`` object that will get executed by
                ``renku run``.
    """
    #for v in dir(tool):
    #    print("### ",v,getattr(tool, v))
    if "easemlclient" in tool.command_line:
        print("Running easemlclient as a tool inside Renku")
    return tool

@renku.core.plugins.hookimpl
def cmdline_tool_annotations(tool):
    """Plugin Hook to add ``Annotation`` entry list to a ``WorkflowTool``.

    called by renku run

    :param run: A ``WorkflowTool`` object to get annotations for.
    :returns: A list of ``renku.core.models.cwl.annotation.Annotation``
              objects.
    """
    print("@@@ renku POST HOOK")
    annotations=[]
    sys_temp = tempfile.gettempdir()
    easeml_temp_folder = os.path.join(sys_temp,'easeml_tmp')
    if "easemlclient" in tool.command_line:
        print("Collecting easemlclient results and cleaning up")

        for subdir, dirs, files in os.walk(easeml_temp_folder):
            for file in files:
                filepath = os.path.join(subdir, file)
                if filepath.endswith("mls_renku_metadata.json"):
                    # print(filepath)
                    annotations.append(Annotation(
                        id='_:annotation{}'.format(uuid1().fields[0]),
                        source="MLS plugin",
                        body=load_json(filepath)
                    ))
                elif filepath.endswith(".json"):
                    # print(filepath)
                    annotations.append(Annotation(
                        id='_:annotation{}'.format(uuid1().fields[0]),
                        source="Ease.ml orchestration",
                        body=load_json(filepath)
                    ))
        shutil.rmtree(easeml_temp_folder)
    return annotations

#@renku.core.plugins.hookimpl
#def process_run_annotations(run):
#    """``process_run_annotations`` hook implementation.
#
#        Called by: renku.core.commands.graph.build(), renku log
#    """
#    print("#### Ease.ml annotations processing")
#    return []
