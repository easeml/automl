<template>
<modal name="new-dataset" transition="pop-out" height="auto" width="1000" @before-open="beforeOpen">

        <button type="button" class="close" onclick="">
            <span>&times;</span><span class="sr-only">Close</span>
        </button>
        <h4 class="custom-modal-title">Add a New Dataset</h4>
        <div class="custom-modal-text">
            <form class="form-horizontal" action="#">

                <div :class="wizContainerClasses">

                    <transition name="fade" mode="out-in">
                        <div class="wiz-step" v-show="step === 1">
                            <h4> Choose a Dataset </h4>

                            <div class="form-group row">
                            
                                <label class="col-2 col-form-label">Source</label>
                                <div class="col-10">
                                    <select class="form-control" v-model="datasetSource" >
                                        <option value="upload">Upload from the browser</option>
                                        <option value="download">Download from a remote location</option>
                                        <option value="local">Copy from a local directory</option>
                                    </select>
                                </div>

                            </div>

                            <div v-if="datasetSource === 'upload'">
                                <file-dropzone @new-file="newDatasetInput"></file-dropzone>
                                <div class="text-danger" v-if="error">
                                    <p><i class="mdi mdi-close-circle"></i>&nbsp;{{error}}</p>
                                </div>
                            </div>

                            <div v-if="datasetSource === 'download'">
                                <div class="form-group row">
                                    <label class="col-2 col-form-label">Source URL</label>
                                    <div class="col-10">
                                        <input type="text" class="form-control" v-model="datasetSourceAddress">
                                        <span class="help-block">
                                            <small>Note: The URL must point to a downloadable *.zip, *.tar or *.tar.gz file.</small>
                                        </span>
                                    </div>
                                </div>
                            </div>

                            <div v-if="datasetSource === 'local'">
                                <div class="form-group row">
                                    <label class="col-2 col-form-label">Source path</label>
                                    <div class="col-10">
                                        <input type="text" class="form-control" v-model="datasetSourceAddress">
                                        <span class="help-block">
                                            <small>Note: This path must be accessible by the ease.ml controller service.</small>
                                        </span>
                                    </div>
                                </div>
                            </div>
                        
                        </div>
                    

                    </transition>

                    <transition name="fade" mode="out-in">

                        <div class="wiz-step" v-show="step === 2">
                            <h4> Uploaded Dataset Structure </h4>

                            <div class="form-group row">

                                <div class="col-6">
                                    <h5>Input Schema:</h5>
                                    <pre class="schemacode" v-html="datasetSchemaInDump"></pre>
                                    <!--<textarea rows="15" class="form-control" style="resize:none;overflow-y: scroll" :value="datasetSchemaInDump"></textarea>-->
                                </div>

                                <div class="col-6">
                                    <h5>Output Schema:</h5>
                                    <pre class="schemacode" v-html="datasetSchemaOutDump"></pre>
                                    <!--<textarea rows="15" class="form-control" style="resize:none;overflow-y: scroll" :value="datasetSchemaOutDump"></textarea>-->
                                </div>

                            </div>

                        </div>

                    </transition>

                    <transition name="fade" mode="out-in">

                        <div class="wiz-step" v-show="step === 3">
                            <h4> Specify Details </h4>

                            <div class="form-group row">
                                <label class="col-2 col-form-label">ID</label>
                                <div class="col-10">
                                    <div class="input-group">
                                        <div class="input-group-prepend">
                                            <span class="input-group-text">{{currentUserId}}/</span>
                                        </div>
                                        <input type="text" class="form-control" v-model="datasetId">
                                    </div>
                                </div>
                            </div>

                            <div class="form-group row">
                                <label class="col-2 col-form-label">Name</label>
                                <div class="col-10">
                                    <input type="text" class="form-control" v-model="datasetName">
                                </div>
                            </div>

                            <div class="form-group row">
                                <label class="col-2 col-form-label">Description</label>
                                <div class="col-10">
                                    <textarea rows="10" class="form-control" style="resize:none;overflow-y: scroll" v-model="datasetDescription"></textarea>
                                </div>
                            </div>
                            <div class="text-danger" v-if="error">
                                <p><i class="mdi mdi-close-circle"></i>&nbsp;{{error}}</p>
                            </div>
                        </div>

                    </transition>

                    <transition name="fade" mode="out-in">

                        <div class="wiz-step" v-show="step === 4">
                            <h3> Dataset Upload: {{currentUploadProgress}} % </h3>

                            <div class="progress progress-lg mb-0">
                                <div class="progress-bar progress-bar-warning progress-bar-striped progress-bar-animated"
                                role="progressbar" :aria-valuenow="currentUploadProgress" aria-valuemin="0" aria-valuemax="100"
                                :style="'width: '+currentUploadProgress+'%'"></div>
                            </div>

                            <div class="text-danger" v-if="error">
                                <p><i class="mdi mdi-close-circle"></i>&nbsp;{{error}}</p>
                            </div>
                        </div>

                    </transition>

                </div>
                
                <div class="mt-1">
                    <div class="button-list wiz-buttons">
                        <button class="btn btn-custom waves-light waves-effect" v-show="prevVisible" @click.prevent="prev()">Previous</button>
                        <button class="btn btn-custom waves-light waves-effect" v-show="nextVisible" :disabled="nextDisabled" @click.prevent="next()">Next</button>
                        <button type="submit" class="btn btn-custom waves-light waves-effect" v-show="finishVisible" @click.prevent="finish()">Finish</button>
                    </div>
                </div>

            </form>
        </div>

