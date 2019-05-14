"use strict";

const ID_FORMAT = new RegExp("^[a-zA-Z0-9_-]+$");
const NAME_FORMAT = /^[a-zA-Z0-9!"#$%&'()*+,.\/:;<=>?@\[\] ^_`{|}~-]*$/;
const API_KEY_HEADER = "X-API-KEY";

// Do a cursor based request to get all the data from the server. Returns a promise with all results combined.
function runGetQuery(axiosInstance, url, query) {
    return new Promise((resolve, reject) => {
        axiosInstance.get(url, { params: query })
        .then(response => {

            let cursor = response.data.metadata["next-page-cursor"];

            if (cursor==="") {
                let data = response.data.data || [];
                resolve(data);
            } else {
                query["cursor"] = cursor;
                runGetQuery(axiosInstance, url, query)
                .then(data => {
                    resolve(response.data.data.concat(data));
                })
                .catch(e => {
                    reject(e);
                });
            }

        })
        .catch(e => {
            reject(e);
        });
    });
}

/* const instance = axios.create({
    baseURL: "http://localhost:8080/api/v1/",
    timeout: 1000,
    headers: {"X-API-KEY": "0328551c-0dac-4407-a73a-59218cf551b2"}
  });

runGetQuery(instance, "/tasks", {limit: 2})
.then(data => {
    console.log(data);
})
.catch(e => {
    console.log(e);
})
 */

export default {
    runGetQuery: runGetQuery,
    ID_FORMAT: ID_FORMAT,
    NAME_FORMAT: NAME_FORMAT,
    API_KEY_HEADER: API_KEY_HEADER
};