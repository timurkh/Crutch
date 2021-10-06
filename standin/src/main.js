import { createApp } from 'vue'
import VTooltip from 'v-tooltip'
import App from './App.vue'

let app = createApp(App)
app.use(VTooltip)
app.mount('#app')
