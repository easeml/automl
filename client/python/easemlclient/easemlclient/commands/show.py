import argparse
import sys

from easemlclient.commands.action import EasemlAction
from easemlclient.model import Dataset, DatasetSource, DatasetStatus, DatasetQuery
from easemlclient.model import Job, JobQuery
from easemlclient.model import Task, TaskQuery


class ShowActionGroup(EasemlAction):
    """ Defines the download action group
        Uses the default action (print help)
    """

    def help_description(self):
        return "Shows an item"

    def group_description(self):
        return "Available items to show"


class ShowDatasetAction(EasemlAction):
    """ Defines the show dataset action
    """

    def help_description(self):
        return "Shows dataset(s)"

    def action_flags(self):
        # Optional item id
        optitem_subparser = argparse.ArgumentParser(add_help=False)
        optitem_subparser.add_argument(
            '--id', type=str, help='id, if not set shows all available items.')
        return [optitem_subparser]

    def action(self, config, connection):
        all_datasets = []
        if 'id' in config and config['id']:
            try:
                dataset = Dataset({"id": config["id"]}).get(connection)
                all_datasets.append(dataset)
            except Exception as error:
                if "response" in error and error.response.status_code == 404:
                    print("There is no dataset with id: {}".format(config["id"]))
                print(error)
                sys.exit(1)
            print("Dataset ID: {} : Status: {}".format(
                dataset.id, dataset.status))
        else:
            all_datasets, next_query = DatasetQuery().run(connection)
            for data in all_datasets:
                print("Dataset ID: {} : Status:{}".format(data.id, data.status))
        return all_datasets


class ShowJobAction(EasemlAction):
    """ Defines the show job action
    """

    def help_description(self):
        return "Shows Job(s)"

    def action_flags(self):
        # Optional item id
        optitem_subparser = argparse.ArgumentParser(add_help=False)
        optitem_subparser.add_argument(
            '--id', type=str, help='id, if not set shows all available items.')
        return [optitem_subparser]

    def action(self, config, connection):
        all_jobs = []
        if 'id' in config and config['id']:
            try:
                job = Job({'id': config['id']}).get(connection)
                all_jobs.append(job)
            except Exception as error:
                if "response" in error and error.response.status_code == 404:
                    print("There is no job with id: {}".format(config["id"]))
                print(error)
                exit(1)
            print("Job ID: {} : Status:{}".format(job.id, job.status))
        else:
            all_jobs, query = [], JobQuery()
            next_result, next_query = [], query
            while next_query is not None:
                next_result, next_query = next_query.run(connection)
                all_jobs.extend(next_result)
            for j in all_jobs:
                print(j.id, j.status)

            if not len(all_jobs):
                print("No jobs found")
        return all_jobs


class ShowTaskAction(EasemlAction):
    """ Defines the show job action
    """

    def help_description(self):
        return "Shows Task(s)"

    def action_flags(self):
        # Optional item id
        optitem_subparser = argparse.ArgumentParser(add_help=False)
        group = optitem_subparser.add_mutually_exclusive_group()
        group.add_argument('--task-id', type=str, help='Show a specific task')
        group.add_argument('--job-id', type=str,
                           help='Show tasks for a job-id')
        return [optitem_subparser]

    def action(self, config, connection):
        print(config)
        all_tasks = []
        if 'task_id' in config and config['task_id']:
            try:
                task = Task({'id': config['task_id']}).get(connection)
                all_tasks.append(task)
            except Exception as error:
                if "response" in error and error.response.status_code == 404:
                    print("There task in job id: {}".format(config["task_id"]))
                print(error)
                exit(1)
            for t in all_tasks:
                print("Task ID: {}, Quality: {}, Status:{}".format(
                    t.id, t.quality, t.status))
        else:
            if 'job_id' in config and config['job_id']:
                job = Job({'id': config['job_id']})
                all_tasks, query = [], TaskQuery(
                    job=job, order_by="quality", order='desc')
            else:
                all_tasks, query = [], TaskQuery(
                    order_by="quality", order='desc')

            next_result, next_query = [], query
            while next_query is not None:
                next_result, next_query = next_query.run(connection)
                all_tasks.extend(next_result)
            for t in all_tasks:
                print("Task ID: {}, Quality: {}, Status:{}".format(
                    t.id, t.quality, t.status))

            if len(all_tasks) == 0:
                print("No tasks found")
        return all_tasks


show_action_group = ShowActionGroup()
show_dataset = ShowDatasetAction()
show_job = ShowJobAction()
show_task = ShowTaskAction()
