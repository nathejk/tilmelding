import "@babel/polyfill";
//import "./assets/safari.scss"
import Vue from "vue";
import moment from "moment";
import router from './router'
//import 'bootstrap'
import App from "./App";
//import axios from 'axios'

/*
import VueResource from 'vue-resource';
Vue.use(VueResource);

*/
import VCalendar from 'v-calendar'
//console.log('vcal', VCalendar)
Vue.use(VCalendar);

Vue.config.productionTip = false;

// Attempt at some error handling
//Vue.config.errorHandler = function (err, vm, info) {
//  Vue.notify({ type: "error", text: err });
//};

Vue.filter('formatDate', datetimeString => {
  if (datetimeString) {
    return moment.parseZone(datetimeString).format('YYYY-MM-DD')
  }
})

//window.addEventListener("unhandledrejection", function (err, promise) {
//  Vue.notify({ type: "error", text: err });
//});

//window.onerror = function (msg, url, line, col, error) {
//  Vue.notify({ type: "error", text: msg });
//};

// Start timer. Provides a reactive timestamp, updated each second
//store.dispatch("time/start");

    // Launch vue app!
    window.Vue = new Vue({
      el: "#app",
      render: h => h(App),
      router,
    });

