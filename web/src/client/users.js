"use strict";

//let common = require("./common");
import common from "./common";
import decamelizeKeys from "decamelize-keys";


function transformDataItem(input) {
    return {
        id: input.id,
        name: input.name,
        status: input.status
    };
}

function getUsers(query) {
    
    // This allows us to accept camel case keys.
    query = decamelizeKeys(query || {}, "-");

    // Run query and collect results as a promise.
    return new Promise((resolve, reject) => {

        common.runGetQuery(this.axiosInstance, "/users", query)
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

function getUserById(id) {

    // Run query and collect results as a promise.
    return new Promise((resolve, reject) => {

        this.axiosInstance.get("/users/"+id)
        .then(response => {
            let result = transformDataItem(response.data.data);
            resolve(result);
        })
        .catch(e => {
            reject(e);
        });
    });

}

function loginUser() {

    // Run query and collect results as a promise.
    return new Promise((resolve, reject) => {

        this.axiosInstance.get("/users/login")
        .then(response => {
            
            this.userCredentials.apiKey = response.headers[common.API_KEY_HEADER];
            resolve(response);

        })
        .catch(e => {
            reject(e);
        });
    });

}

function logoutUser() {

    // Run query and collect results as a promise.
    return new Promise((resolve, reject) => {

        this.axiosInstance.get("/users/logout")
        .then(response => {
            this.userCredentials.apiKey = "";
            resolve();
        })
        .catch(e => {
            reject(e);
        });
    });

}


export default {
    getUsers: getUsers,
    getUserById: getUserById,
    loginUser: loginUser,
    logoutUser: logoutUser
};