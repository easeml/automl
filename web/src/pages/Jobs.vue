<template>
    <div class="row">
        <div class="col-12">
            <div class="card-box">
                <h4 class="header-title">Manage Jobs</h4>

                <table class="table table-hover m-0 tickets-list table-actions-bar dt-responsive nowrap" cellspacing="0" width="100%" id="datatable">
                <thead>
                <tr>
                    <th>
                        ID
                    </th>
                    <th>User</th>
                    <th>Dataset</th>
                    <th>Number of Models</th>
                    <th>Objective</th>
                    <th>Task Limit</th>
                    <th>Creation Time</th>
                    <th>Running Time</th>
                    <th>Status</th>
                    <th class="hidden-sm">Action</th>
                </tr>
                </thead>

                <tbody>
                    <tr v-for="item in items" :key="item.id">

                        <td><b><a :href="item.link">{{item.id}} </a></b></td>

                        <td>{{item.user}}</td>

                        <td>{{item.dataset}}</td>

                        <td>{{item.models.length}}</td>

                        <td>{{item.objective}}</td>

                        <td>{{item.maxTasks}}</td>

                        <td>{{ item.creationTimeString }}</td>

                        <td>{{ item.runningDurationString }}</td>

                        <td>{{item.status}}</td>

                        <td></td>

                    </tr>
                </tbody>
                </table>
            </div>
        </div>
    </div>
</template>
<script>
import client from "easemlclient"

export default {
    data() {
        return {
            items: []
        };
    },
    methods: {
        loadData: function() {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));

            context.getJobs()
            .then(data => {
                this.items = data;
            })
            .catch(e => console.log(e));
            
        }
    },
    mounted() {

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
