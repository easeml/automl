import Vue from "vue";
import VueRouter from "vue-router";
import routes from "./routes";
Vue.use(VueRouter);

// configure router
const router = new VueRouter({
  routes, // short for routes: routes
  mode: 'abstract',
  linkActiveClass: "active"
});

router.beforeEach((to, from, next) => {
  if(to.matched.some(record => record.meta.requiresAuth)) {
    if (localStorage.getItem('user') == null) {

      // If the user isn't logged in, we redirect to the login screen.
      next({
          path: '/login',
          params: { nextUrl: to.fullPath },
          query: to.query
      });
    } else {
      next();
    }
  } else {
    next();
  }
});

router.replace('/')

export default router;
