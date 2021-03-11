import matplotlib
matplotlib.use('tkagg')

from netCDF4 import Dataset
import xarray as xr
import pickle
import os

import math
from math import sqrt
import csv
import pandas as pd
import numpy as np
import tensorflow as tf
from tensorflow import keras
from time import time
import scipy
import keras.backend as K

from sklearn.preprocessing import StandardScaler
from sklearn.metrics import mean_squared_error
from sklearn.utils import shuffle
from sklearn.metrics import mean_squared_error

from keras.utils import multi_gpu_model
from keras.models import load_model


import matplotlib.pyplot as plt
import numpy as np
import matplotlib.pyplot as plt
from sklearn.utils import shuffle

from nnModel.utils import cosmoe_DataUtils_training
from nnModel.Model import cosmoe_controlmodel
from nnModel.Model import cosmoe_simplemodel
from nnModel.Model.cosmoe_simplemodel import crps_cost_function
from nnModel.dataPreparation import cosmoe_datapreparation_simplemodel

# method for prediction with a neuralnetwork based on network ready data
# Running on one CPU
def prediction (Queue, GridSize, DB, DE, T, n_parallel, chunk_size, ParamOBS, ParamDATA, ADDRESSprep, ListParam, TopoListParam, n_gpus, num_epochs, num_batch, with_chunk_randomization, withTopo, modelName):
    # ===============================
    # model definition, compile and load stored model weights
    model = cosmoe_simplemodel.cosmoe_model(n_gpus, num_epochs, num_batch, with_chunk_randomization, modelName)
    print("[Prediction] loaded model architecture")

    model.compile(loss= 'mse', optimizer='adam')
    print("[Prediction] Compiled model") 

    model.load_weights("model_weights"+modelName+".h5")
    print("[Prediction] loaded stored model weights")
    
    # ===============================
    # predict model
    counter = 2
    done = 1
    while (done):
        while not Queue.empty():
            print("[Prediction] Collected chunk from queue of size %s" % (Queue.qsize()))
            data = Queue.get() # load data chunk from queue
            time_start = time()
            if data == "stop_flag": # stop when receiving stop flag
                done = 0
                break
            
            # prediction with loaded data
            plt.switch_backend('Agg')
            
            print("[Prediction] Predicted collected chunk ")

            for i in range(21):
                X, y_obs, y_pred = cosmoe_datapreparation_simplemodel.convertData(freshScaler = 0, trainingRun = 0, data = data, modelName = modelName, ensemble = i) # convert data
            
                counter += 1
                plt.plot(y_obs, label="y_obs")
                plt.plot(y_pred, label="y_cosmoe")
                y_net = model.predict(X)
                
                plt.plot(y_net[:,0]+y_pred, label=("y_net_%s" % (counter)))
                #plt.plot(y_pred + y_net[:,0], label="y_net + y_cosmoe")
                
                print("MSE y_cosmoe %s" % (mean_squared_error(y_obs,y_pred)))
                print("MSE y_cosmoe+y_net %s" % (mean_squared_error(y_obs,y_net[:,0]+y_pred)))
            #plt.legend()
            plt_name = "Plot_prediction_%s" % (counter)
            plt.savefig(plt_name+'.png')
            plt.clf()
            
            time_end = time()
            print(('[Prediction] Chunk predicted in %s') % (round(time_end-time_start,2)))

