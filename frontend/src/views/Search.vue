<template>
	<div v-if="error_message.length > 0" class="alert alert-danger m-1 p-1 text-wrap text-break" role="alert">
		{{ error_message }}
	</div>


	<form v-on:submit.prevent="onSearchSubmit" class="form-horizontal">
		<div class="d-flex flex-wrap">
			<div class="form-inline p-1 pb-0 pr-0">
				<select id="city" class="form-control w-auto" v-model="searchQuery.cityId">
					<option :value="Number(0)">Город</option>
					<option v-for="city in user.cities" :value="city.id" :key="city.id">{{ city.name }} </option>
				</select>
			</div>
			<div class="form-inline flex-grow-1 p-0 mx-1">
				<input v-model="searchQuery.text" type="text" id="search-input" placeholder="Продукт" class="form-control flex-fill">
			</div>
			<div class="ml-auto p-1 mb-0">
				<button class="btn btn-info my-1" :disabled="searchButtonDisabled">Найти!</button>
			</div>
		</div>
		
		<div class="d-flex flex-wrap">
			<div class="form-inline p-1 pb-0 pr-0">
				<input id="searchCategory" v-model="searchQuery.category" class="form-control m-0" style="width:100%" placeholder="Категория"/>
			</div>
			<div class="form-inline flex-grow-1 p-0 mx-1">
				<input id="searchCode" v-model="searchQuery.code" class="form-control m-0" style="width:100%" placeholder="Артикул"/>
			</div>
			<div class="form-inline flex-grow-1 p-0 mx-1">
				<input id="searchName" v-model="searchQuery.name" class="form-control m-0" style="width:100%" placeholder="Название"/>
			</div>
			<div class="form-inline flex-grow-1 p-0 mx-1">
				<input id="searchProperty" v-model="searchQuery.property" class="form-control m-0" style="width:100%" placeholder="Свойства"/>
			</div>
		</div>
	</form>


	<div class="table-responsive-lg p-0 pr-1 mr-1">
		<table class="table table-sm table-striped table-borderless m-1" ref="productsTable">
			<thead class="thead-dark text-truncate">
				<tr class="d-flex">
					<th class="text-wrap col-2">Категория</th>
					<th class="text-wrap col-2">Артикул</th>
					<th class="text-wrap col-2">Название</th>
					<th class="text-wrap col-2">Описание</th>
					<th class="text-wrap col-1">Остаток</th>
					<th class="text-wrap col-1">Цена</th>
					<th class="text-wrap col-2 pr-1">Поставщик</th>
				</tr>
			</thead>
			<tbody > 
				<tr class="d-flex" v-for="(product, index) in searchResults" :key="index">

					<td class="text-wrap col-2 "> {{showInDevMode("[" + product.score + "] ") + product.category}}</td>
					<td class="text-wrap col-2 "> {{product.code}} </td>
					<td class="text-wrap col-2 "> <a  target="_blank" :href="'/catalog/product/'+product.id" >{{product.name}}</a> </td>
					<td class="text-wrap col-2 " data-toggle="tooltip" :title="product.description">  {{ truncate(stripHTML(product.description), 100, true)}} </td>
					<td class="text-wrap col-1 "> {{product.rest}} </td>
					<td class="text-wrap col-1 "> {{product.price}} </td>
					<td class="text-wrap col-2 "> {{product.supplier}} </td>

				</tr>
			</tbody>
		</table>
	</div>

	<div v-if="loading">
		<div class="mt-1" align="center">
			<div class="spinner-border mt-1" role="status">
				<span class="sr-only">Loading...</span>
			</div>
		</div>
	</div>
</template>

.<style>
.form-control::placeholder { /* Chrome, Firefox, Opera, Safari 10.1+ */
            color: #999999;
            opacity: 1; /* Firefox */
}

.form-control:-ms-input-placeholder { /* Internet Explorer 10-11 */
            color: #999999;
}

