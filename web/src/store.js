import Vue from "vue";
import Vuex from "vuex";
import client from "easemlclient"
import router from './router'

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    clientContext: null,
    clientCallbacks: null,
  },
  getters: {
    getClientContext: state => {
      return state.clientContext
    }
  },
  mutations: {
    createClient: function(state, payload) {
      let callbacks = {
        unauthErrorCallback: function() {
          console.log("Unauthorized access callback called")
          localStorage.removeItem("user");
          localStorage.removeItem("context");
          router.push({name: "login"})
        }
      }
      state.clientContext = new client.Context(payload.serverAddress, payload.userCredentials, callbacks);
    },
    deleteClient: function(state) {
      state.clientContext=null
    }
  },
  actions: {
    getClient: ({ commit, state }, payload) =>
        new Promise((resolve, reject) => {
          if (state.clientContext) {
            resolve(state.clientContext)
          } else {
            let oldContext = JSON.parse(localStorage.getItem("context"))
            if (oldContext.serverAddress && oldContext.userCredentials){
              commit('createClient', { serverAddress: oldContext.serverAddress, userCredentials:oldContext.userCredentials})
              resolve(state.clientContext)
            }else{
              reject()
            }
          }
        })
  }
});
