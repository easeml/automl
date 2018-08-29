<template>
    <div class="row">
        <div class="col-12">
            <div class="card-box">
                <h4 class="header-title">Manage Datasets</h4>

                <table class="table table-hover m-0 tickets-list table-actions-bar dt-responsive nowrap" cellspacing="0" width="100%" id="datatable">
                <thead>
                <tr>
                    <th>
                        ID
                    </th>
                    <th>Name</th>
                    <th>Owner</th>
                    <th>Source</th>
                    <th>Source Address</th>
                    <th>Creation Time</th>
                    <th>Status</th>
                    <th class="hidden-sm">Action</th>
                </tr>
                </thead>

                <tbody>
                    <tr v-for="item in items" :key="item.id">

                        <td><b>{{item.id}}</b></td>

                        <td>{{item.name}}</td>

                        <td>{{item.user}}</td>

                        <td>{{item.source}}</td>

                        <td>{{item.sourceAddress}}</td>

                        <td>{{ item.creationTime.toLocaleString() }}</td>

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
import client from "@/client/index"

export default {
    data() {
        return {
            items: []
        };
    },
    methods: {
        loadData: function() {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));

            context.getDatasets()
            .then(data => {
                this.items = data;
            })
            .catch(e => console.log(e));
        }
    },
    mounted() {

        this.loadData();
        
        // Repeat call every 10 seconds.
        this.timer = setInterval(function() {
            this.loadData();
        }.bind(this), 10000);
    },
    beforeDestroy() {
        clearInterval(this.timer);
    }
};
</script>
<style>
</style>
