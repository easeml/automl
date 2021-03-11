import random
import os
import operator
import numpy as np
import numpy.random
import netCDF4 as nc
import xarray as xr
import re #@livios: regular expression module
import pickle as pkl
from collections import defaultdict
from math import radians, cos, sin, asin, sqrt
from torch.utils.data import Dataset
from datetime import datetime, timedelta
from time import time, strftime, gmtime

# this dictionary can be used to get the days that have already passed for a certain month in year
passed_days_per_month_dict = {
    1 : 0,
    2 : 31,
    3 : 59,
    4 : 90,
    5 : 120,
    6 : 151,
    7 : 181,
    8 : 212,
    9 : 243,
    10 : 273,
    11 : 304,
    12 : 334
}

# split list l into chunks of size n from https://chrisalbon.com/python/data_wrangling/break_list_into_chunks_of_equal_size/
def chunks(l, n):
    # For item i in a range that is a length of l,
    for i in range(0, len(l), n):
        # Create an index range for l of n items:
        yield l[i:i+n]

# transform filenames of type YYMMDDHH -> time
def getTimeFromFileName(file_names, lead=1):
    times = []
    for file_name in file_names:
        year = file_name[:2]
        month = file_name[2:4]
        day = file_name[4:6]
        hour = file_name[6:8]
        # by loading the observation data again for the init time, we need to add the lead time for the prediction
        dt = datetime(year=2000+int(year), month=int(month), day=int(day), hour=int(hour)) + timedelta(hours=lead)
        times += [np.datetime64(dt)]
    return times

# calculate the time [0,3,...,21] for which the label stands for
def getInversedHour(time_data, lead_idx):
    inversed_hour = []
    for l in lead_idx:
        x = np.arccos(time_data[l][0])
        if time_data[l][1] < 0:
            x = -x
            inversed_hour += [24 + int((x / (2 * np.pi)) * 24)]
        else:
            inversed_hour += [int((x / (2 * np.pi)) * 24)]
    return inversed_hour


# calculate the month [1,...,12] for which the label stands for
def getInversedMonth(time_data, lead_idx):
    inversed_month = []
    for l in lead_idx:
        x = np.arccos(time_data[l][2])
        if time_data[l][3] < 0:
            x = -x
            inversed_month += [12 + int((x / (2 * np.pi)) * 12) + 1]
        else:
            inversed_month += [int((x / (2 * np.pi)) * 12) + 1]
    return inversed_month


# calculates all raw COSMO-1 files, that have not yet been preprocessed,
# according all the preprocessed files in DESTINATION
def get_all_files(src_path, stations, inits):
    print("DataUtils.py/get_all_files")
    all_files = []

    # get all files from all the station folder if inis is None. In this case we are interested in the per
    # station and init time preprocessed single files.
    if inits is None:
        for s in stations:
            all_files += [(str(s), f[:-4]) for f in os.listdir(src_path + '/station_init/grid_size_9/' + 'Station_' + str(s))]
    # generate files according to all stations and init times that were found in the data set
    else:
        # the source path is not appended to reduce redundancy
        for s in stations:
            all_files += [(s, init) for init in inits]

    # only keep one file for each day and initialization time
    all_files = set(all_files)

    return all_files

# find parameter to search for current file
def getCosmoeParameter(ListParam, filename):
    for param in ListParam:
        if param in filename: return param
    return 0
    
# create temprary folders for each station
def createTempStationFolders(DESTINATION, station_ids):
    print("createTempStationFolders")
    for S in station_ids:
        temp_output_path = DESTINATION + '/temp/station_%s' % S
        if not os.path.exists(temp_output_path):
            os.makedirs(temp_output_path)