</modal>
</template>

<script>
import yaml from "js-yaml";
import vue2Dropzone from "vue2-dropzone";
import FileDropzone from "@/components/FileDropzone.vue";
import tarOpener from "@/schema/tar-opener";
import client from "@/client/index";
import showdown from "showdown";
var converter = new showdown.Converter();

import easemlSchema from "@/schema/src/index";

const NAME_FORMAT = /[a-zA-Z0-9 -]+/g;

function findReadmeAndScan(opener) {
    let name = "";
    let description = "";
    let rootFiles = opener("", "", true, true);

    for (let i = 0; i < rootFiles.length; i++) {
        if (rootFiles[i].startsWith("README")) {
            let readmeFile = opener("", rootFiles[i], false, true);
            let readmeFileLines = readmeFile.readLines();

            for (let j = 0; j < readmeFileLines.length; j++) {
                let line = readmeFileLines[j].trim();

                if (line) {
                    if (!name) {
                        // Check if match.
                        let match = line.match(NAME_FORMAT);
                        if (match) {
                            name = match.map(x=>x.trim()).join("");
                        }
                    } else {
                        // Append to description while skipping lines.
                        let trimmedLine = line.trim();
                        if (trimmedLine || (description && !description.endsWith("\n"))) {
                            // Trim whitespace on the right side.
                            line = line.replace(/\s+$/,"");
                            description += line + "\n";
                        }
                    }
                }
            }

        }
    }

    return {name, description};
}