.form-control::-ms-input-placeholder { /* Microsoft Edge */
            color: #999999;
 }
</style>

<script>
import axios from 'axios'
axios.defaults.baseURL = '/' + process.env.VUE_APP_BASE_URL

function doubleRaf (callback) {
	requestAnimationFrame(() => {
		requestAnimationFrame(callback)
	})
}

export default {
	name: 'App',
	props: ["user"],
	data() { return {
		loading:false,
		error_message:"",
		searchQuery: {cityId:0},
		currentSearchQuery: {},
		searchResults:[],
		page:0,
		totalPages:0,
	} },
	computed: {
		searchButtonDisabled() {
			return this.loading || !(
				this.searchQuery.text != null && this.searchQuery.text.length > 2 || 
				this.searchQuery.category != null && this.searchQuery.category.length > 2 || 
				this.searchQuery.code != null && this.searchQuery.code.length > 2 || 
				this.searchQuery.property != null && this.searchQuery.property.length > 2 || 
				this.searchQuery.name != null && this.searchQuery.name.length > 2  
			) 
		},
	},
	created() {
		window.addEventListener('popstate', e => {
				this.searchQuery = e.state;
				this.searchProducts();
		});
	},
	mounted() {

		this.$nextTick(function() {
			window.addEventListener('scroll', this.onScroll)
		})
	},
	watch : {
		user : function(newVal){
			if(!newVal.admin && (newVal.cities == null || newVal.cities.length == 0)) {
				this.error_message = "У вашего аккаунта не задано ни одного склада на который можно доставить груз"
			}
		}
	},
	beforeUnmount() {
		window.removeEventListener('scroll', this.onScroll)
	},  
  methods: {
		showInDevMode : function(value) {
			if(process.env.NODE_ENV === 'development') {
				return value;
			}
			return ""
		},
		stripHTML: function (value) {
			return value.replace(/<\/?[^>]+>/ig, " ");
		},
		onSearchSubmit() {
			history.pushState( Object.assign({}, this.searchQuery), this.searchQuery.text, "/" + process.env.VUE_APP_BASE_URL + "/search?" + this.searchQuery.text)  
			this.searchQuery.operator = ""
			this.searchProducts().then( r  => { // eslint-disable-line no-unused-vars
				if(this.searchResults===undefined || this.searchResults.length == 0) {
					this.searchQuery.operator = "OR"
					this.searchProducts()
				}
			})
		},
		getProductsList() {
			return axios({
					method: "GET", 
					url: "/methods/products",
					params: this.searchQuery
				})      
				.then(res => {
					this.error_message = ""
					this.searchResults = res.data.results
					this.page = res.data.page
					this.totalPages = res.data.totalPages

					this.loading = false

					this.$nextTick(doubleRaf(() => this.onScroll()))
				})
				.catch(error => {
					this.error_message = "Ошибка во время поиска: " + this.getAxiosErrorMessage(error)
					this.searchResults = []
					this.loading = false
				})
		},
    searchProducts() {
			this.searchResults = []
			this.currentSearchQuery = Object.assign({}, this.searchQuery)
      this.loading = true

			this.getProductsList()
    },
		loadMoreProducts() {
      this.loading = true
			this.currentSearchQuery.page = this.page+1

			this.getProductsList()
		},
    getAxiosErrorMessage : function(error) {
      if (error.response != null && error.response.data != null && error.response.data != "") {
        return error.response.data

      } else {
        return error
      }
    },
		onScroll : function () {
			if (!this.loading && this.page +1 < this.totalPages) {
				let element = this.$refs.productsTable
				if ( element != null && element.getBoundingClientRect().bottom < window.innerHeight ) {
					this.loadMoreProducts()
				}
			}
		},
		truncate : function( str, n, useWordBoundary ){
			if (str.length <= n) { return str; }
				const subString = str.substr(0, n-1); 
				return (useWordBoundary
					? subString.substr(0, subString.lastIndexOf(" "))
					: subString) + "...";
		}
  },
}
</script>
