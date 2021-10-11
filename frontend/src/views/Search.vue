<template>
	<div v-if="error_message.length > 0" class="alert alert-danger m-1 p-1 text-wrap text-break" role="alert">
		{{ error_message }}
	</div>

	<div v-if="sortingEnabled()" class="alert alert-primary m-1 p-1 text-wrap text-break" role="alert">
		Пока включена сортировка по цене, товары при прокрутке не подгружаются.
	</div>

	<form v-on:submit.prevent="onSearchSubmit" class="form-horizontal">
		<div class="d-flex flex-wrap">
			<div class="d-flex form-inline py-0 m-1">
				<VueMultiselect v-model="searchQuery.city" 
					:options="user.cities" 
					:multiple="false" 
					:searchable="true" 
					track-by="id"
					label="name"
					:close-on-select="true" 
					:show-labels="false"
					placeholder="Город">
				</VueMultiselect>
			</div>
			<div class="form-inline p-0 mx-1">
				<input class="form-check-input" v-model="searchQuery.inStockOnly" type="checkbox" value="" id="flexCheckDefault">
				<label class="form-check-label" for="flexCheckDefault">
					В наличии
				</label>
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
			<div class="form-inline flex-grow-1 p-0 mx-1">
				<input id="searchSupplier" v-model="searchQuery.supplier" class="form-control m-0" style="width:100%" placeholder="Поставщик"/>
			</div>
		</div>
	</form>


	<div class="table-responsive-lg p-0 pr-1 mr-1">
		<table class="table table-sm table-striped table-borderless m-1" ref="productsTable">
			<thead class="thead-dark text-truncate">
				<tr class="d-flex text-wrap">
					<th v-if="devMode" class="col-1">Score</th>
					<th v-if="devMode" class="col-1">Категория</th>
					<th v-else class="col-2">Категория</th>
					<th class="col-2">Артикул</th>
					<th class="col-2">Название</th>
					<th class="col-2">Описание</th>
					<th class="col-1">Остаток</th>
					<th class="col-1" role="button" @click="switchSorting">
						<div class="d-flex flex-row">
							<div>Цена (без НДС) </div>
							<div v-if="(priceSorting==='up')" class="mx-1"><i class="fas fa-sort-up"></i></div>
							<div v-if="(priceSorting==='down')" class="mx-1"><i class="fas fa-sort-down"></i></div>
						</div>
					</th>
					<th class="col-2 pr-1">Поставщик</th>
				</tr>
			</thead>
			<tbody > 
				<tr class="d-flex text-wrap text-break" v-for="(product, index) in searchResults" :key="index">

					<td v-if="devMode" class="col-1 "> {{product.score}}</td>
					<td v-if="devMode" class="col-1 "> {{product.category}}</td>
					<td v-else class="col-2 "> {{product.category}}</td>
					<td class="col-2 "> {{product.code}} </td>
					<td class="col-2 "> <a  target="_blank" :href="'/catalog/product/'+product.id" >{{product.name}}</a> </td>
					<td class="col-2 " data-toggle="tooltip" :title="product.description">  {{ truncate(stripHTML(product.description), 100, true)}} </td>
					<td class="col-1 "> {{product.rest}} </td>
					<td class="col-1 "> {{product.price}} </td>
					<td class="col-2 "> {{product.supplier}} </td>

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

<style src="vue-multiselect/dist/vue-multiselect.css">
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
.table-nostriped tbody tr:nth-of-type(odd) {
  background-color:transparent;
}
</style>

<script>
import axios from 'axios'
axios.defaults.baseURL = '/' + process.env.VUE_APP_BASE_URL

import VueMultiselect from 'vue-multiselect'
import { VueCookieNext } from 'vue-cookie-next'

function doubleRaf (callback) {
	requestAnimationFrame(() => {
		requestAnimationFrame(callback)
	})
}

