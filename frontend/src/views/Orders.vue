<template>
	<div v-if="error_message.length > 0" class="alert alert-danger m-1 p-1 text-wrap text-break" role="alert">
		{{ error_message }}
	</div>


	<form class="form-horizontal" @submit.prevent>
		<div class="d-flex flex-wrap m-1 mb-3">
			<div class="form-inline font-weight-bold">
				{{totalOrders}} {{declOfNum(totalOrders, ["заказ", "заказа", "заказов"])}} на {{totalSum}} руб
			</div>
			<div class="form-inline flex-grow-1 mx-1">
					<input type="text" class="flex-fill form-control" />
			</div>
			<div class="form-inline mx-1">
				<div class="d-flex justify-center items-center">
					<input class="form-control" v-model="rangeStart" type="date"/>
					<div class="d-flex align-items-center">&#10132;</div>
					<input class="form-control" v-model="rangeEnd" type="date"/>
				</div>
			</div>
			<div class="d-flex align-items-end form-inline p-0 mr-1 dropleft">
				<button class="btn btn-secondary dropdown-toggle" type="button" id="dropdownMenuButton" data-toggle="dropdown" aria-haspopup="true" aria-expanded="false">
					Выбрать
				</button>
				<div class="dropdown-menu" aria-labelledby="dropdownMenuButton">
					<a class="dropdown-item" href="#" @click="thisWeek">Эта неделя</a>
					<a class="dropdown-item" href="#" @click="prevWeek">Предыдущая неделя</a>
					<a class="dropdown-item" href="#" @click="thisMonth">Этот месяц</a>
					<a class="dropdown-item" href="#" @click="prevMonth">Предыдущий месяц</a>
				</div>
			</div>
			<div class="d-flex align-items-end form-inline p-0">
				<button type="button" class="btn btn-success" @click="exportExcel">
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
				<tr class="d-flex">
					<th class="text-wrap col-1 text-left">
							#
					</th>
					<th class="text-wrap col-7 text-left">
						<tr class="d-flex table-borderless m-0 p-0">
							<td class="col-3 text-left">Закупщик</td>
							<td class="col-3 text-left">Покупатель</td>
							<td class="col-3 text-left">Грузополучатель</td>
							<td class="col-3 text-left">Поставщик</td>
						</tr>
					</th>
					<th class="text-wrap col-1">Cумма</th>
					<th class="text-wrap col-1">Статус</th>
					<th class="text-wrap col-1">Создан</th>
					<th class="text-wrap col-1">Дата поставки</th>
				</tr>
			</thead>
			<tbody> 
				<div v-for="(order, index) in filtered_orders" :key="index">
					<tr data-toggle="collapse" role="button" :data-target="'#order' + index" class="accordion-toggle d-flex" @click="toggleDetails" :id="order.id">
						<td class="text-wrap col-1 text-left">
								{{order.id}} {{order.contractor_number}}
						</td>
						<td class="text-wrap col-7 text-left" :id="order.id">
							<tr class="d-flex table-borderless m-0 p-0" :id="order.id">
								<td class="text-wrap col-3 text-left">{{order.buyer}}</td>
								<td class="text-wrap col-3 text-left">{{order.customer_name}}</td>
								<td class="text-wrap col-3 text-left">{{order.consignee_name}}</td>
								<td class="text-wrap col-3 text-left">{{order.seller_name}}</td>
							</tr>
						</td>
						<td class="text-wrap col-1">{{order.sum}}</td>
						<td class="text-wrap col-1">{{order.status}}</td>
						<td class="text-wrap col-1">{{order.ordered_date}}</td>
						<td class="text-wrap col-1">{{order.shipping_date}}</td>
					</tr>
					<div class="accordian-body collapse" :id="'order' + index"> 
						<th class="d-flex table-borderless">
							<td class="text-wrap col-1"></td>
							<td class="text-wrap col-3 text-left">Товар</td>
							<td class="text-wrap col-1">Количество</td>
							<td class="text-wrap col-2">Цена (без НДС)</td>
							<td class="text-wrap col-1">НДС</td>
							<td class="text-wrap col-2 text-left">Комментарий</td>
							<td class="text-wrap col-1">Склад</td>
						</th>
						<tr class="d-flex table-borderless small" v-for="(oi, index) in order_details[order.id]" :key="index">
							<td class="text-wrap col-1"></td>
							<td class="text-wrap col-3 text-left">{{oi.name}}</td>
							<td class="text-wrap col-1">{{oi.count}}</td>
							<td class="text-wrap col-2">{{oi.price}}</td>
							<td class="text-wrap col-1">{{oi.nds}}</td>
							<td class="text-wrap col-2 text-left">{{oi.comment}}</td>
							<td class="text-wrap col-1">{{oi.warehouse}}</td>
						</tr>
					</div> 
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

