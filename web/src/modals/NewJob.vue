<template>
<modal name="new-job" transition="pop-out" height="550" width="1000" @before-open="beforeOpen">

        <button type="button" class="close"  @click.prevent="close()">
            <span>&times;</span><span class="sr-only">Close</span>
        </button>
        <h4 class="custom-modal-title">Start New Job</h4>
        <div class="custom-modal-text">
            <form class="form-horizontal" action="#">

                <div class="wiz-container">
                    <transition name="fade" mode="out-in">
                        <div class="wiz-step" v-show="step === 1">
                            <h4> Choose a Dataset </h4>
                            <div class="row">
                                <div class="col-4">
                                    <select size="10" class="form-control" v-model="selectedDataset">
                                        <option v-for="dataset in datasets" :key="dataset.id" :value="dataset">
                                            {{dataset.name || dataset.id}}
                                        </option>
                                    </select>
                                </div>

                                <div class="col">
                                    <h1>{{selectedDataset.name || selectedDataset.id}}</h1>
                                    <span v-html="selectedDatasetDescriptionHtml"></span>
                                </div>
                            </div>
                            
                        </div>
                    

                    </transition>

                    <transition name="fade" mode="out-in">

                        <div class="wiz-step" v-show="step === 2">
                            <h4> Choose an Objective </h4>

                            <div class="row">
                                <div class="col-4">
                                    <select size="10" class="form-control" v-model="selectedObjective">
                                        <option v-for="objective in objectives" :key="objective.id" :value="objective">
                                            {{objective.name || objective.id}}
                                        </option>
                                    </select>
                                </div>

                                <div class="col">
                                    <h1>{{selectedObjective.name || selectedObjective.id}}</h1>
                                    <span v-html="selectedObjectiveDescriptionHtml"></span>
                                </div>
                            </div>

                        </div>

                    </transition>

                    <transition name="fade" mode="out-in">

                        <div class="wiz-step" v-show="step === 3">
                            <h4> Choose Models </h4>

                            <div class="row">
                                <div class="col-4">
                                    <select multiple size="10" class="form-control" v-model="selectedModels">
                                        <option v-for="model in models" :key="model.id" :value="model">
                                            {{model.name || model.id}}
                                        </option>
                                    </select>
                                </div>

                                <div class="col">
                                    <h1>{{selectedModelName}}</h1>
                                    <span v-html="selectedModelDescriptionHtml"></span>
                                </div>
                            </div>
                        </div>

                    </transition>

                    <transition name="fade" mode="out-in">

                        <div class="wiz-step" v-show="step === 4">
                            <h4> Ready to Start </h4>

                            <div class="form-group row">
                                <label class="col-3 col-form-label">Dataset</label>
                                <div class="col-9">
                                    <input type="text" class="form-control" readonly="" :value="selectedDataset.name || selectedDataset.id">
                                </div>
                            </div>

                            <div class="form-group row">
                                <label class="col-3 col-form-label">Objective</label>
                                <div class="col-9">
                                    <input type="text" class="form-control" readonly="" :value="selectedObjective.name || selectedObjective.id">
                                </div>
                            </div>

                            <div class="form-group row">
                                <label class="col-3 col-form-label">Models</label>
                                <div class="col-9">
                                    <textarea rows="5" class="form-control" readonly="" style="resize:none;overflow-y: scroll" :value="selectedModelNames"></textarea>
                                </div>
                            </div>

                            <div class="form-group row align-items-center">
                                <label class="col-3 col-form-label">Maximum tasks</label>
                                <div class="col-7">
                                    <input type="text" class="form-control" v-model="maxTasks" :disabled="maxTasksUnlimited">
                                </div>
                                <div class="col-1 form-check checkbox checkbox-custom ml-2">
                                    <input id="chk-max-task-unlimited" type="checkbox" class="form-check-input" v-model="maxTasksUnlimited">
                                    <label for="chk-max-task-unlimited" class="form-check-label">Unlimited</label>
                                </div>
                            </div>

                            <div class="form-group row align-items-center">
                                <label class="col-3 col-form-label" for="chk-accept-models">When new models arrive</label>
                                <div class="col-9">
                                    <select class="form-control" v-model="acceptNewModels" >
                                        <option :value="false">Ignore</option>
                                        <option :value="true">Add to this job if applicable</option>
                                    </select>
                                </div>
                            </div>

                        </div>

                    </transition>
                </div>

                <div class="mt-1">
                    <div class="button-list wiz-buttons">
                        <button class="btn btn-custom waves-light waves-effect" v-show="prevVisible" @click.prevent="prev()">Previous</button>
                        <button class="btn btn-custom waves-light waves-effect" v-show="nextVisible" @click.prevent="next()">Next</button>
                        <button type="submit" class="btn btn-custom waves-light waves-effect" v-show="!nextVisible" @click.prevent="finish()">Finish</button>
                    </div>
                </div>

            </form>
        </div>

