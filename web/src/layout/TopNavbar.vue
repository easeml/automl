<template>
<div class="topbar">
    <nav class="navbar-custom">

        <div class="float-right mb-0">
            <div class="row">
                <img src="../assets/images/user.png" alt="user" class="rounded-circle usericon">
                <div class="lead ml-1 usernametext">{{user}}</div>
                <button type="button" class="btn btn-light btn-sm waves-effect" @click="logout">Logout</button>
            </div>
        </div>

        <ul class="list-inline menu-left mb-0">
            <li class="float-left">
                <button class="button-menu-mobile open-left disable-btn">
                    <i class="dripicons-menu"></i>
                </button>
            </li>
            <li>
                <div class="page-title-box">
                    <ol class="breadcrumb">
                        <li v-for="b in breadcrumbs" class="breadcrumb-item" :key="b.cname"><router-link :to="{ name: b.name}">{{b.cname}}</router-link></li>
                    </ol>
                    <h4 class="page-title">{{currentName}} </h4>
                </div>
            </li>
        </ul>

    </nav>
</div>

</template>
<script>
export default {
    data() {
        return {
            user: "",
        };
    },
    computed: {
        breadcrumbs() {
            let result = [];
            for (let i = 0; i < this.$route.matched.length; i++) {
                let item = {
                    cname: this.capitalizeFirstLetter(this.$route.matched[i].name),
                    name: this.$route.matched[i].name,
                };
                result.push(item);
            }
            return result;
        },
        currentName() {
            return this.capitalizeFirstLetter(this.$route.name);
        }
    },
    methods: {
        capitalizeFirstLetter(string) {
            return string.charAt(0).toUpperCase() + string.slice(1);
        },
        logout() {
            localStorage.removeItem("user");
            localStorage.removeItem("context");
            this.$router.push({name: "home"});
        }
    },
    mounted() {
        console.log("top navbar mounted");
        let user = JSON.parse(localStorage.getItem("user"));
        console.log(user);
        this.user = user.name || user.id;
    }
};
</script>
<style>
.usericon {
    max-width: 30px;
    max-height: 30px;
    margin-right: 5px;
}
.usernametext {
    margin-right: 25px;
}
</style>
