#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# The code was influenced by preprocessing code used on a project of the "Data Science Lab" class in autumn 2018 (Louis de Gaste & Felix Schaumann)
import json
import os
import pickle as pkl
import json
import re
import math
import sys
from datetime import timedelta
from multiprocessing import Pool
from multiprocessing import Process

from time import time

import random
import numpy as np
import pandas as pd
import xarray as xr
#import matplotlib.pyplot as plt
import datetime as dt

from .utils import cosmoe_DataUtils_2


# this exception is used to break innter loop and continue outer loop with next iteration
class SkipException(Exception):
    pass

# method to preprocess COSMO-E output to later train a machine learning approach
# GridSize=1
# DataBegin
# DateEnd
# PredictionWindow
# ListParam
# WithTopo
# TopoListParam
# isLocal
# n_parallel
def CreateDataByStation(freshScaler, onlypreprocessingRun, trainingRun, Queue, GridSize, DateBegin, DateEnd, PredictionWindow, ListParam, WithTopo, TopoListParam, isLocal, n_parallel, chunk_size, ParamDATA, ParamOBS, ADDRESSdata, ADDRESSprep, ADDRESStopo, ADDRESSobst, DESTINATION):
    time_begin = time()

    if DateEnd < DateBegin:
        raise Exception('DateEnd is smaller than DateBegin.')
    
    # get all COSMO-E files (as dictionary) that are in the given time interval and have not yet been processed and thus do not
    # already exists in the output folder
    folders = cosmoe_DataUtils_2.getFilesToProcess(ADDRESSdata, DESTINATION, ListParam, 'Station', DateBegin, DateEnd)
    folders_keys = [f for f in folders.keys()]
    folders_keys.sort()

    # calculate begin and end index of array to exclude files, that are not in the specified time interval
    begin, end = -1, -1
    for idx, folder in enumerate(folders_keys):
        if folder >= DateBegin:
            begin = idx
            break

    for idx, folder in enumerate(folders_keys):
        if folder <= DateEnd:
            end = idx
        else:
            break

    if begin == -1 or end == -1:
        raise Exception('Could not find start or end in array.')
    
    # reduce dictionary to initializations between begin and end
    folders_keys = folders_keys[begin:end+1]
    filterByKey = lambda keys: {x: folders[x] for x in keys}
    folders = (filterByKey(folders_keys))
    print('[Process 0] %s files within time period were found.' % len(folders_keys))

    if len(folders_keys) is 0:
        print("=============================================================== [Process 0] no files to process" )
        return

    # ===================
    # Part to set Datascaler
    # pick random keys from dictionary to approx. data
    if freshScaler:
        time_start = time()
        cur_keys = folders_keys
        if len(folders_keys) > 735:
            cur_keys = folders_keys[-730:] # take data corresponding to last year
        chunk_scaler = int(math.ceil(len(cur_keys)*0.02))
        print('[Process 0] Computing new scaler paramters based on %s evenly spaced initializations' % (chunk_scaler))
        
        idx = np.round(np.linspace(0, len(cur_keys) - 1, chunk_scaler)).astype(int)
        folders_keys_random = np.array(cur_keys)[idx]
        print('[Process 0] Random initializations selected %s' % (folders_keys_random))

        split = filterByKey(folders_keys_random)
        data = GetData(onlypreprocessingRun, trainingRun, Queue, 0, ADDRESSdata, ADDRESStopo, ADDRESSobst,
                                                                DESTINATION, ListParam, TopoListParam, GridSize,
                                                                WithTopo, split, PredictionWindow, isLocal, chunk_scaler,
                                                                ParamOBS, ParamDATA)
        
        Queue.put(data) # add scaler data to queue
        print('=============================================================== [Process 0] added data to queue')

        time_end = time()
        print('[Process 0] Scaler chunk preprocessed in %s.' %  str(timedelta(seconds=(time_end - time_start))))
    # ===================

    # take timestamp after set-up
    time_setup = time()


    # =============================
    # chunk timeline into chunks of size chunksize
    folders_keys =  [f for f in folders.keys()] # get all keys of current dictionarry split
    folders_keys.sort()
    
    # split list of keys into chunks of size chunk_size 
    folders_keys_subsplits = list(cosmoe_DataUtils_2.chunks(folders_keys, chunk_size))
    print('[Process 0] Splitted into %s chunks ' % (len(folders_keys_subsplits)))

    # split folders dictionary into according to splitted list of keys into list of chunked dictionarys.
    folder_splits = [] #
    for folder_split in folders_keys_subsplits:
        filterByKey = lambda keys: {x: folders[x] for x in keys}
        folder_splits.append(filterByKey(folder_split))


    # =============================
    # sliding preprocessing window
    start_time = time()
    processes = []
    processesId = []
    iterator_chunks = iter(folder_splits)

    # start n_parallel processes
    with Pool(processes=n_parallel) as pool:
        for idx_p in range(n_parallel):
            try:
                chunk = next(iterator_chunks) # get chunk for process
            except:
                break
            # set new process
            processes.append(pool.apply_async(GetData, (onlypreprocessingRun, trainingRun, Queue, idx_p, ADDRESSdata, ADDRESStopo, ADDRESSobst,
                                                              DESTINATION, ListParam, TopoListParam, GridSize,
                                                              WithTopo, chunk, PredictionWindow, isLocal, chunk_size,
                                                              ParamOBS, ParamDATA)))              
            processesId.append(idx_p)

            split_keys = [f for f in chunk.keys()]
            split_keys.sort()
            print('[Process %s] Range [%s, %s] queued.' % (idx_p, split_keys[0], split_keys[-1])) #TODO: might cause error since its now a dictionary

        # feed processes with next chunk as soon as joined
        while True:
            p = processes.pop(0) # get first process in list
            data = p.get()
            curId = processesId.pop(0) # get id of first process in list

            if data is not 0:
                Queue.put(data) # add data to queue    
                print('=============================================================== [Process %s] added data to queue' % (curId))


            try:
                chunk = next(iterator_chunks) # get next chunk for process
            except:
                if not processes: # stop when processes list is empty
                    break
                else: # else wait for results from processes
                    continue

            processes.append(pool.apply_async(GetData, (onlypreprocessingRun, trainingRun, Queue, curId, ADDRESSdata, ADDRESStopo, ADDRESSobst,
                                                              DESTINATION, ListParam, TopoListParam, GridSize,
                                                              WithTopo, chunk, PredictionWindow, isLocal, chunk_size,
                                                              ParamOBS, ParamDATA)))   
            processesId.append(curId)

            split_keys = [f for f in chunk.keys()]
            split_keys.sort()
            print('[Process %s] Range [%s, %s] queued.' % (curId, split_keys[0], split_keys[-1]))
        

    end_time = time()
    print('[Process 0] Preprocessing successfully finished in %s.' %  str(timedelta(seconds=(end_time - start_time))))