export default {
    name: 'NewDatasetModal',
    components: {
        vueDropzone: vue2Dropzone,
        FileDropzone
    },
    data() {
        return {
            error: "",
            step: 1,
            datasetSource: "upload",
            datasetSourceAddress: "",
            datasetId: "",
            datasetName: "",
            datasetDescription: "",
            datasetRawData: null,
            datasetObject: null,
            datasetSchemaIn: null,
            datasetSchemaOut: null,
            datasetSchemaInDump: "",
            datasetSchemaOutDump: "",
            currentUserId: "",
            currentUploadProgress: 0,
        }
    },
    computed: {
        prevVisible() {
            return this.step > 1 && this.step < 4;
        },
        nextVisible() {
            return this.step < 3;
        },
        finishVisible() {
            return this.step === 3;
        },
        nextDisabled() {
            if (this.datasetSource === "upload") {
                return !(this.datasetSchemaInDump && this.datasetSchemaOutDump);
            } else {
                return !this.datasetSourceAddress;
            }
        },
        wizContainerClasses() {
            if (this.step < 4) {
                return ["wiz-container"]
            } else {
                return ["wiz-container wiz-container-short"]
            }
        }
    },
    methods : {
        prev() {
            this.switchStep(-1);
        },
        next() {
            this.switchStep(+1);
        },
        sucess() {
            this.$modal.hide("new-dataset");
            this.$swal({
                title: "Dataset Created",
                type: "success",
                showConfirmButton: false,
                timer: 3000
            });
        },
        finish() {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));
            
            let dataset = {
                id: this.currentUserId + "/" + this.datasetId,
                source: this.datasetSource,
                sourceAddress: this.datasetSourceAddress,
                name: this.datasetName,
                description: this.datasetDescription
            }
            context.createDataset(dataset)
            .then(id => {

                // Creation successful. Procede to upload if needed.
                if (this.datasetSource === "upload") {

                    // Switch to upload panel.
                    this.switchStep(+1);

                    context.uploadDataset(
                        dataset.id,
                        this.datasetRawData,
                        this.datasetRawData.name,
                        (bytesUploaded, bytesTotal) => {

                            this.currentUploadProgress = Math.round(100 * bytesUploaded / bytesTotal);
                            console.log(this.currentUploadProgress);

                    }).then(() => {

                        // Now we set the new dataset state to transferred.
                        context.updateDataset(dataset.id, {status: "transferred"})
                        .then(() => {
                            this.sucess();
                        }).catch(e => {
                            this.error = "Failed to update the dataset.";
                            console.log(e);
                        });
                        
                    }).catch(e => {
                        this.error = "Failed to upload the dataset.";
                        console.log(e);
                    });


                } else {
                    this.sucess();
                }

            })
            .catch(e => {
                this.error = "Dataset creation error.";
                console.log(e);
            });

        },
        switchStep(direction) {

            // If we are in the first or last steps, do nothing.
            if (direction < 0 && this.step <= 1) {
                return
            }
            if (direction > 0 && this.step >= 4) {
                return
            }
            this.step += direction;
            
            if (this.step === 1) {
                // Dataset choose step.
                this.datasetRawData = null;


            } else if (this.step === 2) {
                // Objective choose step.

                // We show this step only if the dataset is uploaded.
                if (this.datasetRawData) {

                } else {
                    console.log("no data uploaded");
                    this.switchStep(direction);
                }

            } else if (this.step === 3) {
                // Model choose step.

                // For the chosen dataset, find applicable models.
                
            }

        },
        beforeOpen() {
            this.step = 1;
            this.switchStep(0);

            // Get current user.
            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));
            context.getUserById("this")
            .then(result => {
                this.currentUserId = result.id;
            })
            .catch(e => console.log(e));
        },
        extractSchema(filestruct) {

            if ( !("train" in filestruct) || !("val" in filestruct) ) {
                this.error = "The dataset root must contain directories \"train\" and \"val\".";
                return false;
            }

            if ( !("input" in filestruct["train"]) || !("output" in filestruct["train"]) ) {
                this.error = "The dataset \"train\" directory must contain directories \"input\" and \"output\".";
                return false;
            }

            if ( !("input" in filestruct["val"]) || !("output" in filestruct["val"]) ) {
                this.error = "The dataset \"val\" directory must contain directories \"input\" and \"output\".";
                return false;
            }

            let openerTrainIn = tarOpener.new_opener(filestruct["train"]["input"]);
            let openerTrainOut = tarOpener.new_opener(filestruct["train"]["output"]);

            // Load datasets.
            try {
                var datasetTrainIn = easemlSchema.dataset.load("", openerTrainIn, true);
            } catch (err) {
                if (err instanceof easemlSchema.dataset.DatasetException) {
                    this.error = "Dataset load error: " + err.message + " @ /train/input" + err.path;
                } else {
                    this.error = "Dataset load error.";
                    console.log(err);
                }
                return false;
            }

            try {
                var datasetTrainOut = easemlSchema.dataset.load("", openerTrainOut, true);
            } catch (err) {
                if (err instanceof easemlSchema.dataset.DatasetException) {
                    this.error = "Dataset load error: " + err.message + " @ /train/output" + err.path;
                } else {
                    this.error = "Dataset load error.";
                    console.log(err);
                }
                return false;
            }

            // Infer schemas.
            try {
                this.datasetSchemaIn = datasetTrainIn.inferSchema();
            } catch (err) {
                if (err instanceof easemlSchema.schema.SchemaException) {
                    this.error = "Input schema inference error: " + err.message + " @ " + err.path;
                } else {
                    this.error = "Input schema inference error.";
                    console.log(err);
                }
                return false;
            }
            try {
                this.datasetSchemaOut = datasetTrainOut.inferSchema();
            } catch (err) {
                if (err instanceof easemlSchema.schema.SchemaException) {
                    this.error = "Output schema inference error: " + err.message + " @ " + err.path;
                } else {
                    this.error = "Output schema inference error.";
                    console.log(err);
                }
                return false;
            }

            // Dump JSON schemas.
            this.datasetSchemaInDump = yaml.safeDump(this.datasetSchemaIn.dump());
            this.datasetSchemaOutDump = yaml.safeDump(this.datasetSchemaOut.dump());

            return true;
        },
        linesToParagraphs(input) {
            return input.split("\n").map(text => "<p>"+text+"</p>").join("\n");
        },
        newDatasetInput(file) {

            this.datasetRawData = file;

            // Extract identifier candidate from name.
            const EXT_FORMAT = /\.[a-zA-Z0-9]+$/;
            this.datasetId = file.name.replace(EXT_FORMAT,"").replace(EXT_FORMAT,"");

            tarOpener.loadTarFile(file)
            .then(result => {
                console.log(result)

                // Check if the root contains a readme file.
                let opener = tarOpener.new_opener(result);
                
                // Try to find a README file and extract name and description.
                let r = findReadmeAndScan(opener);
                this.datasetName = r.name;
                this.datasetDescription = r.description;

                // Extract the schema.
                let success = this.extractSchema(result);

                if (success) {
                    this.switchStep(+1);
                }

            }).catch(e => console.log(e));
        }
    }
};
</script>
<style>
.fade-enter-active, .fade-leave-active {
  transition: opacity .5s;
}
.fade-enter, .fade-leave-to /* .fade-leave-active below version 2.1.8 */ {
  opacity: 0;
}
.wiz-container {
    position: relative;
    height: 420px;
}
.wiz-container-short {
    height: 200px;
}
.wiz-step {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
}
.wiz-buttons {
    /*float: right;*/
    text-align: right;
    vertical-align: bottom;
}
.close {
    margin: 15px;
}
.schemacode {
    border: 1px solid #e3eaef;
    padding: 10px;
    height: 320px;
    overflow-y: scroll;
}
.progress.progress-lg {
  height: 20px;
}
</style>