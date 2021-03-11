from netCDF4 import Dataset
import xarray as xr
import pickle

import math
import csv
import pandas as pd
import numpy as np
import tensorflow as tf
from tensorflow import keras
from sklearn.preprocessing import StandardScaler
from sklearn.utils import shuffle
from keras.utils import multi_gpu_model
from time import time
import numpy as np
import matplotlib.pyplot as plt
from sklearn.utils import shuffle

def getTrainingData(Data):
    time_start = time()    
    chunk_data = Data

    # load datachunk as dataframe, remove NaN entries and 
    dataframe = pd.DataFrame(Data)
    dataframe = dataframe.fillna(0)

    # remove rows of zeros (should be already checked in preprocessing)
    """
    dataframe = np.array(dataframe)
    
    dataframe_nonzero = []
    for x in dataframe: # TODO: SLOW?
        print(x)
        if x[3] != 0: dataframe_nonzero.append(x)
    dataframe = pd.DataFrame(dataframe_nonzero)
    """

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
    y = dataframe.loc[: , 'y_obs'] - dataframe.loc[: , 'y_pred']
    y = np.array(y)

    # scale data
    sc =  StandardScaler()
    X= sc.fit_transform(X)
    
    time_end = time()
    print(('[Training] Data prepared in %s') % (round(time_end-time_start,2)))

    return X,y

def getPredictionData(Data):
    dataframe = Data

    # load datachunk as dataframe, remove NaN entries and 
    dataframe = pd.DataFrame(chunk_data)
    dataframe = dataframe.dropna()

    # remove rows of zeros (should be already checked in preprocessing)
    dataframe = np.array(dataframe)
    dataframe_nonzero = []
    for x in dataframe:
        if x[3] != 0: dataframe_nonzero.append(x)

    dataframe = pd.DataFrame(dataframe_nonzero)

    # get shape of dataframe (rows, columnes)
    shape = dataframe.shape()

    # name columnes
    dataframe.columns = ["y_obs", "y_pred"] + range(shape[1]-2)
   
    # prepare data
    # select X, select y
    X = dataframe.loc[: , '0':]
    y = dataframe.loc[: , 'y_obs'] - dataframe.loc[: , 'y_pred']

    # scale data
    sc =  StandardScaler()
    X= sc.fit_transform(X)
    return X,y
