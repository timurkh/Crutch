<template>
	<div v-if="error_message.length > 0" class="alert alert-danger m-1 p-1 text-wrap text-break" role="alert">
		{{ error_message }}
	</div>


	<form class="form-horizontal" @submit.prevent>
		<div class="d-flex flex-wrap m-1 mb-2 justify-content-end">
			<div class="form-inline font-weight-bold mb-1 mr-5 p-0 mr-sm-0">
				{{summary}}
			</div>
			<div class="form-inline flex-grow-1 m-1 p-0">
					<input type="text my-0" class="flex-fill form-control" v-model="filter.text" @change="onChange"/>
			</div>
			<div class="form-inline align-self-center py-0 m-0" style="min-width:120px;">
				<VueMultiselect v-model="filter.role" 
					:options="roles" 
					:multiple="false" 
					:searchable="false" 
					track-by="id"
					label="name"
					:close-on-select="true" 
					:show-labels="false"
					@update:model-value="onChange"
					placeholder="Роль">
				</VueMultiselect>
			</div>
			<div class="form-inline form-check m-1 p-0">
				<input class="" type="checkbox" value="" id="haveOrders" v-model="filter.haveOrders" @change="onChange">
				<label class="form-check-label m-1" for="haveOrders">
					Есть заказы
				</label>
			</div>
			<div class="form-inline m-1 d-none d-sm-block">
				<div class="d-flex justify-center items-center ">
					<label for="filterStart" class="mr-1 ml-0 my-1">с</label> 
					<input id="filterStart" class="form-control" v-model="filterStart" type="date" @change="onChange"/>
					<label for="filterEnd" class="m-1">по</label> 
					<input id="filterEnd" class="form-control" v-model="filterEnd" type="date" @change="onChange"/>
				</div>
			</div>
			<div class="d-flex form-inline mr-1 dropleft">
				<button class="btn btn-secondary dropdown-toggle" type="button" id="dropdownMenuButton" data-toggle="dropdown">
					Выбрать
				</button>
				<div class="dropdown-menu" aria-labelledby="dropdownMenuButton">
					<a class="dropdown-item" href="#" @click="today">Сегодня</a>
					<a class="dropdown-item" href="#" @click="yesterday">Вчера</a>
					<a class="dropdown-item" href="#" @click="thisWeek">Эта неделя</a>
					<a class="dropdown-item" href="#" @click="prevWeek">Предыдущая неделя</a>
					<a class="dropdown-item" href="#" @click="thisMonth">Этот месяц</a>
					<a class="dropdown-item" href="#" @click="prevMonth">Предыдущий месяц</a>
				</div>
			</div>
			<div class="d-flex form-inline">
				<button type="button" class="btn btn-success px-1" @click="exportExcel" :disabled="gettingExcel">
					<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-file-excel" viewBox="0 0 16 16">
