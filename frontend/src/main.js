import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { VueWindowSizePlugin } from 'vue-window-size/option-api';

const app = createApp(App);
app.use(VueWindowSizePlugin);
app.use(router);
app.mount('#app');
