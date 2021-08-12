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
				<div class="justify-content-between align-items-center flex-grow-1 p-1 pb-2 my-1 mx-1">

					<input v-model="searchQuery" type="text" id="search-input" placeholder="эмаль ПФ-115" class="form-control">
				</div>
				<div class="ml-auto p-0 mr-1 my-1">
					<button class="btn btn-primary m-1" :disabled="loading">Найти!</button>
				</div>
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
		</div>
  </div>
</template>

<script>
import axios from 'axios';

export default {
  name: 'App',

  data() { return {
    loading:false,
    error_message:"",
    searchQuery: '',
  } },

  methods: {
    searchProducts() {
      this.loading = true;
      axios.post("/api/searchProducts", {
        searchQuery: this.searchQuery
      })      
      .then(res => {
        this.searchResults = res.data;
        this.loading = false;
      })
      .catch(error => {
        this.error_message = "Ошибка во время поиска: " + this.getAxiosErrorMessage(error);
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
