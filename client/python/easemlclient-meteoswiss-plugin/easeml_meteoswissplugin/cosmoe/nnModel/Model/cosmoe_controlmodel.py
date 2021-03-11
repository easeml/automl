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
from keras.utils import multi_gpu_model
from keras.models import load_model
import keras.backend as K

import numpy as np
import matplotlib.pyplot as plt
from sklearn.utils import shuffle


from tensorflow.python.client import device_lib

def get_available_gpus():
    local_device_protos = device_lib.list_local_devices()
    return [x.name for x in local_device_protos if x.device_type == 'GPU']

# method training an neuralnetwork based on network ready data
# Running on one CPU
def cosmoe_model (n_gpus, num_epochs, num_batch, with_chunk_randomization, modelName):
    # ===============================
    # model definition
    model = keras.Sequential()
    model.add(keras.layers.Dense(1024, activation = 'relu', input_shape = (21,)))
    model.add(keras.layers.Dropout(0.2))
    model.add(keras.layers.Dense(512, activation = 'relu'))
    model.add(keras.layers.Dropout(0.2))
    model.add(keras.layers.Dense(64, activation = 'relu'))
    model.add(keras.layers.Dense(1))

    print("[Helper] Model setup")
    
    return model

# loss function from https://github.com/slerch/ppnn/blob/master/nn_postprocessing/nn_src/losses.py
def crps_cost_function(y_true, y_pred, theano=False):
    """Compute the CRPS cost function for a normal distribution defined by
    the mean and standard deviation.
    Code inspired by Kai Polsterer (HITS).
    Args:
        y_true: True values
        y_pred: Tensor containing predictions: [mean, std]
        theano: Set to true if using this with pure theano.
    Returns:
        mean_crps: Scalar with mean CRPS over batch
    """

    # Split input
    mu = y_pred[:, 0]
    sigma = y_pred[:, 1]
    
    # To stop sigma from becoming negative we first have to 
    # convert it the the variance and then take the square
    # root again. 
    var = K.square(sigma)
    # The following three variables are just for convenience
    loc = (y_true - mu) / K.sqrt(var)
    phi = 1.0 / np.sqrt(2.0 * np.pi) * K.exp(-K.square(loc) / 2.0)
    Phi = 0.5 * (1.0 + tf.math.erf(loc / np.sqrt(2.0)))
    # First we will compute the crps for each input/target pair
    crps =  K.sqrt(var) * (loc * (2. * Phi - 1.) + 2 * phi - 1. / np.sqrt(np.pi))
    # Then we take the mean. The cost is now a scalar
    return K.mean(crps)
