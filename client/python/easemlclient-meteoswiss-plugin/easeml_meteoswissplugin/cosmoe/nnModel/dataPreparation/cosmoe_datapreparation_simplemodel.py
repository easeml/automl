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
import time as TIME

import glob
import itertools
import random
import shutil
import string
import tarfile
import csv

import numpy as np
import pandas as pd
import xarray as xr
import datetime as dt
from sklearn.preprocessing import StandardScaler
from sklearn.preprocessing import MinMaxScaler
import joblib

# this exception is used to break innter loop and continue outer loop with next iteration
class SkipException(Exception):
    pass

def convertData(freshScaler, trainingRun, data, modelName, ensemble):
    print('[Training] Started converting data chunk')
    # ===========================================================================================================================================
    # Queue to transfer data to training process, given data contains:
    # - station_position_data: Structure = [#station, #station parameters], station parameters = ['lon', 'lat', 'height']
    # - station_grid_data: Structure = [#station, #grid parameters], grid parameters = TIME_INV_features
    # - cosmoe_data: Structure = [#station, #initialization, #lead time, #ensemble members, #cosmoe parameter], cosmoe parameter =  ListParam 
    # - time_data: Structure = [#initialization, #leadtime, #time parameters], time parameters = ['cos_hour', 'sin_hour', 'cos_day', 'sin_day','lead']
    # - temp_forecast: Structure = [#station, #initialization, # lead time, #ensemble members]
    # - temp_station: Structure = [#station, #initialization]
    # - dimension_data: Structure = Dictionary containing size of dimension for ensemble, init, lead, parameters, station_id, time_features

    # ====================================
    if trainingRun:
        DATA_NN = flatten_cosmoe_training(data) # flatten data
    else:
        DATA_NN = flatten_cosmoe_prediction(data, ensemble) # flatten data
    X,y_obs, y_pred = prep_flatten(freshScaler, DATA_NN, modelName) # scale data

    return X,y_obs, y_pred


def flatten_cosmoe_training(data):
    # ====================================
    # mapping from queue output
    TOPOData = data[0] 
    cosmoe_data = data[1]
    time_data = data[2] 
    temp_forecast = data[3]
    temp_station = data[4]
    dimension_data = data[5]      
    
    # load dimension data
    n_ensemble = dimension_data['ensemble']
    n_init = dimension_data['init']
    n_lead = dimension_data['lead']
    n_parameters = dimension_data['parameters']
    n_station_id = dimension_data['station_id']
    n_time_features = dimension_data['time_feature']
    
    # ====================================
    # IMPLEMENT FLATTENING BELOW
    # start processing data
    time_start = time()
    DATA_NN = []
    for idx_cur in range(n_init): # loop over files in range
        for idx_T, T in enumerate(range(n_lead)): # loop over leadtimes in PredictionWindow
            time_features = list(time_data[idx_cur,idx_T]) # time features 
            for idx_S, S in enumerate(range(n_station_id)): # loop over stations of stationIds
                cosmoe_features = list(cosmoe_data[idx_S, idx_cur, idx_T, 0, :])
                TOPO_features = list(TOPOData[idx_S])
                y_std = np.std(temp_forecast[idx_S, idx_cur, idx_T, :])
                y_mean = np.mean(temp_forecast[idx_S, idx_cur, idx_T, :])
                y_obs = temp_station[idx_S, idx_cur, idx_T]
                y_pred = temp_forecast[idx_S, idx_cur, idx_T, 0]
                # append DATA_nn, version with python list
                DATA_NN.append( [y_obs] + [y_pred] + [y_std] + [y_mean] + cosmoe_features + TOPO_features + time_features)
    time_end = time()
    print(('[Training] Prepared DATA_NN in %s') % (round(time_end-time_start,2)))

    return DATA_NN