export default {
	name: 'App',
	props: ["user"],
	components: { VueMultiselect },
	data() { 
		return {
			loading:false,
			error_message:"",
			searchQuery: {cityId:0},
			currentSearchQuery: {},
			searchResults:[],
			page:0,
			totalPages:0,
			devMode:(process.env.NODE_ENV  === "development"),
			priceSorting:"",
		} 
	},
	computed: {
		searchButtonDisabled() {
			return this.loading || !this.searchQueryNotEmpty()
		},
	},
	created() {
		window.onpopstate = this.onPopState
		this.loadFromCookie(this.searchQuery, "supplier")
		this.loadFromCookie(this.searchQuery, "inStockOnly")
		this.loadFromCookie(this.searchQuery, "city")
		this.searchQuery.text = this.$route.query.query		
		if(this.searchQueryNotEmpty())
			this.searchProducts()
	},
	watch : {
		user : function(newVal){
			if(!newVal.admin && (newVal.cities == null || newVal.cities.length == 0)) {
				this.error_message = "У вашего аккаунта не задано ни одного склада на который можно доставить груз"
			}
		}
	},
	mounted() {

		this.$nextTick(function() {
			window.addEventListener('scroll', this.onScroll)
		});
	},
	beforeUnmount() {
		window.removeEventListener('scroll', this.onScroll)
	},  
  methods: {
		loadFromCookie(val, name) {
			if(VueCookieNext.getCookie(name)) {
				var v = VueCookieNext.getCookie(name)

				if(v != null && v != undefined) {
					val[name] = v
				}
			}
		},
		saveToCookie(val, name) {
			if(name in val && val[name] !== undefined)
				VueCookieNext.setCookie(name, val[name])
		},
		sortingEnabled() {
			return this.priceSorting === 'up' || this.priceSorting === 'down'
		},
		switchSorting() {
			if (this.priceSorting == "up") {
				this.priceSorting = "down"
			}
			else if (this.priceSorting == "down") {
				this.priceSorting = ""
			}
			else {
				this.priceSorting = "up"
			}
			this.sortResults()
		},
		sortResults() {
			if (this.priceSorting == "down") {
				this.searchResults.sort((b, a) => (a.price > b.price) ? 1 : ((b.price > a.price) ? -1 : 0))
			}
			else if (this.priceSorting == "up") {
				this.searchResults.sort((a, b) => (a.price > b.price) ? 1 : ((b.price > a.price) ? -1 : 0))
			}
			else {
				this.searchResults.sort((b, a) => (a.score > b.score) ? 1 : ((b.score > a.score) ? -1 : 0))
			}
		},
		searchQueryNotEmpty() {
				return this.searchQuery.text != null && this.searchQuery.text.length > 2
		},
		onPopState(e) {
			this.searchQuery = e.state
			if(this.searchQueryNotEmpty())
				this.searchProducts()
		},
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
			this.saveToCookie(this.searchQuery, "supplier")
			this.saveToCookie(this.searchQuery, "inStockOnly")
			this.saveToCookie(this.searchQuery, "city")
			this.searchProducts().then( ()  => {
				history.pushState( JSON.parse(JSON.stringify(this.searchQuery)), this.searchQuery.text, "/" + process.env.VUE_APP_BASE_URL + "/search?" + this.searchQuery.text)  
			})
		},
    searchProducts() {
			this.searchResults = []
			this.currentSearchQuery = JSON.parse(JSON.stringify(this.searchQuery))
			if(this.currentSearchQuery.city != null)
				this.currentSearchQuery.cityId = this.currentSearchQuery.city.id
			else
				this.currentSearchQuery.cityId = 0
			delete this.currentSearchQuery.city
      this.loading = true

			return this.getProductsList()
    },
		loadMoreProducts() {
      this.loading = true
			this.currentSearchQuery.page = this.page+1

			this.getProductsList()
		},
		getProductsList() {
			return axios({
					method: "GET", 
					url: "/methods/products",
					params: this.currentSearchQuery
				})      
				.then(res => {
					this.error_message = ""

					if(res.data.results.length > 0) {
						this.searchResults = this.searchResults.concat(res.data.results)
						this.sortResults()
					}

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
    getAxiosErrorMessage : function(error) {
      if (error.response != null && error.response.data != null && error.response.data != "") {
        return error.response.data

      } else {
        return error
      }
    },
		onScroll : function () {
			if (!this.loading && this.page +1 < this.totalPages && !this.sortingEnabled()) {
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
