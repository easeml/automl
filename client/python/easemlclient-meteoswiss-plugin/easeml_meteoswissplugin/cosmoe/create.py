import argparse
from easemlclient.commands.action import EasemlAction


import json
from datetime import timedelta
from time import time
import multiprocessing
from multiprocessing import Process


import shutil


from easemlclient.model import Connection
from easemlclient.model import Dataset, DatasetSource, DatasetStatus, DatasetQuery


import os

# ================================================================================
# select desired methods
from .dataPreparation import cosmoe_CreateDataByStation_6

from .nnModel.dataPreparation import cosmoe_datapreparation_easeml

from typing import List, Dict


class CreateCosmoeDatasetAction(EasemlAction):
    """ Defines the create dataset action
    """

    def help_description(self) -> str:
        return "Creates a Dataset from COSMOE"

    def action_flags(self) -> List[argparse.ArgumentParser]:
        # ================================================================================
        # ================================================================================
        # ADD PARSER
        parser = argparse.ArgumentParser(
            description='Produce error corrections for MeteoSwiss COSMO-E weather forecast.',
            epilog='Hope you enjoy using the parser!', formatter_class=argparse.MetavarTypeHelpFormatter, add_help=False)
        parser.add_argument('--DB_t', type=str,
                            help='Date of first initialization used for Training set, e.g \'2016010200\'',
                            default='2016010200')
        parser.add_argument('--DE_t', type=str,
                            help='Date of last initialization used for Training set, e.g \'2016010200\'',
                            default='2016010200')
        parser.add_argument('--DB_v', type=str,
                            help='Date of first initialization used for Validation set, e.g \'2016010200\'',
                            default='2018072000')
        parser.add_argument('--DE_v', type=str,
                            help='Date of last initialization used for Validation set, e.g \'2016010200\'',
                            default='2018072000')
        parser.add_argument('--freshScaler', type=int, help='1: fit new scaler on data subset; 0: load stored scaler',
                            choices=range(2), required=False, default=1)
        parser.add_argument('--modelName', type=str, help='Select name of ML model', default='ML_model_1')
        parser.add_argument('--datasetName', type=str, help='Select name of Dataset', default='Cosmoe_Dataset_1')
        parser.add_argument('--n_parallel', type=int, help='Select number of parallel Workers', default=1)
        parser.add_argument('--chunk_size', type=int, help='Select number of initializations grouped into one Chunk', default=1)
        parser.add_argument('--ParamDATA', type=str, help='Select label from COSMO-E data ', default='T_2M')
        parser.add_argument('--ParamOBS', type=str, help='Select label from observation data ', default='t2m')
        parser.add_argument('--ADDRESSdata', type=str, help='Select path to COSMO-E initialization data ',
                            default='/mnt/ds3lab-scratch/bhendj/grids/cosmo/cosmoe')
        parser.add_argument('--ADDRESSprep', type=str, help='Select path to preprocessed data (deprecated) ',
                            default='/mnt/ds3lab-scratch/livios/preprocessed_data/basic_cosmoe/data_prep')
        parser.add_argument('--ADDRESStopo', type=str, help='Select path to topological data',
                            default='/mnt/ds3lab-scratch/livios/preprocessed_data/topo')
        parser.add_argument('--ADDRESSobst', type=str, help='Select path to observation data',
                            default='/mnt/ds3lab-scratch/livios/preprocessed_data/observations')
        parser.add_argument('--DESTINATION', type=str, help='Select path to store data (deprecated)',
                            default='/mnt/ds3lab-scratch/livios/preprocessed_data/cosmoe_workflow')
        return [parser]

    def action(self, config: dict, connection: Connection) -> Dict[str, Dataset]:
        # Select chosable parameters

        DB_t = config['DB_t']  # begin date training set
        # DE_t = '2016010200' # begin date training set
        DE_t = config['DE_t']  # end date training set

        DB_v = config['DB_v']  # begin date valuation set
        # DE_v = '2018072000' # begin date valuation set
        DE_v = config['DE_v']  # end date valuation set

        freshScaler = config[
            'freshScaler']  # select 1: for setting new scaling parameters or select 0: to load a scaler from storage
        modelName = config['modelName']  # select name for storing/loading model as .h5
        datasetName = config['datasetName']  # select name for dataset

        n_parallel = config['n_parallel']  # select number of parallel workers
        chunk_size = config['chunk_size']  # select number of grouped initializations
        ParamDATA = config['ParamDATA']  # select label selected from cosmo-e outputs
        ParamOBS = config['ParamOBS']  # select label selected from observation data, should match with ParamDATA

        # define paths
        ADDRESSdata = config['ADDRESSdata']  # address to data from cosmoe prediction
        ADDRESSprep = config['ADDRESSprep']  # address of preprocessed network ready data
        # ADDRESStopo = '/mnt/ds3lab-scratch/MeteoSwissData/topo' # address to data of topology
        ADDRESStopo = config['ADDRESStopo']  # address to data of topology
        ADDRESSobst = config['ADDRESSobst']  # address to observation data (now ca. 500 stations)
        DESTINATION = config['DESTINATION']  # address to store any files

        # ================================================================================

        # ================================================================================
        # fixed parameters
        onlypreprocessingRun = 0  # select 1: for only preprocessing Data or select 0: for running peprocessing and training or prediction
        trainingRun = 1  # select 1: for training or select 0: for prediction of data


        # select desired parameters from cosmoe data
        ListParam = ['CLCH', 'CLCL', 'CLCM', 'CLCT', 'HPBL', 'PS', 'TD_2M', 'T_2M', 'U_10M', 'VMAX_10M', 'V_10M']
        # select desired parameters from topological data
        TopoListParam = ['HH', 'HH_DIFF', 'FR_LAND', 'SOILTYP', 'LAT', 'LAT_DIFF', 'RLAT', 'LON', 'LON_DIFF', 'RLON',
                         'ABS_2D_DIST', 'ABS_2D_DIST_RAW']

        T = range(121)
        isLocal = 0
        withTopo = 1
        GridSize = 1
        with_chunk_randomization = 0
        # ================================================================================

        dict_time = {}
        time_start = time()
        # ================================================================================
        # in parallel
        m_t = multiprocessing.Manager()
        m_v = multiprocessing.Manager()
        processes = []
        response = {}

        #  setup path to dataset
        # remove old folders
        print("[Helper] Start removing old dataset")
        if os.path.isdir(os.path.join("train")):
            shutil.rmtree(os.path.join("train"))
            shutil.rmtree(os.path.join("val"))
        print("[Helper] Removed old dataset")

        #  start collecting training data
        queue_t = m_t.Queue()
        p_t = Process(target=cosmoe_CreateDataByStation_6.CreateDataByStation, args=(
        freshScaler, onlypreprocessingRun, trainingRun, queue_t, GridSize, DB_t, DE_t, T, ListParam, withTopo,
        TopoListParam, isLocal, n_parallel, chunk_size, ParamDATA, ParamOBS, ADDRESSdata, ADDRESSprep, ADDRESStopo,
        ADDRESSobst, DESTINATION))

        # start collecting validation data
        queue_v = m_v.Queue()
        p_v = Process(target=cosmoe_CreateDataByStation_6.CreateDataByStation, args=(
        0, onlypreprocessingRun, trainingRun, queue_v, GridSize, DB_v, DE_v, T, ListParam, withTopo, TopoListParam,
        isLocal, n_parallel, chunk_size, ParamDATA, ParamOBS, ADDRESSdata, ADDRESSprep, ADDRESStopo, ADDRESSobst,
        DESTINATION))

        #  transform collected data into easeml format
        p = Process(target=cosmoe_datapreparation_easeml.prepEasemlData,
                    args=(1, freshScaler, queue_t, DESTINATION, datasetName, modelName))
        processes.append(p)  #  for training data

        p = Process(target=cosmoe_datapreparation_easeml.prepEasemlData,
                    args=(0, 0, queue_v, DESTINATION, datasetName, modelName))
        processes.append(p)  #  for validation data

        #  start processes
        p_t.start()
        p_v.start()
        for x in processes:
            x.start()
        print("[Helper] All processes started")

        #  joining processes
        p_t.join()
        queue_t.put("stop_flag")
        time_end = time()
        dict_time["end preprocessing training"] = str(timedelta(seconds=(time_end - time_start)))
        print("=============================================================== [Helper] Training data stop flag")

        p_v.join()
        queue_v.put("stop_flag")
        time_end = time()
        dict_time["end preprocessing valuation"] = str(timedelta(seconds=(time_end - time_start)))
        print("=============================================================== [Helper] Validation data stop flag")

        for idx, x in enumerate(processes):
            x.join()
            time_end = time()
            dict_time["end conversion %s" % idx] = str(timedelta(seconds=(time_end - time_start)))

        #  test to generate tar
        print("[Helper] starting tar conversion")

        # TODO Check if tar is installed
        os.system("tar -cvf " + datasetName + ".tar train val")  # generate tar file

        time_end = time()
        dict_time["ease.ml dataset completed"] = str(timedelta(seconds=(time_end - time_start)))

        with open(modelName + '.json', 'w') as fp:
            json.dump(dict_time, fp)

        print('[Helper] ease.ml dataset created in %s.' % str(timedelta(seconds=(time_end - time_start))))

        path = datasetName+".tar"
        with open(path, "rb") as f:
            dataset = Dataset.create(id=datasetName, source=DatasetSource.UPLOAD, name="COSMOE Dataset").post(connection)
            dataset.upload(connection=connection, data=f)
            dataset.status = DatasetStatus.TRANSFERRED
            dataset.patch(connection)

create_cosmoe_dataset = CreateCosmoeDatasetAction()