<script>
	import axios from 'axios'
	axios.defaults.baseURL = '/' + process.env.VUE_APP_BASE_URL
	import moment from 'moment'


export default {
	name: 'App',
	props: ["user"],
	data() { 
		return {
			loading:false,
			error_message:"",
			orders:[],
			filtered_orders:[],
			order_details:{},
			range: {
        start: new Date(2021, 0, 1),
        end: new Date(),
      },
		} 
	},
	computed: {
    totalOrders: function() {
      return this.filtered_orders.length
    },
    totalSum: function() {
      return Math.round(this.filtered_orders.reduce(function(a, c){return a + Number((c.sum) || 0)}, 0)*100)/100
    },
		rangeStart: {
			get() {
				return moment(this.range.start).format('YYYY-MM-DD')
			},
			set(d) {
				this.range.start = moment(d).toDate()
			}
		},
		rangeEnd: {
			get() {
				return moment(this.range.end).format('YYYY-MM-DD')
			},
			set(d) {
				this.range.end = moment(d).toDate()
			}
		}

  },
	created() {
		axios({
			method: "GET", 
			url: "/methods/orders",
		})      
		.then(res => {
			this.error_message = ""
			this.orders = res.data.orders
			this.filtered_orders = this.orders
			this.loading = false
		})
		.catch(error => {
			this.error_message = "Не удалось загрузить список заказов: " + this.getAxiosErrorMessage(error)
			this.filtered_orders = this.orders = []
			this.loading = false
		})
	},
  methods: {
		exportExcel() {
			axios({
				method: 'GET',
				url: '/methods/orders/excel?1', //your url
				responseType: 'blob', // important
			}).then(res => {
				const blob = new Blob([res.data], {type: 'application/xlsx'})
				const url = URL.createObjectURL(blob)
				const link = document.createElement('a')
				link.href = url;
				link.download = 'заказы.xlsx'
				link.click();
				link.remove();
				URL.revokeObjectURL(link.href)
			}).catch(error => {
				this.error_message = "Не удалось загрузить excel со списком заказов: " + this.getAxiosErrorMessage(error)
			})
		},
		thisWeek() {
			var d = new Date()
			var day = d.getDay() // 0 - Sunday
      var diff = d.getDate() - day; 
			this.range.start = new Date(d.setDate(diff));
			this.range.end = new Date()
			console.log(this.range.end, this.range.start)
		},
		prevWeek() {
			this.thisWeek()
			this.range.end = new Date(this.range.start.getDate()-1)
			this.range.start.setDate(this.range.start.getDate()-7)
			console.log(this.range.end, this.range.start)
		},
		thisMonth() {
			var now = new Date();
			this.range.start = new Date(now.getFullYear(), now.getMonth(), 1);
			this.range.end = new Date()
		},
		prevMonth() {
			var now = new Date();
			this.range.start = new Date(now.getFullYear() - (now.getMonth() > 0 ? 0 : 1), (now.getMonth() - 1 + 12) % 12, 1);
			this.range.end = new Date(now.getFullYear(), now.getMonth(), 0);
		},
		declOfNum(n, text_forms) {
			n = Math.abs(n) % 100;
			var n1 = n % 10;
			if (n > 10 && n < 20) { return text_forms[2]; }
			if (n1 > 1 && n1 < 5) { return text_forms[1]; }
			if (n1 == 1) { return text_forms[0]; }
			return text_forms[2];
		},
		toggleDetails (entry) {
			let order_id = entry.target.parentElement.id
			if (!(order_id in this.order_details)) {
				axios({
					method: "GET", 
					url: "/methods/order/"+order_id,
				})      
				.then(res => {
					this.error_message = ""
					this.order_details[order_id] = res.data
					this.loading = false
				})
				.catch(error => {
					this.error_message = "Не удалось загрузить детали заказа: " + this.getAxiosErrorMessage(error)
					this.loading = false
				})
			}
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
