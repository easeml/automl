<template>
    <div class="row">
        <div class="col-12">
            <div class="card-box">
                <h4 class="header-title">Manage Datasets</h4>

                <table class="table table-hover m-0 tickets-list table-actions-bar dt-responsive nowrap" cellspacing="0" width="100%" id="datatable">
                <thead>
                <tr>
                    <table-field item-title="ID" item-value="ID" class="mainField"></table-field>
                    <table-field item-title="Name" item-value="Name" class="mainField"></table-field>
                    <table-field item-title="Owner" item-value="Owner" class="mainField"></table-field>
                    <table-field item-title="Source" item-value="Source" class="mainField"></table-field>
                    <table-field item-title="Source Address" item-value="Source Address" class="mainField"></table-field>
                    <table-field item-title="Creation Time" item-value="Creation Time" class="mainField"></table-field>
                    <table-field item-title="Status" item-value="Status" class="mainField"></table-field>
                    <table-field item-title="Action" item-value="Action" class="hidden-sm mainField"></table-field>
                </tr>
                </thead>

                <tbody>
                    <tr  v-for="item in items" :key="item.id">
                        <table-field :item-title=item.id :item-value=item.id class="mainField"></table-field>
                        <table-field :item-title=item.name :item-value=item.name ></table-field>
                        <table-field :item-title=item.user :item-value=item.user ></table-field>
                        <table-field :item-title=item.source :item-value=item.source ></table-field>
                        <table-field :item-title=item.sourceAddress :item-value=item.sourceAddress ></table-field>
                        <table-field :item-title=item.creationTime.toLocaleString() :item-value=item.creationTime.toLocaleString() ></table-field>
                        <table-field :item-title=item.status :item-value=item.status ></table-field>
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
import TableField from "@/components/TableField.vue";

export default {
    data() {
        return {
            items: []
        };
    },
    components: {
        TableField
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
</style>
