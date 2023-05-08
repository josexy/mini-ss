import { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  {
    path: "/",
    component: () => import("@/layout/index.vue"),
    children: [
      {
        path: "/",
        name: "home",
        component: () => import("@/views/home/index.vue"),
        meta: { keepAlive: true },
      },
      {
        path: "/server",
        name: "server",
        component: () => import("@/views/server/index.vue"),
        meta: { keepAlive: true },
      },
      {
        path: "/connection",
        name: "connection",
        component: () => import("@/views/connection/index.vue"),
        meta: { keepAlive: true },
      },
      {
        path: "/rule",
        name: "rule",
        component: () => import("@/views/rule/index.vue"),
        meta: { keepAlive: true },
      },
      {
        path: "/setting",
        name: "setting",
        component: () => import("@/views/setting/index.vue"),
        meta: { keepAlive: true },
      },
      {
        path: "/about",
        name: "about",
        component: () => import("@/views/about/index.vue"),
        meta: { keepAlive: true },
      },
    ],
  },
];
