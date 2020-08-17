import Vue from "vue";
import App from "./App.vue";
import router from "./router/index";
import store from "./store";
import VModal from 'vue-js-modal'
import VueSweetalert2 from 'vue-sweetalert2';
import Notifications from 'vue-notification';

Vue.use(VModal);
Vue.use(VueSweetalert2);
Vue.use(Notifications);

global.jQuery = require("jquery");
var $ = global.jQuery;
window.$ = $;

require("bootstrap");

//import MetisMenu from "metismenujs";

Vue.config.productionTip = false;

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount("#app");

/*
new MetisMenu("#side-menu", {
  toggle: false
})*/