def flatten_cosmoe_prediction(data, ensemble):
    # mapping from queue output
    TOPOData = data[0] 
    cosmoe_data = data[1]
    time_data = data[2] 
    temp_forecast = data[3]
    temp_station = data[4]
    dimension_data = data[5]            
    
    # load dimension data
    n_ensemble = dimension_data['ensemble']
    n_init = dimension_data['init']
    n_lead = dimension_data['lead']
    n_parameters = dimension_data['parameters']
    n_station_id = dimension_data['station_id']
    n_time_features = dimension_data['time_feature']

    """
    # ====================================
    # IMPLEMENT FLATTENING BELOW
    # standard version
    # start processing data
    time_start = time()
    DATA_NN = []
    for idx_cur in range(n_init): # loop over files in range
        for idx_T, T in enumerate(range(n_lead)): # loop over leadtimes in PredictionWindow
            time_features = list(time_data[idx_cur,idx_T]) # time features 
            #for idx_S, S in enumerate(range(n_station_id)): # loop over stations of stationIds
            idx_S = 100
            cosmoe_features = list(cosmoe_data[idx_S, idx_cur, idx_T, 0, :])
            TOPO_features = TOPOData[idx_S]
            y_std = np.std(temp_forecast[idx_S, idx_cur, idx_T, :])
            y_mean = np.mean(temp_forecast[idx_S, idx_cur, idx_T, :])
            y_obs = temp_station[idx_S, idx_cur, idx_T]
            y_pred = temp_forecast[idx_S, idx_cur, idx_T, 0]
            # append DATA_nn, version with python list
            DATA_NN.append( [y_obs] + [y_pred] + [y_std] + [y_mean] + cosmoe_features + [TOPO_features] + time_features)
    time_end = time()
    print(('[Prediction] Prepared DATA_NN in %s') % (round(time_end-time_start,2)))
    """
    
    # ====================================
    # IMPLEMENT FLATTENING BELOW
    # test version
    # start processing data
    time_start = time()
    DATA_NN = []
    for idx_cur in range(n_init): # loop over files in range
        #for idx_m in range(21):
        idx_m = ensemble
        for idx_T, T in enumerate(range(n_lead)): # loop over leadtimes in PredictionWindow
            time_features = list(time_data[idx_cur,idx_T]) # time features 
            #for idx_S, S in enumerate(range(n_station_id)): # loop over stations of stationIds
            idx_S = 100
            cosmoe_features = list(cosmoe_data[idx_S, idx_cur, idx_T, idx_m, :])
            TOPO_features = list(TOPOData[idx_S])
            y_std = np.std(temp_forecast[idx_S, idx_cur, idx_T, :])
            y_mean = np.mean(temp_forecast[idx_S, idx_cur, idx_T, :])
            y_obs = temp_station[idx_S, idx_cur, idx_T]
            y_pred = temp_forecast[idx_S, idx_cur, idx_T, 0]
            # append DATA_nn, version with python list
            DATA_NN.append( [y_obs] + [y_pred] + [y_std] + [y_mean] + cosmoe_features + TOPO_features + time_features)
    time_end = time()
    print(('[Prediction] Prepared DATA_NN in %s') % (round(time_end-time_start,2)))
    

    return DATA_NN

def prep_flatten(freshScaler, DATA_NN, modelName):
    # ====================================
    # IMPLEMENT SCALING BELOW
    time_start = time()  
    chunk_data = DATA_NN

    # load datachunk as dataframe, remove NaN entries and 
    dataframe = pd.DataFrame(chunk_data)
    dataframe = dataframe.dropna()

    # get shape of dataframe (rows, columnes)
    shape = dataframe.shape

    # name columnes
    columne_list = []
    for integer in (list(range(0,shape[1]-2))):
        columne_list.append(str(integer))
     
    dataframe.columns = ["y_obs", "y_pred"] + columne_list
   
    # prepare data
    # select X, select y
    X = dataframe.loc[: , '0':]
    y_obs = dataframe.loc[: , 'y_obs']
    y_pred = dataframe.loc[: , 'y_pred']
    y_pred = np.array(y_pred)
    y_obs = np.array(y_obs)

    # scale data
    X = np.array(X)
    
    scaler_file = "scaler_"+modelName+".save"

    if freshScaler:
        scaler = MinMaxScaler()
        scaler.fit(X)
        # Save it
        joblib.dump(scaler, scaler_file) 
        print("=============================================================== [Flatten] Set new scaler")
    else:
        flag = 1
        while flag:
            try:
                # Load it 
                scaler = joblib.load(scaler_file)
                print("=============================================================== [Flatten] Loaded scaler")
                flag = 0
            except:
                print("================================================================ [Flatten] Waiting for scaler")
                TIME.sleep(50)
            
        
    X= scaler.transform(X) # TODO: removed for testing purposes
    
    time_end = time()
    print(('[Flatten] Data scaled in %s') % (round(time_end-time_start,2)))
    return X,y_obs,y_pred
   

def convertToEasemlFormat (train_or_validate, X,y_obs, y_pred, DESTINATION, datasetName):
    print("[Easeml conversion] Building dataset.")

    if train_or_validate:
        #target_dir = os.path.join(datasetName, "train")
        target_dir = os.path.join("train")
    else:
        #target_dir = os.path.join(datasetName, "val")
        target_dir = os.path.join("val")
    input_name = "test1"
    feature_name = "data"
    output_name = "label"

    print("[Easeml conversion] Create new folders.")
    # Generate sample name.
    sample_name = ''.join(random.choice(string.ascii_lowercase + string.digits) for _ in range(10))

    # Write input numpy file.
    input_dir = os.path.join(target_dir, "input", sample_name, input_name)
    os.makedirs(input_dir)
    np.save(os.path.join(input_dir, feature_name + ".ten.npy"), X)

    # Write output numpy file.
    output_dir = os.path.join(target_dir, "output", sample_name, input_name)
    os.makedirs(output_dir)
    test = np.zeros((len(y_pred),1))
    test[:,0] = (y_obs - y_pred)
    np.save(os.path.join(output_dir, output_name + ".ten.npy"), test)
    print("[Easeml conversion] Chunk conversion completed.")