# get all file paths for the stations under the given source path
def getFilesToProcess(ADDRESSdata, DESTINATION, ListParam, Preprocessing, DateBegin, DateEnd):
    print("DataUtils.py/getFilesToProcess")
    
    # get all COSMO-E filenames, one file per predicted variable and initialization, ATTENTION: multiple naming styles
    # file name 2: i.e. cosmo-e_refcst_2017081200_CLCH.nc
    all_cosmoe_refcst_files = [(D[15:25], D) for D in os.listdir(ADDRESSdata) if D[0:14] == 'cosmo-e_refcst']
    # file name 3: i.e. cosmo-e_pp_veri_2017081200_CLCH.nc
    all_cosmoe_pp_veri_files = [(D[16:26],D) for D in os.listdir(ADDRESSdata) if D[0:10] == 'cosmo-e_pp']
    # file name 1: i.e. cosmo-e_2017081200_CLCH.nc
    all_cosmoe_files = [(D[8:18], D) for D in os.listdir(ADDRESSdata) if D[0:9] == 'cosmo-e_2']

    # store filenames in dictionary per date and initialisation
    folders = defaultdict(list)
    for f in (sorted(all_cosmoe_files) + sorted(all_cosmoe_refcst_files) + sorted(all_cosmoe_pp_veri_files)):
        if len(folders[f[0]]) != 11: # needed since multiple predictions for same date!
            if getCosmoeParameter(ListParam,f[1]) == 0: 
                continue
            folders[f[0]].append(f[1])
    
    folders_keys = [f for f in folders.keys()]

    return folders
    
    # TODO: Rest of method has to be checked when some preprocessing has been done
    
    # for all preprocessed station files, take the intersection to only
    # keep files that are preprocessed for all the stations
    intersection = None
    
    regex = r'^station_([0-9]+?)_data.nc$'
    all_station_files = [f for f in os.listdir(DESTINATION) if re.match(regex, f)]

    # for all preprocessed station files, take the intersection to only
    # keep files that are preprocessed for all the stations
    for file in all_station_files:
        # store station error data
        ds = xr.open_dataset(DESTINATION + '/' + file)
        if intersection is None:
            intersection = set(ds.coords['init'].data)
        else:
            intersection = intersection.intersection(set(ds.coords['init'].data))

    # return all files that yet have not been processed
    if intersection is None:
        return folders
    # filter dictionary by intersection
    filterByKey = lambda keys: {x: folders[x] for x in keys}
    return filterByKey(intersection)


# splits the data into K-folds for cross validation, where granularity of data samples is a consecutive section of length L
# with stations it can be determined what stations should be included
def split_data_into_section(src_path, stations, inits, K, L, seed):
    print("DataUtils.py/split_data_into_section")

    # get all folders found in source folder
    all_files = get_all_files(src_path=src_path, stations=stations, inits=inits)
    # get all distinct datetime stamps
    datetimes = list(set(map(operator.itemgetter(1), all_files)))
    datetimes.sort()
    # calculate minimal L to generate at least K splits
    L=min(L,int(len(datetimes)/K))
    # split all files into lists of size L
    datetime_splits = [datetimes[i:i + L] for i in range(0, len(datetimes), L)]

    # initialize random seed to generate always the same data splits
    random.seed(seed)

    random.shuffle(datetime_splits)
    # map each split element efficiently to one of the K folds
    datetime_to_fold_dict = {}
    for idx, split in enumerate(datetime_splits):
        fold_number = idx % K
        for elem in split:
            datetime_to_fold_dict.update({elem: fold_number})

    folds = [[] for _ in range(K)]
    # put all elements to correct list
    for file in all_files:
        folds[datetime_to_fold_dict[file[1]]] += [file]

    return folds


