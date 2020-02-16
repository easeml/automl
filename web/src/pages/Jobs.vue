<template>
    <div class="row">
        <div class="col-12">
            <div class="card-box">
                <h4 class="header-title">Manage Jobs</h4>

                <table class="table table-hover m-0 tickets-list table-actions-bar dt-responsive nowrap" cellspacing="0" width="100%" id="datatable">
                <thead>
                <tr>
                    <table-field item-title="ID" item-value="ID" class="mainField"></table-field>
                    <table-field item-title="User" item-value="User" class="mainField"></table-field>
                    <table-field item-title="Dataset" item-value="Dataset" class="mainField"></table-field>
                    <table-field item-title="Number of Models" item-value="Number of Models" class="mainField"></table-field>
                    <table-field item-title="Objective" item-value="Objective" class="mainField"></table-field>
                    <table-field item-title="Task Limit" item-value="Task Limit" class="mainField"></table-field>
                    <table-field item-title="Creation Time" item-value="Creation Time" class="mainField"></table-field>
                    <table-field item-title="Running Time" item-value="Running Time" class="mainField"></table-field>
                    <table-field item-title="Status" item-value="Status" class="mainField"></table-field>
                </tr>
                </thead>

                <tbody>
                    <tr v-for="item in items" :key="item.id">
                        <td><b><router-link :to="'/jobs/'+item.id">{{item.id}} </router-link></b></td>
                        <table-field :item-title=item.user :item-value=item.user ></table-field>
                        <table-field :item-title=item.dataset :item-value=item.dataset ></table-field>
                        <table-field :item-title=item.models.length :item-value=item.models.length ></table-field>
                        <table-field :item-title=item.objective :item-value=item.objective ></table-field>
                        <table-field :item-title=item.maxTasks :item-value=item.maxTasks ></table-field>
                        <table-field :item-title=item.creationTimeString :item-value=item.creationTimeString ></table-field>
                        <table-field :item-title=item.runningDurationString :item-value=item.runningDurationString ></table-field>
                        <table-field :item-title=item.status :item-value=item.status ></table-field>
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