def GetData(onlypreprocessingRun, trainingRun, Queue, processId, ADDRESSdata, ADDRESStopo, ADDRESSobst, DESTINATION, ListParam, TopoListParam,
            GridSize, WithTopo, Folders, PredictionWindow, isLocal, chunk_size, ParamOBS, ParamDATA):
    # processId: (int) -> the id of the process running this method
    # ADDRESSdata: (string) -> base path to COSMO-1 data
    # ADDRESStopo: (string) -> base path to all topology files
    # ADDRESSobs: (string) -> base path to all observation files
    # DESTINATION: (string) -> base path to target output folder
    # GridSize: (int)-> side length of square around each station
    # WithTopo: (bool)-> whether we want to generate preprocessed time invariant features for each station
    # Folders: (dict(list)) -> dictionary of all cosmoe files to be processed, key: date and initialization, e.g. ['15031200', '15031212', ...], per key: one file ('filename') per parameter e.g.  'cosmo-e_15031200_2_TM.nc'
    # PredictionWindow: (list of int) -> all future hours t's [t,t+1,t+2,...] being processed, e.g. y_t, y_t+1,...
    # isLocal: (bool) -> for setting the right paths if the script is running on a local machine or in cloud, etc.
    
    # to fix parallelization errors, each process gets its own set of TOPO and OBS files
    # OBS = xr.open_dataset(ADDRESSobst + '/process_%s/meteoswiss_t2m_20151001-20180331.nc' % processId) # open station observations
    OBS = xr.open_dataset(ADDRESSobst + '/process_%s/meteoswiss_t2m_20160101-20190715_asDWH.nc' % processId) # open station observations
    #TOPO = xr.open_dataset(ADDRESStopo + '/process_%s/topodata.nc' % processId)  # open topo data
    #TOPO = xr.open_dataset(ADDRESStopo + '/process_%s/cosmoetopo.nc' % processId)  # open topo data
    TOPO = xr.open_dataset(ADDRESStopo + '/process_%s/cosmoe_surface_gitterDB.nc' % processId)  # open topo data

    # load all station ids
    stationIds = OBS['station_id'].data
 
    # generate a view on temperature observation at each station
    TempObs = OBS[ParamOBS].sel(station_id = stationIds)
        
    # =================
    # 1) localize stations on cosmoe grid (2.2km * 2.2km, dim: 127*188)
    # computed when first cosmoe parameterfile is opened
    cosmoe_closestGridPointPerStation = None # list of tuples to store (lat,lon)
    # ============================
  
    # =============================
    # Agregate necessary data for current chunk: initialize data variables
    DATA = None
    TOPOData = None
    FileLabels = []
    skipped_files = 0

    # we now start iterating through the current split of dictionary: Each key (day/initialization), each file (per parameter), each station
    chunk = Folders
    for idx_folder, folder in enumerate(chunk):  # loop over all  outputs of COSMO-E, 
        start_time = time()

        try:
            # mark start of preprocessing of n-th file
            print('[Process %s] Start processing %s' % (processId, folder))

            # adapt file index to skipped files
            idx_folder = idx_folder - skipped_files 

            # check that we do not process a data point before the first observation or after the last observation
            OBS_DB = str(OBS['time'].data[0])
            OBS_DE = str(OBS['time'].data[-1])
            
            if int(folder[:8]) < int(OBS_DB[0:4]+OBS_DB[5:7]+OBS_DB[8:10]):
                print('[Process %s] Skipped %s, before first observation' % (processId, folder))
                raise SkipException()

            if (int(folder[:8])-5) > int(OBS_DE[0:4]+OBS_DE[5:7]+OBS_DE[8:10]): # max. lead is 120h = 5days
                print('[Process %s] Skipped %s, after last observation' % (processId, folder))
                raise SkipException()
            
            # loop over all parameters per initialization
            for idx_param, file_param in enumerate(sorted([f for f in Folders[folder].keys()])):
                # ====================
                # open one dataset per time
                name_param = cosmoe_DataUtils_2.getCosmoeParameter(ListParam, file_param) # get name of current parameter
                
                try:
                    time_start = time()

                    NAME = ADDRESSdata + '/' + Folders[folder][file_param] # get path to file to process
                    dataset = xr.open_dataset(NAME) # load current netCDF4 dataset

                    time_end = time()
                    print(('[Process %s] ' + Folders[folder][file_param] + ' opened in %s') % (processId, round(time_end-time_start,2)))
                except:
                    print(('[Process %s] unable to load file %s') % (processId, Folders[folder][file_param]))
                    continue
                
                # Compute COSMOE grid data once, see description above
                if cosmoe_closestGridPointPerStation is None:
                    cosmoe_closestGridPointPerStation = []
                    cosmoe_GPSgrid = np.dstack((dataset['lat_1'][:, :], dataset['lon_1'][:, :]))  # 127*188*2 grid of lon lat values of each square of cosmoe resuolution

                    for S in stationIds:
                        dist = cosmoe_GPSgrid - np.array([[OBS['lat'].sel(station_id = S), OBS['lon'].sel(station_id = S)]])
                        dist *= dist
                        Id = (dist.sum(axis=2)).argmin()
                        Id = np.unravel_index(Id, (127, 188))
                        cosmoe_closestGridPointPerStation += [Id]  # Id=(x,y) coordinates of the station (approx.. to the closest point)

                t = dataset['time'].data # get times series of current intialization

                # Initialiaze needed data structures once
                if DATA is None:
                    files_in_range = len(chunk)
                    # DATA stores all COSMOE data, DATA: [# stations, # files in split,  #PredictionWindow, #prediction line, #ListParam]
                    DATA = np.zeros((len(stationIds), files_in_range, len(PredictionWindow), 21 ,len(ListParam)))
                    # TempForecast stores forcasted temperature, TempForecast: [# stations, #files in split, #PredictionWindow, # prediction lines]
                    TempForecast = np.zeros((len(stationIds), files_in_range, len(PredictionWindow), 21))
                    # Target: stores measured temperature, Target: [#stations, # files in split, #PredictionWindow]
                    Target = np.zeros((len(stationIds),files_in_range, len(PredictionWindow)))
                    # TimeStamp: [# files in split, #PredictionWindow]
                    TimeStamp = np.zeros((files_in_range,len(PredictionWindow)))
                    # TimeData: stores time info, TimeData: [# files in range, # PredictionWindow]
                    TimeData = np.zeros((files_in_range, len(PredictionWindow), 5))

                # ======================
                # Get timedata: Transform day and hour into a cyclic datetime feature
                days_rad = (cosmoe_DataUtils_2.passed_days_per_month_dict[int(folder[4:6])] + int(folder[6:8])) / 365 * (2 * np.pi)
                hours = (int(folder[6:8])) % 24
                hour_rad = hours / 24 * (2 * np.pi)

                for idx_T, T in enumerate(PredictionWindow):
                    TimeData[idx_folder,idx_T] = [np.cos(hour_rad), np.sin(hour_rad), np.cos(days_rad), np.sin(days_rad), T/121 ]
                
                TimeStamp[idx_folder] = t
                # ======================
                
                # ======================
                # compute topodata once
                if TOPOData is None:
                    start_time = time()
                    # TOPOData: stores data from TOPO file
                    TOPOData = np.zeros((len(stationIds),3))
                    TOPO_FR_LAND = TOPO['FR_LAND'].data.squeeze() # create dataview on current parameter
                    TOPO_HSURF = TOPO['HSURF'].data.squeeze() # create dataview on current parameter
                    TOPO_SOILTYP = TOPO['SOILTYP'].data.squeeze() # create dataview on current parameter
                    for idx_S, S in enumerate(stationIds):  # loop over stations
                        station_coord = cosmoe_closestGridPointPerStation[idx_S]  # get coordinate idx of current station
                        TOPOData[idx_S, 0] = TOPO_FR_LAND[station_coord[0], station_coord[1]] # load COSMO E data
                        TOPOData[idx_S, 1] = TOPO_HSURF[station_coord[0], station_coord[1]] # load COSMO E data
                        TOPOData[idx_S, 2] = TOPO_SOILTYP[station_coord[0], station_coord[1]] # load COSMO E data
                    end_time = time()
                    print(('[Process %s] ' + Folders[folder][file_param] + ' TOPO processed in %s') % (processId, round(end_time-start_time,2)))

                # ======================

                # ======================
                # get data of current parameter
                time_start = time()
                MAP = dataset[name_param].data.squeeze() # create dataview on current parameter
                
                for idx_S, S in enumerate(stationIds):  # loop over stations
                    station_coord = cosmoe_closestGridPointPerStation[idx_S]  # get coordinate idx of current station
                    DATA[idx_S, idx_folder, :, :, idx_param] = MAP[:, : , station_coord[0], station_coord[1]] # load COSMO E data
                    # Store label parameter seperately, convert from Kelvin to Celsius TODO: do we want conversion?
                    if name_param == ParamDATA: TempForecast[idx_S, idx_folder, :, :] =  (MAP[:, : , station_coord[0], station_coord[1]]) - (273.15) 
                    
                dataset.close()
                # ======================
                
                # ======================
                # get data from observation parameter
                try:
                    Target[:, idx_folder, :] = np.transpose(TempObs.sel(time = t).data) # load observation data
                except RuntimeError:
                    print('Error with time=%s.' % t)
                    raise
                
                # ======================

                time_end = time()
                print(('[Process %s] ' + Folders[folder][file_param] + ' processed in %s') % (processId, round(time_end-time_start,2)))

            FileLabels += [folder] #store initializaiton information
            # print status of current initialization
            print(('[Process %s] Files in chunk finished: %s of %s') % (processId, idx_folder+1, len(chunk)))

            if idx_folder % 10 == 0:
                sys.stdout.flush()
        except SkipException:
            continue
        
    OBS.close() 
    TOPO.close()

    if DATA is None:
        print('[Process %s] No data was preprocessed. Possibly all files to preprocess were skipped, because their date is before'
            'the first observation.' % processId)
        return None

    # =========================================
    # Version to store selected data as netCDF
    leads = PredictionWindow
    ensemble = range(21)
    time_features = ['cos_hour', 'sin_hour', 'cos_day', 'sin_day', 'lead']
    dims = ('station_id','init','lead', 'ensemble', 'parameters')
    cosmo_data = xr.DataArray(DATA,
                                dims=dims,
                                coords=[stationIds, FileLabels, leads, ensemble, ListParam])
    temp_forecast = xr.DataArray(TempForecast,
                                    dims=('station_id','init', 'lead', 'ensemble'),
                                    coords=[stationIds, FileLabels, leads, ensemble])
    temp_station = xr.DataArray(Target,
                                dims=('station_id', 'init', 'lead'),
                                coords=[stationIds, FileLabels, leads])
    time_data = xr.DataArray(TimeData,
                                dims=('init', 'lead', 'time_feature'),
                                coords=[FileLabels, leads, time_features])
    time_data.attrs['time_stamp'] = TimeStamp
    ds = xr.Dataset({'cosmo_data': cosmo_data,
                        'temp_forecast': temp_forecast,
                        'temp_station': temp_station,
                        'time_data': time_data})
    ds.attrs['station_id'] = S
    
    """
    file_name = FileLabels[0] + '_' + FileLabels[-1] + '_data_process_' + str(processId) + '.nc' 
    ds.to_netcdf(DESTINATION +'/data_prep/'+ file_name)
    """

    # store dimension parameters in dictionary
    dataset = ds
    dimension_data = {}
    dimension_data['ensemble'] = dataset.sizes['ensemble']
    dimension_data['init'] = dataset.sizes['init']
    dimension_data['lead'] = dataset.sizes['lead']
    dimension_data['parameters'] = dataset.sizes['parameters']
    dimension_data['station_id'] = dataset.sizes['station_id']
    dimension_data['time_feature'] = dataset.sizes['time_feature']

    print('[Process %s] Data chunk successfully preprocessed!' % processId)

    # add data to queue
    if not onlypreprocessingRun:
        data = (TOPOData, cosmo_data.data, time_data.data, temp_forecast.data, temp_station.data, dimension_data)
        return data
    return 0


