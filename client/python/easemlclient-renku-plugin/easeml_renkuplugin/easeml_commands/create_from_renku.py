import sys
import argparse
import traceback
from easemlclient.commands.action import EasemlAction
from easemlclient.model import Dataset, DatasetSource, DatasetStatus, DatasetQuery
from easemlclient.model import ModuleQuery, ModuleType, ModuleStatus, ModuleSource, Module
from easemlclient.model import Job, JobQuery, User
from easemlclient.model.core import Connection
from easemlclient.model.type import ApiType
from requests.exceptions import HTTPError

import requests

from renku.api import Dataset as RenkuDataset

from typing import List, Dict


def exists(path):
    r = requests.head(path)
    print(r)
    print(r.status_code)
    return r.status_code == requests.codes.ok


class CreateDatasetFromRenkuAction(EasemlAction):
    """ Defines the create dataset action
    """

    def help_description(self) -> str:
        return "Creates a Dataset from Renku"

    def action_flags(self) -> List[argparse.ArgumentParser]:
        # dataset create
        dataset_subparser = argparse.ArgumentParser(add_help=False)
        dataset_subparser.add_argument(
            '--dataset-name', type=str, help='Renku Dataset name.', default="", required=True)
        dataset_subparser.add_argument(
            '--dataset-file-name', help='Dataset description.', type=str, default="", required=True)
        dataset_subparser.add_argument(
            '--access-key',
            type=str,
            help='Data-source specific accessKey, i.e. oauth token.',
            default="")
        return [dataset_subparser]

    def action(self, config: dict, connection: Connection) -> Dict[str, Dataset]:
        response = {}
        try:
            datasets = RenkuDataset.list()
            if len(datasets) == 0:
                print("No datasets available in this project")
                return response

            renku_dataset = None
            renku_names = []
            for rd in datasets:
                renku_names.append(rd.name)
                if config["dataset_name"] == rd.name:
                    renku_dataset = rd
                    break
            if renku_dataset is None:
                print(f"The desired dataset: {config['dataset_name']} is not part of this project")
                print(f"The possible options are:")
                for rdn in renku_names:
                    print(f"\t- {rdn}")
                return response

            if len(renku_dataset.files) == 0:
                print("The dataset is empty")
                return response

            renku_dataset_file = None
            renku_names = []

            for rdf in renku_dataset.files:
                renku_names.append(rdf.name)
                if config["dataset_file_name"] == rdf.name:
                    renku_dataset_file = rdf
                    break

            if renku_dataset_file is None:
                print(f"The desired file: {config['dataset_file_name']} is not part of this dataset")
                print(f"The possible options are:")
                for rfn in renku_names:
                    print(f"\t- {rfn}")
                return response

            upload_local = True
            # if renku_dataset_file._dataset_file.url:
            #    if exists(renku_dataset_file._dataset_file.url):
            #        print(f"Found in url {renku_dataset_file._dataset_file.url}")
            #        file_source = DatasetSource.DOWNLOAD
            #        file_source_address = renku_dataset_file._dataset_file.url
            #        upload_local = False
            #    else:
            #        print(f"Download url {renku_dataset_file._dataset_file.url}, not reachable")
            if upload_local:
                print("Preparing to upload")
                file_source = DatasetSource.UPLOAD
                file_source_address = renku_dataset_file.path

            if "id" in config:
                dataset_id = config["id"]
            else:
                # dataset_id = renku_dataset_file.name
                dataset_id = renku_dataset.name

            if file_source == DatasetSource.UPLOAD:
                with open(file_source_address, "rb") as f:
                    dataset = Dataset.create(
                        id=dataset_id,
                        source=file_source,
                        access_key=config["access_key"],
                        name=renku_dataset_file.name
                    ).post(connection)
                    dataset.upload(connection=connection, data=f)
                    dataset.status = DatasetStatus.TRANSFERRED
                    dataset.patch(connection)
            else:
                dataset = Dataset.create(
                    id=dataset_id,
                    source=file_source,
                    name=renku_dataset_file.name,
                    source_address=file_source_address,
                    access_key=config["access_key"]
                ).post(connection)

            response["response"] = dataset
            print("Dataset with id {} successfully created".format(dataset.id))
        except HTTPError as e:
            if e.response.status_code == 409:
                print("There is an existing dataset with the id: {}".format(
                    config["id"]))
            print(e)
        except Exception as e:
            print("Unable to create Dataset")
            print(e)
        print(response)
        return response


create_dataset_from_renku = CreateDatasetFromRenkuAction()
