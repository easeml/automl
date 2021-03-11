from netCDF4 import Dataset
import xarray as xr
import pickle
import os

import math
import csv
import pandas as pd
import numpy as np
from sklearn.preprocessing import StandardScaler
from sklearn.utils import shuffle
from time import time


from multiprocessing import Queue, Process

#import matplotlib.pyplot as plt
from sklearn.utils import shuffle

from easeml_meteoswissplugin.cosmoe.nnModel.dataPreparation import cosmoe_datapreparation_simplemodel

# method training an neuralnetwork based on network ready data
# Running on one CPU
def prepEasemlData (train_or_validate, freshScaler, Queue, DESTINATION, datasetName, modelName):
    # ===============================
    done = 1
    flag_scaler = freshScaler
    counter = 0 # used for plotting queue size per epoch
    #plt.switch_backend('Agg') # used for plotting queue size per epoch
    list_queue_size = [0] # used for plotting queue size per epoch

    while (done):
        while not Queue.empty():
            print("[Easeml %s] Collect chunk from queue of size %s" % (train_or_validate, Queue.qsize()))
            list_queue_size.append(Queue.qsize()) # used for plotting queue size per epoch
            #plt.switch_backend('Agg') # used for plotting queue size per epoch
            #plt.plot(list_queue_size, label="queue size") # used for plotting queue size per epoch
            #plt.legend() # used for plotting queue size per epoch
            #plt_name = "Plot_queue_size_%s" % (train_or_validate) # used for plotting queue size per epoch
            #plt.savefig(plt_name+'.png') # used for plotting queue size per epoch
            
            counter += 1

            data = Queue.get()  # load data from queue

            if data == "stop_flag": # stop program when receiving stop flag
                done = 0
                break
            if flag_scaler: # if flag_scaler = 1 set new scaler
                flag_scaler = 0
                print("[Easeml] Collected chunk used for scaling")

                cosmoe_datapreparation_simplemodel.convertData(freshScaler = freshScaler, trainingRun = 1, data = data, modelName = modelName, ensemble = 0)
                continue

            # handle easeml data

            data = cosmoe_datapreparation_simplemodel.flatten_cosmoe_training(data) # flatten data
            print("[Easeml %s] Chunk flattend" % (train_or_validate))
            X,y_obs, y_pred = cosmoe_datapreparation_simplemodel.prep_flatten(0, data, modelName)# scale data
            print("[Easeml %s] Chunk scaled" % (train_or_validate))
            cosmoe_datapreparation_simplemodel.convertToEasemlFormat (train_or_validate, X,y_obs, y_pred, DESTINATION, datasetName) # convert to easemlFormate and store
            print("[Easeml %s] Chunk converted to Easeml format" % (train_or_validate))



