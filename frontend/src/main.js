import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import { VueWindowSizePlugin } from 'vue-window-size/option-api';
import iframeResize from 'iframe-resizer/js/iframeResizer';



const app = createApp(App);
app.use(VueWindowSizePlugin);
app.use(router);
app.directive('resize', {
  mounted(el, binding) {
    iframeResize(binding.value, el)
  },
  beforeUnmount: function (el) {
    el.iFrameResizer.removeListeners();
  }
})
app.mixin( {
  methods: {
    getAxiosErrorMessage : function(error) {
      if (error.response != null && error.response.data != null && error.response.data != "") {
        return error.response.data

      } else {
        return error
      }
    },
		getInDevMode : function(value) {
			if(process.env.NODE_ENV === 'development') {
				return value;
			}
		},
  }
})

app.mount('#app');

