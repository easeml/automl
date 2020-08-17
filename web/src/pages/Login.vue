<template>
    <div class="account-page">
        <!-- Begin page -->
        <div :class="['accountbg', bgstyle]"></div>

        <div class="wrapper-page account-page-full">

            <div class="card">
                <div class="card-block">

                    <div class="account-box">

                        <div class="card-box p-5">
                            <h2 class="text-uppercase text-center pb-4">
                                <div class="topbar-left">
                                    <router-link to="/" class="logo">
                                        <span>
                                            <img src="../assets/images/logo-below.png" alt="" height="150">
                                        </span>
                                    </router-link>
                                </div>
                            </h2>

                            <form class="" @submit.prevent="login">

                                <div class="form-group m-b-20 row">
                                    <div class="col-12">
                                        <label for="userid">User ID</label>
                                        <input class="form-control" type="username" id="userid" v-model="userid" placeholder="Enter your user ID">
                                    </div>
                                </div>

                                <div class="form-group row m-b-20">
                                    <div class="col-12">
                                        <label for="password">Password</label>
                                        <input class="form-control" type="password" id="password" v-model="password" placeholder="Enter your password">
                                    </div>
                                </div>

                                <div class="form-group row panel-body">
                                    <div class="col-12">
                                        <h5 class="text-center">- or -</h5>
                                    </div>
                                </div>

                                <div class="form-group row m-b-20">
                                    <div class="col-12">
                                        <label for="password">API Key</label>
                                        <input class="form-control" type="password" id="apiKey" v-model="apiKey" placeholder="Enter your API Key">
                                    </div>
                                </div>

                                <div class="form-group row text-center m-t-10">
                                    <div class="col-12">
                                        <button class="btn btn-block btn-custom waves-effect waves-light" type="submit">Sign In</button>
                                    <div class="text-danger" v-if="error">
                                        <p><i class="mdi mdi-close-circle"></i>&nbsp;{{error}}</p>
                                    </div>
                                    </div>
                                </div>

                            </form>

                        </div>
                    </div>

                </div>
            </div>

            <div class="m-t-40 text-center footcontainer">
                <content-footer></content-footer>
            </div>
            
        </div>
        
    </div>
</template>
<script>
import ContentFooter from "@/layout/ContentFooter.vue";
export default {
    components: {
        ContentFooter
    },
    data() {
        return {
            userid: "",
            password: "",
            apiKey: "",
            error: "",
            bgstyle: "bg1",
        }
    },
    beforeMount() {

        this.bgstyle = "bg" + Math.ceil(Math.random() * 5);

        let apiKey = this.$route.query["api-key"];
        if (apiKey) {
            this.apiKey = apiKey;
            this.login();
        } else {
            this.userid = "";
            this.password = "";
            this.apiKey = "";
        }
    },
    methods: {
        login() {

            this.error = ""
            console.log('login function')
            console.log(process.env.VUE_APP_SERVER_ADDRESS)
            
            // We either get the server address from the environment variable, or from the current browser window.
            let host = process.env.VUE_APP_SERVER_ADDRESS || window.location.host;
            let serverAddress = window.location.protocol + "//" + host + "/";

            console.log(this.apiKey);

            if (this.apiKey) {
                let userCredentials = {apiKey: this.apiKey};
                this.$store.commit('createClient', {serverAddress, userCredentials})
                let context = this.$store.getters.getClientContext

                context.getUserById("this")
                .then(result => {

                    localStorage.setItem("user", JSON.stringify(result));
                    localStorage.setItem("context", JSON.stringify(context));

                    if(this.$route.params.nextUrl != null){
                        this.$router.push(this.$route.params.nextUrl)
                    }
                    else{
                        this.$router.push({name: "home"})
                    }
                    
                })
                .catch(error => {
                    console.log("ERROR: ",error)
                    this.error = "Server error."
                    console.log(error);
                });


            } else {

                let userCredentials = {username: this.userid, password: this.password};
                this.$store.commit('createClient', {serverAddress, userCredentials})
                let context = this.$store.getters.getClientContext

                context.loginUser()
                .then(result => {
                    
                    context.getUserById("this")
                    .then(result => {

                        localStorage.setItem("user", JSON.stringify(result));
                        localStorage.setItem("context", JSON.stringify(context));

                        if(this.$route.params.nextUrl != null){
                            this.$router.push(this.$route.params.nextUrl)
                        }
                        else{
                            this.$router.push({name: "home"})
                        }
                        
                    })
                    .catch(e => {
                        this.error = "Server error."
                        console.log(error);
                    });
                })
                .catch(e => {
                    this.error = "Bad credentials."
                    console.log(e);
                });
            }

        }
    }
};

</script>
<style scoped>
.footcontainer {
    position: fixed;
    bottom: 0;
    width: 100%;
}
.bg1 {
    background: url('../assets/images/background-1.jpg');
    background-size: cover;
}
.bg2 {
    background: url('../assets/images/background-2.jpg');
    background-size: cover;
}
.bg3 {
    background: url('../assets/images/background-3.jpg');
    background-size: cover;
}
.bg4 {
    background: url('../assets/images/background-4.jpg');
    background-size: cover;
}
.bg5 {
    background: url('../assets/images/background-5.jpg');
    background-size: cover;
}
</style>