</modal>
</template>

<script>
import client from "easemlclient";
import showdown from "showdown";
var converter = new showdown.Converter();

export default {
    name: 'NewJobModal',
    data() {
        return {
            step: 1,
            datasets: [],
            selectedDataset: {},
            objectives: [],
            selectedObjective: "",
            models: [],
            selectedModels: [],
            maxTasks: 100,
            maxTasksUnlimited: false,
            acceptNewModels: true,
        }
    },
    computed: {
        prevVisible() {
            return this.step > 1;
        },
        nextVisible() {
            return this.step < 4;
        },
        selectedDatasetDescriptionHtml() {
            if (this.selectedDataset) {
                return converter.makeHtml(this.selectedDataset.description);
            } else {
                return ""
            }
        },
        selectedObjectiveDescriptionHtml() {
            if (this.selectedObjective) {
                return converter.makeHtml(this.selectedObjective.description);
            } else {
                return ""
            }
        },
        selectedModelNames() {
            return this.selectedModels.map(m => m.name || m.id).join("/n");
        },
        selectedModelName() {
            if (this.selectedModels.length == 1) {
                return this.selectedModels[0].name || this.selectedModels[0].id
            } else {
                return ""
            }
        },
        selectedModelDescriptionHtml() {
            if (this.selectedModels.length == 1) {
                return converter.makeHtml(this.selectedModels[0].description);
            } else {
                return ""
            }
        }
    },
    methods : {
        close() {
            this.$modal.hide("new-job");
        },
        prev() {
            if (this.step > 1) {
                this.step--;
                this.initStep();
            }
        },
        next() {
            if (this.step < 4) {
                this.step++;
                this.initStep();
            }
        },
        sucess(id) {
            this.$modal.hide("new-job");
            this.$swal({
                title: "Job Created",
                text: "Job ID: " + id,
                type: "success",
                showConfirmButton: false,
                timer: 3000
            });
            this.$router.push({ name: "job", params: { id: id }});
        },
        finish() {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));
            
            let job = {
                dataset: this.selectedDataset.id,
                objective: this.selectedObjective.id,
                models: this.selectedModels.map(x => x.id),
                acceptNewModels: this.acceptNewModels,
                maxTasks: this.maxTasksUnlimited ? 0 : this.maxTasks
            }
            context.createJob(job)
            .then(id => {
                this.sucess(id);
            })
            .catch(e => console.log(e));

            // Dispose of the modal.

            // Redirect to new job.

        },
        initStep() {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));
            
            if (this.step === 1) {
                // Dataset choose step.
                console

                // Get all datasets.
                context.getDatasets({status: "validated"})
                .then(data => {
                    this.datasets = data;

                    if (data.length > 0) {
                        this.selectedDataset = this.datasets[0];
                    }
                })
                .catch(e => console.log(e));


            } else if (this.step === 2) {
                // Objective choose step.

                // For the chosen dataset, find applicable objectives.
                context.getModules({type: "objective", status: "active", schemaIn: this.selectedDataset.schemaOut})
                .then(data => {
                    this.objectives = data;

                    if (data.length > 0) {
                        this.selectedObjective = this.objectives[0];
                    }
                })
                .catch(e => console.log(e));

            } else if (this.step === 3) {
                // Model choose step.

                // For the chosen dataset, find applicable models.
                context.getModules({
                    type: "model",
                    status: "active",
                    schemaIn: this.selectedDataset.schemaIn,
                    schemaOut: this.selectedDataset.schemaOut
                }).then(data => {
                    this.models = data;
                    this.selectedModels = this.models;
                })
                .catch(e => console.log(e));

            } else if (this.step === 4) {
                // Finalization step.

                // Specify max-tasks and whether new models should be added on the fly.

            }

        },
        beforeOpen() {
            this.step = 1;
            this.initStep();
        }
    }
};
</script>
<style scoped>
.fade-enter-active, .fade-leave-active {
  transition: opacity .3s;
}
.fade-enter, .fade-leave-to /* .fade-leave-active below version 2.1.8 */ {
  opacity: 0;
}
.wiz-container {
    position: relative;
    height: 420px;
}
.wiz-step {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
}
.wiz-buttons {
    float: right;
    vertical-align: bottom;
}
.close {
    margin: 15px;
}
</style>