<path d="M5.18 4.616a.5.5 0 0 1 .704.064L8 7.219l2.116-2.54a.5.5 0 1 1 .768.641L8.651 8l2.233 2.68a.5.5 0 0 1-.768.64L8 8.781l-2.116 2.54a.5.5 0 0 1-.768-.641L7.349 8 5.116 5.32a.5.5 0 0 1 .064-.704z"></path>
<path d="M4 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H4zm0 1h8a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1z"></path>
</svg>
				</button>
			</div>
		</div>
	</form>


	<div class="table-responsive-lg p-0 pr-1 mr-1">
		<table id="ordersTable" class="table table-sm m-1 text-xsmall" ref="ordersTable">
			<thead class="thead-dark text-truncate">
				<tr v-if="$windowWidth > 550" class="d-flex text-wrap text-break">
					<th class="col-4 text-left">
						<tr class="d-flex table-borderless m-0 p-0">
							<td class="col-1 text-left m-0 p-0"> # </td>
							<td class="col-5 text-left m-0 p-0">Название</td>
							<td class="col-3 text-left m-0 p-0">ИНН</td>
							<td class="col-3 text-left m-0 p-0">КПП</td>
						</tr>
					</th>
					<th class="col-3 text-left">Адрес</th>
					<th class="col-1">Роль</th>
					<th class="col-1" role="button" @click="switchSorting('user_count')">
						<div class="d-flex flex-row">
							<div>Пользователей</div>
							<div v-if="(sortingColumn==='user_count' && sortingDirection==='up')" class="m-1"><i class="fas fa-sort-up"></i></div>
							<div v-if="(sortingColumn==='user_count' && sortingDirection==='down')" class="m-1"><i class="fas fa-sort-down"></i></div>
						</div>
					</th>
					<th class="col-3 text-left">
						<tr class="d-flex table-borderless m-0 p-0">
							<td class="col-4 m-0 p-0" role="button" @click="switchSorting('date_joined')">
								<div class="d-flex flex-row">
									<div>Присоединился</div>
									<div v-if="(sortingColumn==='date_joined' && sortingDirection==='up')" class="m-1"><i class="fas fa-sort-up"></i></div>
									<div v-if="(sortingColumn==='date_joined' && sortingDirection==='down')" class="m-1"><i class="fas fa-sort-down"></i></div>
								</div>
							</td>
							<td class="col-4 m-0 p-0" role="button" @click="switchSorting('last_login')">
								<div class="d-flex flex-row">
									<div>Появлялся</div>
									<div v-if="(sortingColumn==='last_login' && sortingDirection==='up')" class="m-1"><i class="fas fa-sort-up"></i></div>
									<div v-if="(sortingColumn==='last_login' && sortingDirection==='down')" class="m-1"><i class="fas fa-sort-down"></i></div>
								</div>
							</td>
							<td class="col-4 m-0 p-0" role="button" @click="switchSorting('order_count')">
								<div class="d-flex flex-row">
									<div>Заказов</div>
									<div v-if="(sortingColumn==='order_count' && sortingDirection==='up')" class="m-1"><i class="fas fa-sort-up"></i></div>
									<div v-if="(sortingColumn==='order_count' && sortingDirection==='down')" class="m-1"><i class="fas fa-sort-down"></i></div>
								</div>
							</td>
						</tr>
					</th>
				</tr>
				<tr v-else class="d-flex text-wrap text-break">
					<th class="col-1 text-left"> # </th>
					<th class="col-3 text-left">Название</th>
					<th class="col-2">Роль</th>
					<th class="col-2" role="button" @click="switchSorting('user_count')">
						<div class="d-flex flex-row">
							<div>Пользователей</div>
							<div v-if="(sortingColumn==='user_count' && sortingDirection==='up')" class="m-1"><i class="fas fa-sort-up"></i></div>
							<div v-if="(sortingColumn==='user_count' && sortingDirection==='down')" class="m-1"><i class="fas fa-sort-down"></i></div>
						</div>
					</th>
					<th class="col-2" role="button" @click="switchSorting('date_joined')">
						<div class="d-flex flex-row">
							<div>Присоединился</div>
							<div v-if="(sortingColumn==='date_joined' && sortingDirection==='up')" class="m-1"><i class="fas fa-sort-up"></i></div>
							<div v-if="(sortingColumn==='date_joined' && sortingDirection==='down')" class="m-1"><i class="fas fa-sort-down"></i></div>
						</div>
					</th>
					<th class="col-2" role="button" @click="switchSorting('order_count')">
						<div class="d-flex flex-row">
							<div>Заказов</div>
							<div v-if="(sortingColumn==='order_count' && sortingDirection==='up')" class="m-1"><i class="fas fa-sort-up"></i></div>
							<div v-if="(sortingColumn==='order_count' && sortingDirection==='down')" class="m-1"><i class="fas fa-sort-down"></i></div>
						</div>
					</th>
				</tr>
			</thead>
			<tbody v-if="!loading"> 
				<div v-for="(cp, index) in counterparts" :key="index">
					<tr v-if="$windowWidth > 550" class="d-flex text-wrap text-break" :id="cp.id">
						<td class="col-4 text-left" :id="cp.id">
							<tr class="d-flex table-borderless m-0 p-0" :id="cp.id">
								<td class="col-1 text-left m-0 p-0"> {{cp.id}} </td>
								<td class="col-5 text-left m-0 p-0">{{cp.name}}</td>
								<td class="col-3 text-left m-0 p-0">{{cp.inn}}</td>
								<td class="col-3 text-left m-0 p-0">{{cp.kpp}}</td>
							</tr>
						</td>
						<td class="col-3">{{cp.address}}</td>
						<td class="col-1">{{cp.role}}</td>
						<td class="col-1">{{cp.user_count}}</td>
						<td class="col-3 text-left" :id="cp.id">
							<tr class="d-flex table-borderless m-0 p-0" :id="cp.id">
								<td class="col-4 m-0 p-0">{{formatDateOnly(cp.date_joined)}}</td>
								<td class="col-4 m-0 p-0">{{formatDateOnly(cp.last_login)}}</td>
								<td class="col-4 m-0 p-0">{{cp.order_count}}</td>
							</tr>
						</td>
					</tr>
					<tr v-else class="d-flex text-wrap text-break" :id="cp.id">
						<td class="col-1 text-left"> {{cp.id}} </td>
						<td class="col-3 text-left">{{cp.name}}</td>
						<td class="col-2">{{cp.role}}</td>
						<td class="col-2">{{cp.user_count}}</td>
						<td class="col-2">{{formatDateOnly(cp.date_joined)}}</td>
						<td class="col-2">{{cp.order_count}}</td>
					</tr>
				</div>
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
			loading:true,
			gettingExcel:false,
			error_message:"",
			counterparts:[],
			sortingColumn:"",
			sortingDirection:"",
			filter: {
        start: new Date(2021, 0, 1),
        end: new Date(),
				role: null,
				text: "",
      },
			filterNormalized: {},
			roles: [
				{name:'Все', id: "0"}, 
				{name:'Покупатель', id: "79"}, 
				{name:'Поставщик', id: "186"}, 
			],
		} 
	},
	computed: {
		summary: function() {
			var ret = ""
			if(this.totalSellers > 0) {
				ret += this.totalSellers
				ret += " "
				ret += this.declOfNum(this.totalSellers, ["поставщик", "поставщика", "поставщиков"])

				if(this.totalBuers > 0)
					ret += ", "
			}
			
			if(this.totalBuers > 0) {
				ret += this.totalBuyers
				ret += " "
				ret += this.declOfNum(this.totalBuyers, ["покупатель", "покупателя", "покупателей"])
			}
			return ret
		},
    totalSellers: function() {
			return this.counterparts.filter(c => c.role_id === 186).length
    },
    totalBuyers: function() {
			return this.counterparts.filter(c => c.role_id === 79).length
    },
		filterStart: {
			get() {
				return moment(this.filter.start).format('YYYY-MM-DD')
			},
			set(d) {
				this.filter.start = moment(d).startOf('day').toDate()
			}
		},
		filterEnd: {
			get() {
				return moment(this.filter.end).format('YYYY-MM-DD')
			},
			set(d) {
				this.filter.end = moment(d).endOf('day').toDate()
			}
		}

  },
	created() {
		window.onpopstate = this.onPopState
		if(!this.user.admin) {
			if(!this.user.can_read_buyers)
				this.roles[1].$isDisabled = true
			if(!this.user.can_read_sellers)
				this.roles[2].$isDisabled = true
		}
		setTimeout(this.getCounterparts, 200)
	},
  methods: {
		sortingEnabled() {
			return this.sortingDirection === 'up' || this.sortingDirection === 'down'
		},
		switchSorting(column) {
			if (this.sortingColumn != column) {
				this.sortingColumn = column
				this.sortingDirection = "up"
			} else {

				if (this.sortingDirection == "up") {
					this.sortingDirection = "down"
				}
				else if (this.sortingDirection == "down") {
					this.sortingDirection = ""
				}
				else {
					this.sortingDirection = "up"
				}
			}
			this.sortResults()
		},
		sortResults() {
			if (this.sortingDirection == "down") {
				this.counterparts.sort((b, a) => (a[this.sortingColumn] > b[this.sortingColumn]) ? 1 : ((b[this.sortingColumn] > a[this.sortingColumn]) ? -1 : 0))
			}
			else if (this.sortingDirection == "up") {
				this.counterparts.sort((a, b) => (a[this.sortingColumn] > b[this.sortingColumn]) ? 1 : ((b[this.sortingColumn] > a[this.sortingColumn]) ? -1 : 0))
			}
			else {
				this.counterparts.sort((a, b) => (a.id > b.id) ? 1 : ((b.id > a.id) ? -1 : 0))
			}
		},
		formatDateOnly(d) {
			if(d != null)
				return moment(d).format('YYYY-MM-DD')
			return ""
		},
		formatDate(d) {
			if(d != null)
				return moment(d).format('YYYY-MM-DD HH:mm:SS')
			return ""
		},
		getFilterText() {
			if(this.filter.text != null)
				return this.filter.text
			return ""
		},
		onPopState(e) {
			if(e.state != null) {
				this.filter = e.state
				return this.getCounterparts()
			}
		},
		onChange() {

			history.pushState( JSON.parse(JSON.stringify(this.filter)), this.getFilterText(), "/" + process.env.VUE_APP_BASE_URL + "/counterparts?" + this.getFilterText())  
			return this.getCounterparts()
		},
		getCounterparts() {
			this.filterNormalized = Object.assign({}, this.filter)
			if (this.filter.role != null) 
				this.filterNormalized.role =  this.filter.role.id
			if (! (this.filter.start instanceof Date) || isNaN(this.filter.start)) 
				this.filterNormalized.start = null
			if (! (this.filter.end instanceof Date) || isNaN(this.filter.end)) 
				this.filterNormalized.end = new Date()

			return axios({
				method: "GET", 
				url: "/methods/counterparts",
				params: this.filterNormalized,
			})      
			.then(res => {
				this.error_message = ""
				this.counterparts = res.data.counterparts
				if (this.sortingEnabled())
					this.sortResults()
				this.loading = false
			})
			.catch(error => {
				this.error_message = "Не удалось загрузить список заказов: " + this.getAxiosErrorMessage(error)
				this.counterparts = []
				this.loading = false
			})
		},
		exportExcel() {
			this.gettingExcel = true

			axios({
				method: "GET",
				url: "/methods/counterparts/excel", 
				responseType: "blob", 
				params: this.filterNormalized,
			}).then(res => {
				const blob = new Blob([res.data], {type: 'application/xlsx'})
				const url = URL.createObjectURL(blob)
				const link = document.createElement('a')
				link.href = url;
				link.download = 'контрагенты.xlsx'
				link.click();
				link.remove();
				URL.revokeObjectURL(link.href)
				this.gettingExcel = false
			}).catch(error => {
				this.gettingExcel = false
				this.error_message = "Не удалось загрузить excel со списком контрагентов: " + this.getAxiosErrorMessage(error)
			})
		},
		today() {
			this.filter.start = new Date(moment().startOf("day"))
			this.filter.end = new Date(moment().endOf("day"))
			this.onChange()
		},
		yesterday() {
			this.filter.start = new Date(moment().subtract(1, "days").startOf("day"))
			this.filter.end = new Date(moment().subtract(1, "days").endOf("day"))
			this.onChange()
		},
		thisWeek() {
			this.filter.start = new Date(moment().startOf("week"))
			this.filter.end = new Date(moment().endOf("day"))
			this.onChange()
		},
		prevWeek() {
			this.filter.start = new Date(moment().startOf("week").subtract(7,"days"))
			this.filter.end = new Date(moment().endOf("week").subtract(7, "days"))
			this.onChange()
		},
		thisMonth() {
			this.filter.start = new Date(moment().startOf('month'))
			this.filter.end = new Date(moment().endOf('month'))
			this.onChange()
		},
		prevMonth() {
			this.filter.start = new Date(moment().subtract(1, 'months').startOf('month'))
			this.filter.end = new Date(moment().subtract(1, 'months').endOf('month'))
			this.onChange()
		},
		declOfNum(n, text_forms) {
			n = Math.abs(n) % 100;
			var n1 = n % 10;
			if (n > 10 && n < 20) { return text_forms[2]; }
			if (n1 > 1 && n1 < 5) { return text_forms[1]; }
			if (n1 == 1) { return text_forms[0]; }
			return text_forms[2];
		},
		getInDevMode : function(value) {
			if(process.env.NODE_ENV === 'development') {
				return value;
			}
		},
    getAxiosErrorMessage : function(error) {
      if (error.response != null && error.response.data != null && error.response.data != "") {
        return error.response.data

      } else {
        return error
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
