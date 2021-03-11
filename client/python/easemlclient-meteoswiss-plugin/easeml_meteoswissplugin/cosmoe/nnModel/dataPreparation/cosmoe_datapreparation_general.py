#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# The code was influenced by preprocessing code used on a project of the "Data Science Lab" class in autumn 2018 (Louis de Gaste & Felix Schaumann)
import json
import os
import pickle as pkl
import json
import re
import sys
from datetime import timedelta
from multiprocessing import Pool
from multiprocessing import Process
from time import time

import numpy as np
import pandas as pd
import xarray as xr
import datetime as dt

from nnModel.dataPreparation.utils import cosmoe_DataUtils
from nnModel.dataPreparation import cosmoe_datapreparation_simplemodel

# this exception is used to break innter loop and continue outer loop with next iteration
class SkipException(Exception):
    pass

def startPool(Queue, GridSize, DateBegin, DateEnd, PredictionWindow, ListParam, WithTopo, TopoListParam, n_parallel, chunk_size, ParamDATA, ParamOBS, ADDRESSdata, ADDRESSprep, ADDRESStopo, ADDRESSobst, DESTINATION):
    time_begin = time()

    if DateEnd < DateBegin:
        raise Exception('DateEnd is smaller than DateBegin.')
    """
    # TODO: all paths should come from initialization file
    # define paths to data
    ADDRESSdata = '/mnt/ds3lab-scratch/livios/preprocessed_data/basic_cosmoe/data_prep' # COSMO-E outputs
    ADDRESStopo = '/mnt/ds3lab-scratch/MeteoSwissData/topo' # base address of topo files
    # ADDRESSobst = '/mnt/ds3lab-scratch/MeteoSwissData/observations' # base adress of obs files
    ADDRESSobst = '/mnt/ds3lab-scratch/livios/preprocessed_data/observations' # base adress of obs files, files for ca. 500 stations
    DESTINATION = '/mnt/ds3lab-scratch/livios/preprocessed_data/cosmoe_workflow' # target directory for all generated files
    """
    
    # =======================
    # get preprocessed chunks
    folders = cosmoe_DataUtils.getFilesToProcess(ADDRESSdata, DESTINATION, ListParam, 'Station', DateBegin, DateEnd)
    
    # ======================
    # filter folders by desired date
    begin, end = -1, -1
    for idx, folder in enumerate(folders):
        if folder[0] >= DateBegin:
            begin = idx
            break

    for idx, folder in enumerate(folders):
        if folder[1] <= DateEnd:
            end = idx
        else:
            break

    if begin == -1 or end == -1:
        raise Exception('Could not find start or end in array.')

    folders = folders[begin:end+1]
    print('%s chunks are ready to be preprocessed for your model.' % len(folders))


    # =======================
    # distribute work to pool of workers. To do so:
    # - split work into number of workers
    if n_parallel <= 1:
        folder_splits = [folders]
    else:
        n_folders = len(folders)
        indices = np.linspace(0, n_folders, n_parallel+1).astype(int)
        folder_splits = [folders[indices[i]:indices[i + 1]] for i in range(n_parallel)]

    folder_splits = [l for l in folder_splits if len(l) > 0]
    time_setup = time()

    
    # - initialize pool of workers and start them async
    with Pool(processes=n_parallel) as pool:
        process_results = []
        for idx_split, split in enumerate(folder_splits):
            print('[Process %s] Range [%s, %s] queued.' % (idx_split, split[0], split[-1]))
             # start async process to load data
            process_results.append(pool.apply_async(GetData, (Queue, idx_split, ADDRESSdata, ADDRESStopo, ADDRESSobst,
                                                              DESTINATION, ListParam, TopoListParam, GridSize,
                                                              WithTopo, split, PredictionWindow, chunk_size,
                                                              ParamOBS, ParamDATA)))  
        # - aggregate results from all processes
        for ps_idx, ps_result in enumerate(process_results):
            # sync processes
            _ = ps_result.get()
            print('[Process %s] Synchronized after data creation.' % ps_idx)

    # take timestamp after completing all processes
    time_end = time()

    """
    # dump preprocessing information in a descriptive JSON file
    preprocessing_information = {
        'grid_size': GridSize,
        'data_begin': DateBegin,
        'data_end': DateEnd,
        'parameters': ListParam,
        'future_hours': PredictionWindow,
        'n_processes' : n_parallel,
        'time_setup': str(timedelta(seconds=(time_setup - time_begin))),
        'time_preprocessing' : str(timedelta(seconds=(time_end - time_setup)))
    }

    preprocessing_information_json = json.dumps(preprocessing_information)
    f = open(DESTINATION + '/setup.json', 'w')
    f.write(preprocessing_information_json)
    f.close()
    """

    print('Preprocessing sucessfully finished in %s.' %  str(timedelta(seconds=(time_end - time_begin))))



