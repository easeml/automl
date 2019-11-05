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
                    <tr  v-for="item in items" :key="item.id">


                        <td><b>
                            <span class="hinfo" data-toggle="tooltip" :title=item.name >
                               {{item.id}}
                            </span>
                        </b></td>

                        <td><span class="hinfo" data-toggle="tooltip" :title=item.name >
                            {{item.name}}
                        </span></td>

                        <td><span class="hinfo" data-toggle="tooltip" :title=item.user >
                            {{item.user}}
                        </span></td>

                        <td><span class="hinfo" data-toggle="tooltip" :title=item.source >
                            {{item.source}}
                       </span></td>

                        <td><span class="hinfo" data-toggle="tooltip" :title=item.sourceAddress >
                            {{item.sourceAddress}}
                        </span></td>

                        <td><span class="hinfo" data-toggle="tooltip" :title=item.creationTime.toLocaleString() >
                            {{ item.creationTime.toLocaleString() }}
                        </span></td>

                        <td><span class="hinfo" data-toggle="tooltip" :title=item.status >
                            {{item.status}}
                        </span></td>

                        <td>
                            <button type="button" class="btn btn-icon waves-effect btn-light" v-show="item.status==='validated'" @click.prevent="downloadData(item.id)">
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

            context.getDatasets()
            .then(data => {
                this.items = data;
            })
            .catch(e => console.log(e));
        },
        downloadData: function(datasetId) {

            let context = client.loadContext(JSON.parse(localStorage.getItem("context")));

            context.downloadDatasetByPath(datasetId, ".tar", true)
        }
    },
    mounted() {

        this.loadData();
        // (function runForever(){
        //     this.loadData();
        //     setTimeout(runForever, 5000);
        // })()
        
        // Repeat call every 10 seconds.
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
    table {
        width: 100%;
        table-layout: fixed;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    td{
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    th{
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

</style>
