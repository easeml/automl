"use strict";

//let common = require("./common");
import common from "./common";
import decamelizeKeys from "decamelize-keys";

import moment from "moment";
let momentDurationFormatSetup = require("moment-duration-format"); 
momentDurationFormatSetup(moment);

function transformDataItem(input) {

    let runningDuration = moment.duration(input["running-duration"], "milliseconds");

    return {
        id: input.id,
        intId: parseInt(input.id.split("/")[1]),
        job: input.job,
        process: input.process,
        user: input.user,
        dataset: input.dataset,
        model: input.model,
        objective: input.objective,
        altObjectives: input["alt-objectives"],
        config: input.config,
        quality: input.quality,
        qualityTrain: input["quality-train"],
        qualityExpected: input["quality-expected"],
        altQualities: input["alt-qualities"],
        status: input.status,
        statusMessage: input["status-message"],
        stage: input.stage,
        runningDuration: runningDuration,
        runningDurationString: runningDuration.format()
    };
}

function getTasks(query) {
    
    // This allows us to accept camel case keys.
    query = decamelizeKeys(query || {}, "-");

    // Run query and collect results as a promise.
    return new Promise((resolve, reject) => {

        common.runGetQuery(this.axiosInstance, "/tasks", query)
        .then(data => {

            let items = [];

            if (data) {
                for (let i = 0; i < data.length; i++) {
                    items.push(transformDataItem(data[i]));
                }
            }

            resolve(items);

        })
        .catch(e => {
            reject(e);
        });
    });
}

function getTaskById(id) {

    // Run query and collect results as a promise.
    return new Promise((resolve, reject) => {

        this.axiosInstance.get("/tasks/"+id)
        .then(response => {
            let result = transformDataItem(response.data.data);
            resolve(result);
        })
        .catch(e => {
            reject(e);
        });
    });

}

function updateTask(id, updates) {
    // Collect fields of interest.
    let input = decamelizeKeys(input, "-");
    let data = {};
    if ("status" in updates) {
        data["status"] = updates["status"];
    }

    // Run patch request as a promise.
    return new Promise((resolve, reject) => {
        this.axiosInstance.patch("/tasks/"+id, data)
        .then(result => {
            resolve();
        })
        .catch(e => {
            reject(e);
        });
    });
}

export default {
    getTasks: getTasks,
    getTaskById: getTaskById,
    updateTask: updateTask
};
