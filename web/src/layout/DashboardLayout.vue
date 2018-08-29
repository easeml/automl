<template>
    <div id="wrapper">
        <side-bar></side-bar>
        <div class="content-page">
            <top-navbar></top-navbar>
            <dashboard-content></dashboard-content>
            <div class="m-t-40 text-center footcontainer">
                <content-footer></content-footer>
            </div>
        </div>
    </div>
</template>
<script>
import SideBar from "./Sidebar.vue";
import TopNavbar from "./TopNavbar.vue";
import ContentFooter from "./ContentFooter.vue";
import DashboardContent from "./Content.vue";
export default {
    components: {
        SideBar,
        TopNavbar,
        ContentFooter,
        DashboardContent,
    }
};
</script>
<style scoped>
.footcontainer {
    position: fixed;
    bottom: 0;
    width: 100%;
}
</style>
