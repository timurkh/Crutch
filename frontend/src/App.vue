<template>
	<nav id="navbar" class="navbar navbar-expand-sm bg-dark navbar-dark">
		<a class="navbar-brand" href="/">Industrial.Market</a>
		<button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#main_nav">
			<span class="navbar-toggler-icon"></span>
		</button>

		<div class="navbar-collapse collapse" id="main_nav">
			<ul class="navbar-nav">
				<li class="nav-item"><router-link class="nav-link" to="/">Поиск товаров</router-link></li>
				<li class="nav-item"><router-link class="nav-link" to="/orders">Заказы</router-link></li>
				<li v-if="user.admin || user.staff" class="nav-item"><router-link class="nav-link" to="/counterparts">Контрагенты</router-link></li>
				<li v-if="user.companyAdmin || user.admin" class="nav-item"><router-link class="nav-link" to="/api">API</router-link></li>
			</ul>
			<ul class="navbar-nav ml-auto">
				<li class="nav-item"><hr class="border-top"></li>
				<li class="nav-item"> <a id="navbar-userinfo" class="nav-link" href="/profile_settings/user-profile-editor">{{user.name+"["+user.email+"]"}}</a></li>
			</ul>
		</div>
	</nav>

	<div id="app" class="container-fluid p-0 p-md-2">
    <div v-if="error_message.length > 0" class="alert alert-danger mx-1 my-2 p-1 text-wrap text-break" role="alert">
      {{ error_message }}
    </div>
		<router-view :user='user'/>
  </div>
</template>

<style>
#app {
	font-family: Avenir, Helvetica, Arial, sans-serif;
	-webkit-font-smoothing: antialiased;
	-moz-osx-font-smoothing: grayscale;
	text-align: center;
	color: #2c3e50;
}

#nav {
  padding: 30px;
}

#nav a {
  font-weight: bold;
  color: #2c3e50;
}

#nav a.router-link-exact-active {
  color: #42b983;
}

.multiselect {
  min-height: 0px !important;
}

.multiselect__tags {
	padding: 4px 30px 0 4px !important;
	border: 1px solid #ced4da !important;
	min-height: 38px !important;
}

.multiselect__select {
  margin-bottom: 0px !important;
  margin-top: 0px !important;
  height: 30px !important;
}

.multiselect__placeholder, .multiselect__single {
  margin-top: 0px !important;
  margin-bottom: 0px !important;
	padding-top: 0px !important;
	padding-bottom: 0px !important;
}

@media (max-width:1620px) {
  * {
    font-size: 0.9rem;
  }
	#app [class^="multiselect"] {
		font-size: 0.9rem !important;	
	}
	#app .form-control {
		font-size: 0.9rem !important;	
	}
	#app .btn {
		font-size: 0.9rem !important;	
	}
	.multiselect__tags {
		padding: 4px 30px 0 4px !important;
		border: 1px solid #ced4da !important;
		min-height: calc(1.5em + .75rem + 2px) !important;
	}
}

</style>

<script>
import { ref } from 'vue'
import axios from 'axios'
axios.defaults.baseURL = '/' + process.env.VUE_APP_BASE_URL

export default {
	name: 'App',

	setup() {

			let error_message = ref("")
			let user = ref({
				cities : [],
				admin : false
			}) 

      axios({
				method: "GET", 
				url: "/methods/currentUser"
			})      
      .then(res => {
        user.value = res.data
      })
      .catch(error => {
				console.log(error.response);

        error_message.value = "Ошибка во время проверки сессии. " + error.response.data
				
				if (error.response.status == 401) {
					window.location.href = "/login";
				} else if (error.response.status == 403) {
					window.location.href = "/";
				}
			})
			return { error_message, user }
    },
	}
</script>
