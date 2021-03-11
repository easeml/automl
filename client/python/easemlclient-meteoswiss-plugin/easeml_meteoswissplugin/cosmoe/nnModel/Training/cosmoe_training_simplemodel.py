from netCDF4 import Dataset
import xarray as xr
import pickle
import os

import math
import csv
import pandas as pd
import numpy as np
import tensorflow as tf
from tensorflow import keras
from sklearn.preprocessing import StandardScaler
from sklearn.utils import shuffle
from tensorflow.keras.utils import multi_gpu_model
from keras.models import load_model
import keras.backend as K
from time import time


from multiprocessing import Queue, Process

import matplotlib.pyplot as plt
from sklearn.utils import shuffle

from nnModel.utils import cosmoe_DataUtils_training
from nnModel.Model import cosmoe_controlmodel
from nnModel.Model import cosmoe_simplemodel
from nnModel.Model.cosmoe_simplemodel import crps_cost_function
from nnModel.dataPreparation import cosmoe_datapreparation_general
from nnModel.dataPreparation import cosmoe_datapreparation_simplemodel


# method training an neuralnetwork based on network ready data
# Running on one CPU
def training (freshScaler, Queue, freshModel, GridSize, DB, DE, T, n_parallel, chunk_size, ParamOBS, ParamDATA, ADDRESSprep, ListParam, TopoListParam, n_gpus, num_epochs, num_batch, with_chunk_randomization, withTopo,modelName):
    # ===============================
    # model definition, compile and load stored model weights
    model = cosmoe_simplemodel.cosmoe_model(n_gpus, num_epochs, num_batch, with_chunk_randomization, modelName)
    print("[Training] loaded model architecture")

    model.compile(loss= 'mse', optimizer= 'adam')
    print("[Training] Compiled model")    

    if freshModel:
        model.save_weights("model_weights"+modelName+".h5")
        print("[Training] fresh model weights initialized")

    model.load_weights("model_weights"+modelName+".h5")
    print("[Training] loaded stored model weights")

    #parallel_model = tf.keras.utils.multi_gpu_model(model, gpus = 3)
    
    # ===============================
    # fit model
    done = 1
    flag_scaler = freshScaler
    counter = 0
    plt.switch_backend('Agg')

    list_queue_size = [0] # used for plotting queue size per epoch

    while (done):
        while not Queue.empty():
            print("[Training] Collect chunk from queue of size %s" % (Queue.qsize()))
            list_queue_size.append(Queue.qsize())
            counter += 1
            
            # load data from queue and convert to model format
            time_start = time()
           
            data = Queue.get()

            if data == "stop_flag": # stop training when receiving stop flag
                done = 0
                break
            if flag_scaler:
                flag_scaler = 0
                cosmoe_datapreparation_simplemodel.convertData(freshScaler = freshScaler, trainingRun = 1, data = data, modelName = modelName, ensemble = 0)
                print("[Training] Chunk for scaling (skip training)")
                continue

            X, y_obs, y_pred = cosmoe_datapreparation_simplemodel.convertData(freshScaler = 0, trainingRun = 1, data = data, modelName = modelName, ensemble = 0) # convert data

            error = y_obs-y_pred # compute error between station observations and cosmoe prediction

            history = model.fit(X, error, epochs = num_epochs, validation_split=0.10, batch_size=num_batch, verbose = 1) # do training

            time_end = time()
            print(('[Training] Trained collected chunk in %s') % (round(time_end-time_start,2)))

            plt.switch_backend('Agg')
            plt.plot(list_queue_size, label="queue size")
            plt.legend()
            plt_name = "Plot_queue_size"
            plt.savefig(plt_name+'.png')

    # save model after each data from queue
    model.save_weights("model_weights"+modelName+".h5")
    print("[Training] Model saved")
            