import { createRouter, createWebHistory } from "vue-router";
import Chat from "../views/Chat.vue";
import Home from "../views/Home.vue";
import Login from "../views/Login.vue";
import { useUserStore } from "../store/user.js";
import { usedisksStore } from "../store/disks.js";
import { sendCommand } from "../services/send_commands.ts";

const routes = [
  { path: "/", component: Chat },
  {
    path: "/login",
    component: Login,
    meta: { guestOnly: true },
    name: "Login",
  },
  {
    path: "/home",
    component: Home,
    meta: { requiresAuth: true },
    name: "Home",
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to, from, next) => {
  const { user } = useUserStore();
  const { setdisks, cleardisks } = usedisksStore();

  if (to.meta.guestOnly && user.value) {
    cleardisks();
    const commandObj = {
      command: "disks",
      timestamp: new Date().toISOString(),
      adjunt_file: null,
    };

    const res = await sendCommand(commandObj);
    setdisks(res);

    return next("/home");
  }
  /*
  if (to.meta.requiresAuth && !user.value) {
    cleardisks();
    return next("/login");
  }*/

  if (to.meta.requiresAuth /*&& user.value*/) {
    cleardisks();
    const commandObj = {
      command: "disks",
      timestamp: new Date().toISOString(),
      adjunt_file: null,
    };

    const res = await sendCommand(commandObj);

    setdisks(res);
  }

  next();
});

export default router;
