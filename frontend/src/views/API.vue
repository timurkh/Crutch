<template>
	<div v-if="error_message.length > 0" class="alert alert-danger m-1 p-1 text-wrap text-break" role="alert">
		{{ error_message }}
	</div>

	<div class="border m-3">
		<div class="d-flex font-weight-bold flex-wrap m-1 mb-2">
			Аутентификация
		</div>
		<form class="form-horizontal" @submit.prevent>
			<div class="d-flex flex-wrap m-1 mb-2 justify-content-end">
				<div class="form-inline form-check m-1 p-0">
					<input class="" type="checkbox" value="" id="enabled" v-model="api.enabled" @change="onChangeEnabled">
					<label class="form-check-label m-1" for="haveOrders">
						Включен
					</label>
				</div>
				<div class="form-inline align-self-center py-0 m-0 mx-2" style="min-width:120px;">
					<VueMultiselect v-model="api.authType" 
						:options="authTypes" 
						:multiple="false" 
						:searchable="false" 
						:close-on-select="true" 
						:show-labels="false"
						:allow-empty="false"
						>
					</VueMultiselect>
				</div>
				<div class="form-inline flex-grow-1 m-1 mx-2 p-0">
						<label for="Login" class="mr-1 ml-0 my-1">Login</label> 
						<input type="text my-0" class="flex-fill form-control" v-model="api.login" disabled="true"/>
				</div>
				<div class="form-inline flex-grow-1 m-1 mx-2 p-0">
						<label for="Password" class="mr-1 ml-0 my-1">Password</label> 
						<input :type="passwordFieldType" class="flex-fill form-control" v-model="api.password" disabled="true"/>
				</div>
				<div class="d-flex form-inline mr-1 dropleft">
					<button class="btn btn-secondary" type="button" @click="switchVisibility">
						{{passwordVisButton}}
					</button>
				</div>
				<div class="d-flex form-inline mr-1 dropleft">
					<button class="btn btn-secondary" type="button" @click="genNewPassword">
						Новый пароль
					</button>
				</div>
			</div>
		</form>
	</div>

	<div style="margin:20px;">
		<iframe
			id="iframe"
			:src="getSwaggerUrl()" 
			style="border: none;"
			scrolling="no"
			width="100%" 
			v-resize="{ heightCalculationMethod:'max', }">
		</iframe>
	</div>
</template>

<style src="vue-multiselect/dist/vue-multiselect.css"/>
<style>
.multiselect__tags {
    padding: 6px 38px 0 6px !important;
		border: 1px solid #ced4da !important;
}
</style>

<script>
	import axios from 'axios'
	axios.defaults.baseURL = '/' + process.env.VUE_APP_BASE_URL
	import moment from 'moment'
	moment.updateLocale('en', {
			week : {
					dow :0  // 0 to 6 sunday to saturday
			}
	});
	import VueMultiselect from 'vue-multiselect'


export default {
	name: 'App',
	props: ["user"],
	components: { VueMultiselect },
	data() { 
		return {
			error_message:"",
			api: { 
				authType: "Basic"
			},
			authTypes: [
				"Basic"
			],
			passwordFieldType: "password",
			passwordVisButton: "Показать",
		} 
	},
	created() {
	
			axios({
				method: "GET", 
				url: "/methods/apiCredentials"
			})      
			.then(res => {
				this.api =res.data
			})
			.catch(error => {
				this.error_message = "Не удалось загрузить реквизиты API: " + this.getAxiosErrorMessage(error)
				this.api = []
			})
	
	},
  methods: {
		getSwaggerUrl() {
			return "/" + process.env.VUE_APP_BASE_URL + "/swagger/index.html"
		},
		switchVisibility() {
			if (this.passwordFieldType === "password") {
				this.passwordFieldType = "text"
				this.passwordVisButton = "Скрыть"
			} else {
				this.passwordFieldType = "password"
				this.passwordVisButton = "Показать"
			}
		},
		onChangeEnabled() {
			return axios({
				method: "PUT", 
				url: "/methods/apiCredentials",
				headers: { "X-CSRF-Token": this.user.csrf },
				data: {
					enabled : this.api.enabled
				}
			})      
			.then(() => {
				this.error_message = ""
			})
			.catch(error => {
				this.error_message = "Не удалось сохранить изменения: " + this.getAxiosErrorMessage(error)
			})
		},
		genNewPassword() {
			return axios({
				method: "PUT", 
				url: "/methods/apiCredentials",
				data: {
					password : true
				}
			})      
			.then(res => {
				this.error_message = ""
				this.api.password = res.data.password
			})
			.catch(error => {
				this.error_message = "Не удалось сохранить изменения: " + this.getAxiosErrorMessage(error)
			})
		},
  },
}
</script>
