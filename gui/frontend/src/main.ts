import { createApp } from "vue";
import App from "./App.vue";
import router from "@/router";
import stores from "@/stores";
import naive from "naive-ui";

const app = createApp(App);

app.use(router);
app.use(stores);
app.use(naive);
app.mount("#app");