def GetData(Queue, processId, ADDRESSdata, ADDRESStopo, ADDRESSobst, DESTINATION, ListParam, TopoListParam,
            GridSize, WithTopo, Folders, PredictionWindow, chunk_size, ParamOBS, ParamDATA):
    print("CreateDataByStation.py/GetData")

    # load time invariant data features
    NAME = '/mnt/ds3lab-scratch/livios/preprocessed_data/basic_cosmoe'+'/time_invariant_data_per_station.pkl'  # TODO: automatic name generation, get path to file to process
    station_position_data, station_grid_data = cosmoe_DataUtils.load_time_invariant_data(processId = processId, Name = NAME)
    
    for idx_chunk, chunk in enumerate(Folders):
        # ====================
        # load current netCDF4 data chunk
        # all chunks have the same structure, example data chunk
        """
        Dimensions:        (ensemble: 21, init: 14, lead: 121, prameters: 11, station_id: 547, time_feature: 5)
        Coordinates:
        * init           (init) object '2016010200' '2016010412' '2016010700' ...
        * lead           (lead) int64 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 ...
        * time_feature   (time_feature) object 'cos_hour' 'sin_hour' 'cos_day' ...
        * station_id     (station_id) int32 1 2 3 4 6 7 8 9 10 11 13 14 15 17 18 ...
        * ensemble       (ensemble) int64 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 ...
        * prameters      (prameters) object 'CLCH' 'CLCL' 'CLCM' 'CLCT' 'HPBL' ...
        Data variables:
            time_data      (init, lead, time_feature) float64 ...
            cosmo_data     (station_id, init, lead, ensemble, prameters) float64 ...
            temp_forecast  (station_id, init, lead, ensemble) float64 ...
            temp_station   (station_id, init, lead) float64 ...
        Attributes:
            station_id:  10919
        """
        # TODO: Method to load netCDF data
        # get path to file to process

        NAME = ADDRESSdata + '/' + chunk[2]
        # load meta data of dataset
        time_start = time()
        dataset = xr.open_dataset(NAME) 
        time_end = time()
        print(('[Process %s] ' + chunk[2] + ' opened in %s') % (processId, round(time_end-time_start,2)))

        # store dimension parameters in dictionary
        dimension_data = {}
        dimension_data['ensemble'] = dataset.sizes['ensemble']
        dimension_data['init'] = dataset.sizes['init']
        dimension_data['lead'] = dataset.sizes['lead']
        dimension_data['parameters'] = dataset.sizes['prameters']
        dimension_data['station_id'] = dataset.sizes['station_id']
        dimension_data['time_feature'] = dataset.sizes['time_feature']

        # load cosmoe_data
        time_start = time()
        cosmoe_data = dataset['cosmo_data'].data
        time_end = time()
        print(('[Process %s] ' + chunk[2] + ' loaded cosmoe data in %s') % (processId, round(time_end-time_start,2)))

        # load time_data
        time_start = time()
        time_data = dataset['time_data'].data
        time_end = time()
        print(('[Process %s] ' + chunk[2] + ' loaded time data in %s') % (processId, round(time_end-time_start,2)))

        # load temp_forcast
        time_start = time()
        temp_forecast = dataset['temp_forecast'].data
        time_end = time()
        print(('[Process %s] ' + chunk[2] + ' loaded temp_forecast in %s') % (processId, round(time_end-time_start,2)))

        # load time_data
        time_start = time()
        temp_station = dataset['temp_station'].data
        time_end = time()
        print(('[Process %s] ' + chunk[2] + ' loaded temp_station in %s') % (processId, round(time_end-time_start,2)))

        # ===========================================================================================================================================
        # ADD DESIRED DATAPREPARATION METHODS BELOW

        cosmoe_datapreparation_simplemodel.convertData(processId, Queue, station_position_data, station_grid_data, cosmoe_data, time_data, temp_forecast, temp_station, dimension_data)
        # cosmoe_datapreparation_lstm.convertData(processId, Queue, station_position_data, station_grid_data, cosmoe_data, time_data, temp_forecast, temp_station, dimension_data)
        print('[Process %s] Data split %s of %s successfully preprocessed!' % (processId, idx_chunk+1, len(Folders)))

    