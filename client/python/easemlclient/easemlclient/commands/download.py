import argparse
import sys
from easemlclient.commands.action import EasemlAction
from easemlclient.model import Dataset, DatasetSource, DatasetStatus, DatasetQuery
from easemlclient.model import Task, TaskQuery


class DownloadActionGroup(EasemlAction):
    """ Defines the download action group
        Uses the default action (print help)
    """

    def help_description(self):
        return "Downloads an item"

    def group_description(self):
        return "Available items to download"


class DownloadDatasetAction(EasemlAction):
    """ Defines the download dataset action
    """

    def help_description(self):
        return "Downloads dataset"

    def action_flags(self):
        # item single id
        item_subparser = argparse.ArgumentParser(add_help=False)
        item_subparser.add_argument(
            '--id', type=str, help='id [REQUIRED]', required=True)
        return [item_subparser]

    def action(self, config, connection):
        dataset = Dataset({"id": config["id"]}).get(connection)
        # TODO fix ID dataset.id != config["id"]
        print("Dataset ID: {}, Status:{}".format(dataset.id, dataset.status))
        dataset_file = dataset.get_dataset(connection)
        open("dataset.tar", "wb").write(dataset_file)
        return [dataset]


class DownloadModelAction(EasemlAction):
    """ Defines the download model action
    """

    def help_description(self):
        return "Downloads model"

    def action_flags(self):
        # Task id or Job id
        opttaskjob_subparser = argparse.ArgumentParser(add_help=False)
        opttaskjob_subparser.add_argument(
            '--predictions',
            action='store_true',
            help='Download model predictions')
        opttaskjob_subparser.add_argument(
            '--metadata', action='store_true', help='Download model metadata')
        opttaskjob_subparser.add_argument(
            '--logs', action='store_true', help='Download model logs')
        opttaskjob_subparser.add_argument(
            '--image', action='store_true', help='Download model image')
        group = opttaskjob_subparser.add_mutually_exclusive_group()
        group.add_argument('--task-id', type=str,
                           help='Download model from a specific task')
        group.add_argument('--job-id', type=str,
                           help='Download best model from a specific Job')
        return [opttaskjob_subparser]

    def action(self, config, connection):
        if not (config['predictions'] or config['logs']
                or config['metadata'] or config['image']):
            print("Nothing set to be download use -h to see options")
            sys.exit(1)

        if config['task_id']:
            task = Task({"id": config['task_id']}).get(connection)
        elif config['job_id']:
            print("Getting best task for job {}".format(config['job_id']))
            job = Job({'id': config['job_id']}).get(connection)
            tasks, _ = TaskQuery(job=job, order_by="quality",
                                 order='desc').run(connection)
            if not len(tasks):
                print("No tasks available for job {}".format(config['job_id']))
                sys.exit(1)
            task = tasks[0]
        else:
            print("No Task ID provided")
            sys.exit(1)
        print("Task ID: {}, Quality: {}, Status: {}".format(
            task.id, task.quality, task.status))
        print("Downloading... Please wait")
        if config['predictions']:
            predictions = task.get_predictions(connection)
            open("predictions.tar", "wb").write(predictions)

        if config['logs']:
            logs = task.get_logs(connection)
            open("logs.tar", "wb").write(logs)

        if config['metadata']:
            metadata = task.get_metadata(connection)
            open("metadata.tar", "wb").write(metadata)

        if config['image']:
            image = task.get_image(connection)
            open("image.tar", "wb").write(image)
        return [task]


download_action_group = DownloadActionGroup()
download_dataset = DownloadDatasetAction()
download_model = DownloadModelAction()
