import './assets/style.css'
import 'primeicons/primeicons.css'
import Lara from '@/presets/lara';      //import preset


import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config';


import App from './App.vue'
import router from './router'

const app = createApp(App)

app.use(createPinia())
app.use(PrimeVue, {
    unstyled: true,
    pt: Lara                            //apply preset
});
app.use(router)

app.mount('#app')