# this method filters items a test set, that are closer than W (window) to any test point. With this, it can be
# ensured not to mix train/test data and therefore bias the result of a model. With the parameter "time_serie_length"
# we can additionally say, that a time serie of data points ending a a test point is never to close to any train point
# and vise-versa
def nearDuplicateFilter(trainset, testset, window, test_fraction, seed, time_series_length):
    print("DataUtils.py/nearDuplicateFilter")

    if window % 6 != 0:
        raise Exception('Window around test sample that is excluded should be a multiple of 6.')
    # for each test point, calculate a window of samples around this point to be excluded from training
    near_duplicates = []
    for _, test in testset:
        test_time = datetime.strptime(test[:8],'%y%m%d%H')
        # gather all date times, that are in a too close window around a test point
        near_duplicates += [(test_time + timedelta(hours=i-window//2)).strftime('%y%m%d%H') for i in range(0,window+1,3)]
        # gather all data times, that are in a too close window around a test point, when we want a data set of
        # train/test points as a time series of length "time_series_length". It is explicitly not
        # "time_series_length + 1" because the point at "window/2 + time_series_length" is valid again.
        near_duplicates += [(test_time + timedelta(hours=i+window//2).strftime('%y%m%d%H') for i in range(3, time_series_length, 3))]
    near_duplicates = set(near_duplicates)
    return [train_sample for train_sample in trainset if train_sample[1] not in near_duplicates], testset


# split the data into consecutive sections of length defined by slice size
def getDataFolds(config):
    # split data into train and test sets
    data_split_time = time()
    data_folds = split_data_into_section(src_path=config['input_source'], stations=config['stations'],
                                         inits=config['inits'], K=int(1 / config['test_fraction']),
                                         L=config['slice_size'], seed=config['seed'])
    config['time_data_split'] = time() - data_split_time
    print('[Time]: Splitting the data %s' % strftime("%H:%M:%S", gmtime(config['time_data_split'])))
    return data_folds


# loads the data statistics (min, max, mean, std) for each feature
# and adds the input parameters to the experiment configuration
def getDataStatistics(config):
    data_statistics = None
    # Load statistics from data set for feature scaling
    if config['is_normalization']:
        with open(config['input_source'] + "/feature_summary_grid_%s.pkl" % config['original_grid_size'], "rb") as input_file:
            feature_summary = pkl.load(input_file)
            # get numbers for normalization
            data_statistics = {
                'mean': feature_summary.sel(characteristic = 'mean').data,
                'var': feature_summary.sel(characteristic = 'var').data,
                'min': feature_summary.sel(characteristic = 'min').data,
                'max': feature_summary.sel(characteristic = 'max').data
            }
            config['input_parameters'] = list(feature_summary.coords['feature'].data)
    return data_statistics

# generates for each run the near duplicate filtered train and test folds
def getTrainTestFolds(config, data_folds):
    print("DataUtils.py/getTrainTestFolds")

    train_test_folds = []
    config['train_test_distribution'] = []
    for run in range(config['runs']):
        # select train and test folds
        train_fold = [item for sublist in data_folds[:run] + data_folds[run + 1:] for item in sublist]
        test_fold = data_folds[run]

        # store length of train/test fold before eliminating near duplicates
        n_orginal_train_samples = len(train_fold)
        n_orginal_test_samples = len(test_fold)

        # filter out all train samples that are to close to a test sample
        train_fold, test_fold = nearDuplicateFilter(trainset=train_fold, testset=test_fold,
                                                    window=config['test_filter_window'],
                                                    test_fraction=config['test_fraction'], seed=config['seed'],
                                                    time_series_length= config['time_serie_length'] if 'time_serie_length' in config else 0)

        # store length of train/test fold after eliminating near duplicates
        n_train_samples = len(train_fold)
        n_test_samples = len(test_fold)

        # update train / test sample size for run
        config['train_test_distribution'] += [(n_orginal_train_samples, n_train_samples,
                                                        n_orginal_test_samples, n_test_samples)]

        train_test_folds += [(train_fold, test_fold)]

    with open(config['input_source'] + '/train_test_folds_r_%s_sl_%s_tfw_%s_tf_%s_series_%s_s_%s.pkl' % (config['runs'],config['slice_size'],
                                                                                        config['test_filter_window'],
                                                                                        config['test_fraction'],
                                                                                        config['time_serie_length'] if 'time_serie_length' in config else 0,
                                                                                        config['seed']), 'wb') as f:
        pkl.dump(obj=train_test_folds, file=f)

    return train_test_folds

def filterUnseenTestStations(train_test_folds, config):
    # get number of test stations
    try:
        n_test_stations = config['n_test_stations']
    except KeyError:
        n_test_stations = 5

    # config has to specify from which sation on we select randomly n_test_stations
    first_test_staiton = config['first_test_station']

    # initialize random seed to generate always the same data splits
    seed = 23021
    random.seed(seed)
    all_stations = config['stations']
    random.shuffle(all_stations)
    test_stations = all_stations[first_test_staiton:first_test_staiton+n_test_stations]
    train_stations = [s for s in all_stations if s not in test_stations]
    config['train_stations'] = train_stations
    config['test_stations'] = test_stations

    # filter train and test splits
    temp_train_test_folds = []
    for train_fold, test_fold in train_test_folds:
        train_test_fold = ([item for item in train_fold if item[0] not in test_stations],
                           [item for item in test_fold if item[0] not in train_stations])

        # assert that the splits are station-wise distinct
        assert len(set(train_test_fold[0]).intersection(set(train_test_fold[1]))) == 0
        temp_train_test_folds += [train_test_fold]
    return temp_train_test_folds


def normalizationValues(ADDR, FOLDERS, VARS, WithTopo):
    print("DataUtils.py/normalizationValues")

    # only a subset of all cosmo-1 outputs are samplet to generate approximated measurements
    folder_samples = min(10, len(FOLDERS))
    point_samples = 10
    total_samples = folder_samples * point_samples

    MEAN = np.array([0] * len(VARS), dtype='d')
    STD = np.array([0] * len(VARS), dtype='d')
    MAX = np.array([0] * len(VARS), dtype='d')
    MIN = np.array([np.inf] * len(VARS), dtype='d')

    selected_folders = random.sample(FOLDERS, folder_samples)
    for n in range(folder_samples):
        F = selected_folders[n]
        dataset = nc.Dataset(ADDR + '/' + F + '/c1ffsurf000.nc')
        for i in range(len(VARS)):
            selected_points = np.random.choice(dataset[VARS[i]][:].flatten(), size=point_samples, replace=False)
            MAX[i] = np.max((MAX[i], np.max(selected_points)))
            MIN[i] = np.min((MIN[i], np.min(selected_points)))
            MEAN[i] += np.sum(selected_points)

    MEAN *= 1 / total_samples

    for n in range(folder_samples):
        F = selected_folders[n]
        dataset = nc.Dataset(ADDR + '/' + F + '/c1ffsurf000.nc')
        for i in range(len(VARS)):
            selected_points = np.random.choice(dataset[VARS[i]][:].flatten(), size=point_samples, replace=False)
            STD[i] += np.sum((selected_points - MEAN[i]) ** 2)
    STD = np.sqrt(STD / total_samples)

    if WithTopo:
        MEAN = np.append([530, 0], MEAN)
        STD = np.append([500, 1], STD)

    return MEAN, STD, MAX, MIN

# standardization of data by mean and standard deviation
def standardize(data, mean, std):
    data -= mean
    data = np.divide(data, std)
    return data

# normalization of data approximately between [0,1]
def normalize(data, min, max):

    data -= min
    data = np.divide(data, max-min)

    if np.max(data) > 1.01 or np.min(data) < -0.01:
        raise Exception('Problem im normalization, some features are out of the [0,1] range.')
    return data

def normalizeTimeFeatures(data):
    return np.vectorize(lambda x: (x+1)/2)(data)

# normalize latitude [42.8, 49.8] to [0,1]
def normalizeLatitude(data):
    return normalize(data, 42.8, 49.8)

# normalize longitude [0.3, 16.6] to [0,1]
def normalizeLongitude(data):
    return normalize(data, 0.3, 16.6)

# normalize height grid:[-5.4, 4267.5] and stations:[203.2, 3580] in to [0,1]
def normalizeHeight(data):
    return normalize(data, -5.4, 4267.5)

# normalize soil type [1:9] in to [0:1] (discrete)
def normalizeSoilType(data):
    return normalize(data, 1, 9)

def normalizeDiffFeature(data):
    minimum, maximum = data.min(), data.max()
    return np.vectorize(lambda x: normalize(x, minimum, maximum))(data)

# do not transform the data
def identity(data):
    return data

# returns a list of transformation functions for each parameters data
# param_normalizaiton: dict, e.g. param -> feature_transformation
# statistics: dict, e.g. list of values of 'mean', 'std', 'min', 'max' for each feature
def getFeatureScaleFunctions(param_normalization, statistics=None):
    n_params = len(param_normalization.keys())
    # if no statistic is given, we return an identity function
    scale_functions = [lambda x : identity(x)] * n_params
    if statistics != None:
        for idx, p in enumerate(param_normalization.keys()):
            if param_normalization[p] == 'n':
                scale_functions[idx] = lambda x, idx=idx : normalize(x, statistics['min'][idx], statistics['max'][idx])
            elif param_normalization[p] == 's':
                scale_functions[idx] = lambda x, idx=idx: standardize(x, statistics['mean'][idx], statistics['var'][idx])
            else:
                pass
    return scale_functions

# construct data set with time invariant features per station (-grid)
def getTimeInvariantStationFeatures(TOPO, OBS, stationSquares, stationIds, closestGridPointPerStation, GridSize, Features):
    # TOPO: topological data (netCDF)
    # OBS: observation data (netCDF)
    # stationSquares: data frame with grid point squares around each station
    # stationIds: the id's uniquely identifying the stations (not continuous)
    # closestGridPointPerStation: (lat, lon) of closest grid point for each station
    # GridSize: size of grid point squares around each station
    # Features: all time invariant features that have to be preprocessed

    n_stations = len(stationIds)
    n_features = len(Features)

    grid_indices = list(range(GridSize))

    # container for all features, for all station grids
    DATA = np.zeros((n_stations, GridSize, GridSize, n_features))
    # container for all positional features, for all stations
    station_position_data = np.zeros((n_stations, 8))
    closest_grid_points = []

    # calculate all features per station
    for idx_S, S in enumerate(stationIds):
        stationSquare = stationSquares[S]
        lat_idx = stationSquare.lat_idx
        lon_idx = stationSquare.lon_idx
        stationTopo = TOPO.isel(rlat = lat_idx, rlon = lon_idx)

        # height features
        DATA[idx_S,:,:,Features.index('HH')] = np.vectorize(lambda x: normalizeHeight(x))(stationTopo['HH'])

        # calculate height difference between grid points and station
        station_height = OBS['height'].sel(station_id=S).data
        DATA[idx_S, :, :, Features.index('HH_DIFF')] = stationTopo['HH'] - station_height

        # fraction of land feature
        DATA[idx_S, :, :, Features.index('FR_LAND')] = stationTopo['FR_LAND']

        # soil type feature
        DATA[idx_S, :, :, Features.index('SOILTYP')] = np.vectorize(lambda x: normalizeSoilType(x))(
                                                           stationTopo['SOILTYP'])

        # latitiude features
        DATA[idx_S, :, :, Features.index('LAT')] = np.vectorize(lambda x: normalizeLatitude(x))(
                                                                stationTopo['lat'])
        station_lat = OBS['lat'].sel(station_id = S).data
        DATA[idx_S, :, :, Features.index('LAT_DIFF')] = stationTopo['lat'] - station_lat
        DATA[idx_S, :, :, Features.index('RLAT')] = stationTopo['rlat']

        # longitued features
        DATA[idx_S, :, :, Features.index('LON')] = np.vectorize(lambda x: normalizeLongitude(x))(
                                                                stationTopo['lon'])
        station_lon = OBS['lon'].sel(station_id = S).data
        DATA[idx_S, :, :, Features.index('LON_DIFF')] = stationTopo['lon'] - station_lon
        DATA[idx_S, :, :, Features.index('RLON')] = stationTopo['rlon']

        # calculate horizontal distance in meters
        grid_lat_lon_zip = np.array(list(zip(
            stationTopo['lat'].data.ravel(),
            stationTopo['lon'].data.ravel())),dtype=('float32,float32'))\
            .reshape(DATA[idx_S, :, :, Features.index('LAT')].shape)
        DATA[idx_S, :, :, Features.index('ABS_2D_DIST')] = np.vectorize(
            lambda lat_lon_zip:haversine(lat_lon_zip[0], lat_lon_zip[1], station_lat, station_lon))(grid_lat_lon_zip)
        DATA[idx_S, :, :, Features.index('ABS_2D_DIST_RAW')] = np.vectorize(
            lambda lat_lon_zip: haversine(lat_lon_zip[0], lat_lon_zip[1], station_lat, station_lon))(grid_lat_lon_zip)

        # calculate grid point position in small grid of closest point with regard to horizontal and vertical distance
        # and map this position back to the position on the large grid (674, 1058). Additionally concatenate the
        # purely horizontal closest point
        closest_grid_point_2d_global = closestGridPointPerStation[idx_S]
        closest_grid_point_2d = (list(stationSquare.lat_idx).index(closest_grid_point_2d_global[0]),list(stationSquare.lon_idx).index(closest_grid_point_2d_global[1]))
        closest_grid_point_3d = np.unravel_index((DATA[idx_S, :, :, Features.index('ABS_2D_DIST_RAW')]\
                                  + 500 * np.abs(DATA[idx_S, :, :, Features.index('HH_DIFF')])).argmin(),(GridSize,GridSize))
        closest_grid_point_3d_global = (stationSquare.lat_idx[closest_grid_point_3d[0]],
                                   stationSquare.lon_idx[closest_grid_point_3d[1]])
        closest_grid_points += [closest_grid_point_2d + closest_grid_point_3d
                                + closest_grid_point_2d_global + closest_grid_point_3d_global]
        # fill station position data
        station_position_data[idx_S, 0] = normalizeHeight(station_height)
        station_position_data[idx_S, 1] = station_height
        station_position_data[idx_S, 2] = normalizeLatitude(station_lat)
        station_position_data[idx_S, 3] = station_lat
        station_position_data[idx_S, 4] = normalizeLongitude(station_lon)
        station_position_data[idx_S, 5] = station_lon
        station_position_data[idx_S, 6] = OBS['rlat'].sel(station_id = S).data
        station_position_data[idx_S, 7] = OBS['rlon'].sel(station_id = S).data

    # normalize height difference
    DATA[:, :, :, Features.index('HH_DIFF')] = normalizeDiffFeature(DATA[:, :, :, Features.index('HH_DIFF')])
    # normalize height difference
    DATA[:, :, :, Features.index('LAT_DIFF')] = normalizeDiffFeature(DATA[:, :, :, Features.index('LAT_DIFF')])
    # normalize height difference
    DATA[:, :, :, Features.index('LON_DIFF')] = normalizeDiffFeature(DATA[:, :, :, Features.index('LON_DIFF')])
    # normalize horizontal distance
    DATA[:, :, :, Features.index('ABS_2D_DIST')] = normalizeDiffFeature(DATA[:, :, :, Features.index('ABS_2D_DIST')])

    # generate data array with station dependent grid point features
    grid_data = xr.DataArray(DATA,
                             dims=('station', 'idx_lat', 'idx_lon', 'feature'),
                             coords=[stationIds, grid_indices, grid_indices, Features])

    # generate data array  with closest grid point coordinates for each station
    closest_grid_point_idx = xr.DataArray(np.array(list(zip(*closest_grid_points))).T,
                                          dims=('station', 'direction'),
                                          coords=[stationIds, ['lat_2d', 'lon_2d', 'lat_3d', 'lon_3d', 'lat_2d_global', 'lon_2d_global', 'lat_3d_global', 'lon_3d_global']])

    # generate data array with "pretty" name per station
    # station_name = xr.DataArray(np.char.decode(OBS['name'].data),
    station_name = xr.DataArray(OBS['name'].data,
                                dims = ('station'),
                                coords = [stationIds])

    # generate data array with positional information of each station
    station_position = xr.DataArray(station_position_data,
                                    dims = ('station', 'positinal_attribute'),
                                    coords = [stationIds, ['height', 'height_raw',
                                                       'lat', 'lat_raw',
                                                       'lon', 'lon_raw',
                                                       'rlat', 'rlon']])

    # return data set generated out of data arrays
    return xr.Dataset({'grid_data': grid_data,
                     'closest_grid_point' : closest_grid_point_idx,
                     'station_name' : station_name,
                     'station_position' : station_position})


# source: https://stackoverflow.com/questions/15736995/how-can-i-quickly-estimate-the-distance-between-two-latitude-longitude-points
# date: 11.04.2018
def haversine(lat1, lon1, lat2, lon2):
    """
    Calculate the great circle distance between two points
    on the earth (specified in decimal degrees)
    """
    # convert decimal degrees to radians
    lon1, lat1, lon2, lat2 = map(radians, [lon1, lat1, lon2, lat2])
    # haversine formula
    dlon = lon2 - lon1
    dlat = lat2 - lat1
    a = sin(dlat / 2) ** 2 + cos(lat1) * cos(lat2) * sin(dlon / 2) ** 2
    c = 2 * asin(sqrt(a))
    # Radius of earth in kilometers is 6371
    m = 6371* 1000 * c
    return m