<template>
<div class="left side-menu">

    <div class="slimscroll-menu" id="remove-scroll">

        <!-- LOGO -->
        <div class="topbar-left">
            <a href="index.html" class="logo" style="margin-top: 18px; margin-left: 5px">
                <span>
                    <img src="../assets/images/logo-new.png" alt="" height="60">
                </span>
            </a>
        </div>

        <!--- Sidemenu -->
        <div id="sidebar-menu" style="margin-top: 120px">

            <ul class="metismenu" id="side-menu">

                <!--<li class="menu-title">Navigation</li>-->

                <li>
                    <a href="#/datasets" >
                        <i class="fa fa-database"></i><span> Datasets </span>
                    </a>
                </li>

                <li>
                    <a href="#/jobs" aria-expanded="true">
                        <i class="fa fa-cogs"></i> <span> Jobs </span>
                    </a>
                    <!--<ul class="nav-second-level collapse in" aria-expanded="true">
                        <li><a href="javascript: void(0);">New Job</a></li>
                        <li><a href="javascript: void(0);">Running</a></li>
                        <li><a href="javascript: void(0);">Completed</a></li>
                    </ul>-->
                </li>

                <li>
                    <a href="#/models">
                        <i class="fa fa-cube"></i><span> Models </span>
                    </a>
                    <!--<ul class="nav-second-level collapse in">
                        <li><a href="javascript: void(0);">Models</a></li>
                        <li><a href="javascript: void(0);">Objectives</a></li>
                    </ul>-->
                </li>


            </ul>



        </div>
        <!-- Sidebar -->



        <div class="clearfix"></div>

    </div>
    <!-- Sidebar -left -->

    <new-dataset-modal/>
    <new-job-modal/>

    <div class="btn-group dropup" style="position: absolute; display: block; bottom: 50px; left: 70px">
            <button type="button" class="btn btn-custom dropdown-toggle" data-toggle="dropdown" aria-haspopup="true" aria-expanded="false" style="font-size: 22px;">New</button>
            <div class="dropdown-menu">
                <a class="dropdown-item" href="#" @click="newDataset" style="font-size: 18px;">Dataset</a>
                <a class="dropdown-item" href="#" @click="newJob" style="font-size: 18px;">Job</a>
                <!--<a class="dropdown-item" href="#" style="font-size: 18px;">Model</a>-->
                <!--<div class="dropdown-divider" style="font-size: 18px;"></div>
                <a class="dropdown-item" href="#" style="font-size: 18px;">Shared Computation</a>
                <a class="dropdown-item" href="#" style="font-size: 18px;">Dedicated Computation</a>-->
            </div>
        </div>

</div>
</template>
<script>
import NewDatasetModal from '@/modals/NewDataset.vue'
import NewJobModal from '@/modals/NewJob.vue'

export default {
    components: {
        NewJobModal,
        NewDatasetModal,
    },
    methods: {
        routeContains(string) {
            return this.$route.path.contains(string);
        },
        newDataset() {
            console.log("new dataset");
            this.$modal.show("new-dataset", {clickToClose: false});
        },
        newJob() {
            console.log("new job");
            this.$modal.show("new-job");
        }
    }
};
</script>
<style>
</style>
