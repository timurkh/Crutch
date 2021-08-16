<template>
  <div id="app" class="container">

    <div v-if="error_message.length > 0" class="alert alert-danger mx-1 my-2 p-1 text-wrap text-break" role="alert">
      {{ error_message }}
    </div>


    <div class="h1 mx-1 my-2 p-1 text-wrap text-break" role="alert">
			Поиск по продуктам
		</div>
		<form v-on:submit.prevent="searchProducts">
			<div class="d-flex flex-wrap">
				<div class="justify-content-between align-items-center flex-grow-1 p-1 pb-0 my-1 mx-1">

					<input v-model="searchQuery.text" type="text" id="search-input" placeholder="эмаль ПФ-115" class="form-control">
				</div>
				<div class="ml-auto p-0 mr-0 my-1">
					<button class="btn btn-primary my-1" :disabled="searchButtonDisabled">Найти!</button>
				</div>
			</div>

			<div class="table-responsive-lg mx-1 p-0">
				<table class="table table-sm table-borderless m-1">
					<thead class="thead-dark text-truncate">
						<tr class="noborder">
							<td class="text-wrap pl-0 pt-0">
								<input id="searchCategory" v-model="searchQuery.category" class="form-control m-0" style="width:100%" placeholder="Запчасти и расходные материалы"/>
							</td>
							<td class="text-wrap pt-0">
								<input id="searchCode" v-model="searchQuery.code" class="form-control m-0" style="width:100%" placeholder="ПФ-115 пф 115"/>
							</td>
							<td class="text-wrap pt-0">
								<input id="searchName" v-model="searchQuery.name" class="form-control m-0" style="width:100%" placeholder="краска эмаль пф 115 черная"/>
							</td>
							<td class="text-wrap pr-0 pt-0">
								<input id="searchProperty" v-model="searchQuery.property" class="form-control m-0" style="width:100%" placeholder="автохимия"/>
							</td>
						</tr>
						<tr class="">
							<th class="text-wrap">Категория</th>
							<th class="text-wrap">Артикул</th>
							<th class="text-wrap">Название</th>
							<th class="text-wrap">Свойства</th>
						</tr>
					</thead>
					<tbody v-if="!loading">
						<tr class="" v-for="(product, index) in searchResults" :key="index">

							<td class="text-wrap"> {{product.categories}}</td>
							<td class="text-wrap"> {{product.code}} </td>
							<td class="text-wrap"> <a :href="'/catalog/product/'+product.id">{{product.name}}</a> </td>
							<td class="text-wrap"> {{product.properties}} </td>

						</tr>
					</tbody>
				</table>
			</div>

		</form>

		<div v-if="loading">
			<div class="mt-1" align="center">
				<div class="spinner-border mt-1" role="status">
					<span class="sr-only">Loading...</span>
				</div>
			</div>
		</div>
		<div v-else>
			<div v-if="totalPages > 1" align="center">
				<span class="m-2" v-for="i in Math.min(page, 2)" :key="i"> <a href="#" @click.prevent="setPage(Math.max(page - 2, 0))"> {{ Math.max(page - 2, 0) + i }} </a> </span>
				<span class="m-2"> {{page + 1}}</span>
				<span class="m-2" v-for="i in Math.min(totalPages - page - 1, 2)" :key="i"> <a href="#" @click.prevent="setPage(page + i)"> {{ page + i + 1 }} </a> </span>
			</div>
		</div>
  </div>
</template>

<style>
.form-control::placeholder { /* Chrome, Firefox, Opera, Safari 10.1+ */
            color: #AAAAAA;
            opacity: 1; /* Firefox */
}

.form-control:-ms-input-placeholder { /* Internet Explorer 10-11 */
            color: #AAAAAA;
}

.form-control::-ms-input-placeholder { /* Microsoft Edge */
            color: #AAAAAA;
 }
</style>

<script>
import axios from 'axios';
axios.defaults.baseURL = '/crutch';

export default {
  name: 'App',

  data() { return {
    loading:false,
    error_message:"",
    searchQuery: {},
		currentSearchQuery: {},
		searchResults:[],
		total:0,
  } },
  computed: {
		searchButtonDisabled() {
			return this.loading || !(
					this.searchQuery.text != null && this.searchQuery.text.length > 2 || 
					this.searchQuery.category != null && this.searchQuery.category.length > 2 || 
					this.searchQuery.code != null && this.searchQuery.code.length > 2 || 
					this.searchQuery.property != null && this.searchQuery.property.length > 2 || 
					this.searchQuery.name != null && this.searchQuery.name.length > 2  
			) ;
		},
  },
  methods: {
    searchProducts() {
      this.loading = true;
			console.log(this.searchQuery);
      axios({
				method: "post", 
				url: "/api/searchProducts",
				data: this.searchQuery
			})      
      .then(res => {
				this.error_message = "";
        this.searchResults = res.data.results;
        this.page = res.data.page;
        this.totalPages = res.data.totalPages;

				this.currentSearchQuery = Object.assign({}, this.searchQuery); 
        this.loading = false;
      })
      .catch(error => {
        this.error_message = "Ошибка во время поиска: " + this.getAxiosErrorMessage(error);
				this.searchResults = {};
        this.loading = false;
      })
    },
		setPage(page) {
      this.loading = true;
			this.currentSearchQuery.page = page;
			console.log(this.currentSearchQuery);
      axios({
				method: "post", 
				url: "/api/searchProducts",
				data: this.currentSearchQuery
			})      
      .then(res => {
				this.error_message = "";
        this.searchResults = res.data.results;
        this.page = res.data.page;
        this.totalPages = res.data.totalPages;

        this.loading = false;
      })
      .catch(error => {
        this.error_message = "Ошибка во время поиска: " + this.getAxiosErrorMessage(error);
				this.searchResults = {};
        this.loading = false;
      })
		},
    getAxiosErrorMessage : function(error) {
      if (error.response != null && error.response.data != null && error.response.data != "") {
        return error.response.data;

      } else {
        return error;
      }
    },
  }
}
</script>
