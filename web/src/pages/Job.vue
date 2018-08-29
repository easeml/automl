<template>
    <div class="row">
        <div class="col-12">

            <div class="row">
                <div class="col-sm-12">
                    <!-- meta -->
                    <div class="profile-user-box card-box bg-custom">
                        <div class="row">
                            <div class="col-sm-6">

                                <div class="media-body text-white">
                                <span class="pull-left mr-4"><i class="fa fa-cogs thumb-lg" style="font-size: 88px"></i></span>
                                    <!--<h4 class="mt-1 mb-1 font-18">{{job.id}}</h4>-->
                                    <span class="font-13 text-light">
                                    Status: <b>{{job.status}}</b> <br/>
                                    Runtime: <b>{{job.runningDurationString}}</b><br/>
                                    Dataset: <b>{{job.dataset}}</b><br/>
                                    Objective: <b>{{job.objective}}</b>
                                    </span>


                                </div>
                            </div>
                            <div class="col-sm-6">
                                <div class="text-right">
                                    <button type="button" class="btn btn-light waves-effect" :disabled="pauseDisabled" v-show="pauseShow" @click.prevent="pauseClick()">
                                        <i :class="pauseIcon"></i>&nbsp;&nbsp; {{pauseLabel}}
                                    </button>
                                    <button type="button" class="btn btn-light waves-effect" v-show="stopShow" @click.prevent="stopClick()">
                                        <i class="fa fa-stop-circle"></i>&nbsp;&nbsp; Stop
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                    <!--/ meta -->
                </div>
            </div>

            <div class="card-box">
                

                

                <h4 class="header-title">Job Tasks</h4>

                <table class="table table-hover m-0 tickets-list table-actions-bar dt-responsive nowrap" cellspacing="0" width="100%" id="datatable">
                <thead>
                <tr>
                    <th>
                        ID
                    </th>
                    <th>Model</th>
                    <th>Status</th>
                    <th>Stage</th>
                    <th>Running Time</th>
                    <th>Quality</th>
                    <th class="hidden-sm">Action</th>
                </tr>
                </thead>

                <tbody>
                    <tr v-for="item in items" :key="item.intId">

                        <td><b>{{item.intId}}</b></td>

                        <td>{{item.model}}</td>

                        <td>{{item.status}}</td>

                        <td>{{item.stage}}</td>

                        <td>{{ item.runningDurationString }}</td>

                        <td>{{item.quality}}</td>

                        <td>
                            <button type="button" class="btn btn-icon waves-effect btn-light">
                                <i class="fa fa-cloud-download"></i>
                            </button>
                        </td>

                    </tr>
                </tbody>
                </table>
            </div>
        </div>
    </div>
</template>
<script>
import client from "@/client/index"

export default {
    data() {
        return {
            items: [],
            jobId: "",
            job: {},
            jobModels: null
        };
    },
    computed: {
        pauseLabel() {
            if (this.job.status === "running" || this.job.status === "pausing") {
                return "Pause";
            } else if (this.job.status === "paused" || this.job.status === "resuming") {
                return "Resume";
            } else if (this.job.status === "scheduled") {
                return "Start";
            } else {
                return "[none]";
            }
        },
        pauseIcon() {
            if (this.job.status === "running" || this.job.status === "pausing") {
                return ["fa", "fa-pause-circle"];
            } else if (this.job.status === "paused" || this.job.status === "resuming") {
                return ["fa", "fa-play-circle"];
            } else if (this.job.status === "scheduled") {
                return ["fa", "fa-play-circle"];
            } else {
                return "[none]";
            }
        },
        pauseDisabled() {
            return this.job.status !== "running" && this.job.status !== "paused";
        },
        pauseShow() {
            return ["scheduled", "running", "paused", "pausing", "resuming"].includes(this.job.status);
        },
        stopShow() {
            return ["scheduled", "running", "paused", "pausing", "resuming"].includes(this.job.status);
        }
    },
    methods: {
        loadData: function() {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));

            context.getTasks({job: this.jobId, orderBy: "quality", order: "desc"})
            .then(data => {
                this.items = data;
            })
            .catch(e => console.log(e));

            context.getJobById(this.jobId)
            .then(data => {
                this.job = data;

                // Check if new models have been added to this job.
                if (this.jobModels && this.jobModels.length !== this.job.models.length) {
                    for (let i = 0; i < this.job.models.length; i++) {
                        if (this.jobModels.includes(this.job.models[i]) === false) {
                            // A new model was added to this job. Notify the user.
                            console.log(this.job.models[i]);
                            this.$notify({
                                group: "group",
                                title: "New Model Available",
                                text: "Model <b> \"" + this.job.models[i] + "\" </b> added to this job.",
                                duration: 10000,
                                position: "bottom right",
                                type: "warn"
                            });
                        }
                    }
                }

                this.jobModels = this.job.models;
            })
            .catch(e => console.log(e));

        },
        pauseClick() {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));

            let newStatus = null;
            if (this.job.status === "running") {
                newStatus = "pausing";
            } else if (this.job.status === "paused") {
                newStatus = "resuming";
            } else if (this.job.status === "scheduled") {
                newStatus = "running";
            }

            if (newStatus) {
                context.updateJob(this.jobId, {"status" : newStatus})
                this.job.status = newStatus;
            }

        },
        stopClick() {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));

            let newStatus = null;
            if (["scheduled", "running", "paused", "pausing"].includes(this.job.status)) {
                context.updateJob(this.jobId, {"status" : "terminating"})
                this.job.status = "terminating";
            }

        }
    },
    mounted() {

        this.jobId = this.$route.params.id;
        this.loadData();
        
        // Repeat call every 1 second.
        this.timer = setInterval(function() {
            this.loadData();
        }.bind(this), 5000);
    },
    beforeDestroy() {
        clearInterval(this.timer);
    }
};
</script>
<style>
</style>
