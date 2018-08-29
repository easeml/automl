import Vue from "vue";
import Router from "vue-router";
//import Home from "@/views/Home.vue";

import DashboardLayout from "@/layout/DashboardLayout.vue";

import Datasets from "@/pages/Datasets.vue";
import Jobs from "@/pages/Jobs.vue";
import Models from "@/pages/Models.vue";
import Job from "@/pages/Job.vue";
import Login from "@/pages/Login.vue";

Vue.use(Router);

const routes = [
  {
    path: "/",
    name: "home",
    component: DashboardLayout,
    redirect: "/datasets",
    meta: { 
        requiresAuth: true
    },
    children: [
      {
        path: "datasets",
        name: "datasets",
        component: Datasets,
        meta: { 
            requiresAuth: true
        }
      },
      {
        path: "jobs",
        name: "jobs",
        component: Jobs,
        meta: { 
            requiresAuth: true
        }
      },
      {
        path: "jobs/:id",
        name: "job",
        component: Job,
        meta: { 
            requiresAuth: true
        }
      },
      {
        path: "models",
        name: "models",
        component: Models,
        meta: { 
            requiresAuth: true
        }
      }
    ]
  },
  {
    path: "/login",
    name: "login",
    component: Login,
    meta: { 
        guest: true
    }
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    // component: () => import(/* webpackChunkName: "about" */ "@/views/About.vue")
  }
];

export default routes;
