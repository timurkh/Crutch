<template>
	<div v-if="error_message.length > 0" class="alert alert-danger m-1 p-1 text-wrap text-break" role="alert">
		{{ error_message }}
	</div>


	<form class="form-horizontal" @submit.prevent>
		<div class="d-flex flex-wrap m-1 mb-2 justify-content-end">
			<div class="form-inline font-weight-bold mb-1	mr-5 mr-sm-0">
				{{totalOrders}} {{declOfNum(totalOrders, ["заказ", "заказа", "заказов"])}}, {{totalSum}} руб
			</div>
			<div class="form-inline flex-grow-1 m-1 p-0">
					<input type="text my-0" class="flex-fill form-control" v-model="filter.text" @change="onChange"/>
			</div>
			<div class="d-flex form-inline py-0 m-0" style="min-width:120px;">
				<VueMultiselect v-model="selectedStatuses" 
					:options="statuses" 
					:multiple="true" 
					:searchable="false" 
					track-by="id"
					label="name"
					:close-on-select="false" 
					:show-labels="false"
					@update:model-value="onChangeSelectedStatuses"
					placeholder="Статус">
				<template v-slot:selection="{ values, isOpen }"><span class="multiselect__single" v-if="values.length > 0 || isOpen">{{ values.length }} {{declOfNum(values.length, ["статус", "статуса", "статусов"])}} {{declOfNum(values.length, ["выбран", "выбрано", "выбрано"])}}</span></template>
				</VueMultiselect>
			</div>
			<div class="form-inline m-1">
				<div class="d-flex justify-center items-center" style="min-width:145px">
					<VueMultiselect 
						id="dateColumn"
						v-model="dateColumn" 
						:options="dateColumns" 
						:multiple="false" 
						:searchable="false" 
						track-by="id"
						label="name"
						:close-on-select="true" 
						:show-labels="false"
						@update:model-value="onChangeDateColumn"
						>
					</VueMultiselect>
				</div>
			</div>
			<div class="form-inline m-1 d-none d-sm-block">
				<div class="d-flex justify-center items-center">
					<label for="filterStart" class="mx-1">с</label> 
					<input id="filterStart" class="form-control" v-model="filterStart" type="date" @change="onChange" :disabled="dateColumn.id === ''" style="width:10em;padding-right:2px;"/> 
					<label for="filterEnd" class="mx-1">по</label> 
					<input id="filterEnd" class="form-control" v-model="filterEnd" type="date" @change="onChange" :disabled="dateColumn.id === ''" style="width:10em;padding-right:2px;"/>
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
			<div class="d-flex form-inline" v-if="user.supplier_id == 0">
				<button type="button" class="btn btn-success px-1" @click="exportExcel" :disabled="gettingExcel">
					<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-file-excel" viewBox="0 0 16 16">
<path d="M5.18 4.616a.5.5 0 0 1 .704.064L8 7.219l2.116-2.54a.5.5 0 1 1 .768.641L8.651 8l2.233 2.68a.5.5 0 0 1-.768.64L8 8.781l-2.116 2.54a.5.5 0 0 1-.768-.641L7.349 8 5.116 5.32a.5.5 0 0 1 .064-.704z"></path>
<path d="M4 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H4zm0 1h8a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1z"></path>
</svg>
				</button>
			</div>
		</div>
	</form>


	<div class="tablv-else e-responsive-lg p-0 pr-1 mr-1">
		<table id="ordersTable" class="table table-sm m-1 text-xsmall" ref="ordersTable">
			<thead class="thead-dark text-truncate">
				<tr v-if="$windowWidth > 550" class="d-flex text-wrap text-break">
					<th class="col-1 text-left">
							#
					</th>
					<th class="col-5 text-left">
						<tr class="d-flex table-borderless m-0 p-0">
							<td class="col-3 text-left m-0 p-0">Закупщик</td>
							<td class="col-3 text-left m-0 p-0">Покупатель</td>
							<td class="col-3 text-left m-0 p-0">Грузополучатель</td>
							<td class="col-3 text-left m-0 p-0">Поставщик</td>
						</tr>
					</th>
					<th class="col-2 text-left">
						<tr class="d-flex table-borderless m-0 p-0">
							<th class="col-4 border-0 m-0 p-0">Cумма</th>
							<th class="col-4 border-0 m-0 p-0">Cумма с НДС</th>
							<th class="col-4 border-0 m-0 p-0">Статус</th>
						</tr>
					</th>
					<th class="col-4 text-left">
						<tr class="d-flex table-borderless">
							<td class="col-3 m-0 p-0">Создан</td>
							<td class="col-3 m-0 p-0">Согласован</td>
							<td class="col-3 m-0 p-0">Дата поставки</td>
							<td class="col-3 m-0 p-0">Доставлен</td>
						</tr>
					</th>
				</tr>
				<tr v-else class="d-flex text-wrap text-break">
					<th class="col-1 text-left"> # </th>
					<th class="col-3 text-left m-0 p-0">Закупщик</th>
					<th class="col-2 text-left m-0 p-0">Покупатель</th>
					<th class="col-2 text-left m-0 p-0">Поставщик</th>
					<th class="col-2 m-0 p-0">Cумма</th>
					<th class="col-2 m-0 p-0">Статус</th>
				</tr>
			</thead>
			<tbody> 
				<div v-for="(order, index) in orders" :key="index">
					<tr v-if="$windowWidth > 550" data-toggle="collapse" role="button" :data-target="'#order' + index" class="d-flex accordion-toggle text-wrap text-break" @click="toggleDetails" :id="order.id">
						<td class="col-1 text-left">
								{{order.id}} {{order.contractor_number}}
						</td>
						<td class="col-5 text-left" :id="order.id">
							<tr class="d-flex table-borderless m-0 p-0" :id="order.id">
								<td class="col-3 text-left m-0 p-0">{{order.buyer}}</td>
								<td class="col-3 text-left m-0 p-0">{{order.customer_name}}</td>
								<td class="col-3 text-left m-0 p-0">{{order.consignee_name}}</td>
								<td class="col-3 text-left m-0 p-0">{{order.seller_name}}</td>
							</tr>
						</td>
						<td class="col-2 text-left" :id="order.id">
							<tr class="d-flex table-borderless m-0 p-0" :id="order.id">
								<td class="col-4 m-0 p-0">{{order.sum}}</td>
								<td class="col-4 m-0 p-0">{{order.sum_with_tax}}</td>
								<td class="col-4 m-0 p-0">{{order.status}}</td>
							</tr>
						</td>
						<td class="col-4 text-left" :id="order.id">
							<tr class="d-flex table-borderless m-0 p-0" :id="order.id">
								<td class="col-3 m-0 p-0">{{formatDate(order.ordered_date)}}</td>
								<td class="col-3 m-0 p-0">{{formatDate(order.closed_date)}}</td>
								<td class="col-3 m-0 p-0">{{formatDateOnly(order.shipping_date_req)}}</td>
								<td class="col-3 m-0 p-0">{{formatDate(order.delivered_date)}}</td>
							</tr>
						</td>
					</tr>
					<tr v-else data-toggle="collapse" role="button" :data-target="'#order' + index" class="accordion-toggle text-wrap text-break" @click="toggleDetails" :id="order.id">
						<td class="col-1 text-left"> <span class="d-none d-sm-block">{{order.id}}</span> {{order.contractor_number}} </td>
						<td class="col-3 text-left m-0 p-0">{{order.buyer}}</td>
						<td class="col-2 text-left m-0 p-0">{{order.customer_name}}</td>
						<td class="col-2 text-left m-0 p-0">{{order.seller_name}}</td>
						<td class="col-2 m-0 p-0">{{order.sum_with_nds}}</td>
						<td class="col-2 m-0 p-0">{{order.status}}</td>
					</tr>
					<div class="accordian-body collapse mx-5" :id="'order' + index"> 
						<th v-if="$windowWidth > 550" class="d-flex text-wrap text-break table-borderless">
							<td class="col-1 text-left">Код</td>
							<td class="col-1 text-left">Артикул</td>
							<td class="col-3 text-left">Товар</td>
							<td class="col-1">Количество</td>
							<td class="col-1">Цена за шт (без НДС)</td>
							<td class="col-1">Сумма (включая НДС)</td>
							<td class="col-1">НДС</td>
							<td class="col-2 text-left">Комментарий</td>
							<td class="col-1">Склад</td>
						</th>
						<th v-else class="d-flex text-wrap text-break table-borderless">
							<td class="col-1"></td>
							<td class="col-5 text-left">Товар</td>
							<td class="col-3">Количество</td>
							<td class="col-3">Сумма</td>
						</th>
						<div v-for="(oi, index) in order_details[order.id]" :key="index">
							<tr v-if="$windowWidth > 550" class="d-flex text-wrap text-break table-borderless small">
								<td class="col-1 text-left">{{oi.product_id}}</td>
								<td class="col-1 text-left">{{oi.code}}</td>
								<td class="col-3 text-left">{{oi.name}}</td>
								<td class="col-1">{{oi.count}}</td>
								<td class="col-1">{{oi.price}}</td>
								<td class="col-1">{{oi.sum_with_tax}}</td>
								<td class="col-1">{{oi.tax}}</td>
								<td class="col-2">{{oi.comment}}</td>
								<td class="col-1">{{oi.warehouse}}</td>
							</tr>
							<tr v-else class="text-wrap text-break table-borderless small">
								<td class="col-1"></td>
								<td class="col-5 text-left">{{oi.name}}</td>
								<td class="col-3">{{oi.count}}</td>
								<td class="col-3">{{oi.price}}</td>
							</tr>
						</div>
						<tr class="table-borderless small d-flex">
							<td class="col-12">
								<hr/>
								<div class="d-flex justify-content-center">
									<button type="button" class="btn btn-primary" @click="exportTTN(order, order_details[order.id])">
										<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-file-word" viewBox="0 0 16 16">
											<path d="M4.879 4.515a.5.5 0 0 1 .606.364l1.036 4.144.997-3.655a.5.5 0 0 1 .964 0l.997 3.655 1.036-4.144a.5.5 0 0 1 .97.242l-1.5 6a.5.5 0 0 1-.967.01L8 7.402l-1.018 3.73a.5.5 0 0 1-.967-.01l-1.5-6a.5.5 0 0 1 .364-.606z"></path>
											<path d="M4 0a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V2a2 2 0 0 0-2-2H4zm0 1h8a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V2a1 1 0 0 1 1-1z"></path>
										</svg>
										Скачать
									</button>
								</div>
							</td>
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

<style src="vue-multiselect/dist/vue-multiselect.css"/>

<script>
	import axios from 'axios'
	axios.defaults.baseURL = '/' + process.env.VUE_APP_BASE_URL

	import moment from 'moment'
	moment.updateLocale('en', {
			week : {
					dow :0	// 0 to 6 sunday to saturday
			}
	});

	import VueMultiselect from 'vue-multiselect'

	// eslint-disable-next-line no-unused-vars
	import { Document, Paragraph, Packer, TextRun, HeadingLevel, Table, TableRow, TableCell, WidthType, SectionType, BorderStyle, convertInchesToTwip, HeightRule } from "docx";
	import { saveAs } from 'file-saver';

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
			loading:true,
			gettingExcel:false,
			error_message:"",
			orders:[],
			totalOrders: 0,
			totalSum: 0,
			order_details:{},
			filter: {
				start: new Date(),
				end: new Date(),
				selectedStatuses: null,
				text: "",
			},
			selectedStatuses: [],
			filterNormalized: {},
			statuses: [
				{name:'Корзина', id: "18"}, 
				{name:'Создан', id: "1"}, 
				{name:'В обработке', id: "2"}, 
				{name:'На согласовании', id: "3"}, 
				{name:'На сборке', id: "10"}, 
				{name:'В пути', id: "21"}, 
				{name:'Доставлен', id: "15"}, 
				{name:'Приёмка', id: "20"}, 
				{name:'Принят', id: "22"}, 
				{name:'Ожидает оплаты', id: "23"}, 
				{name:'Оплачен', id: "25"}, 
				{name:'Завершён', id: "24"}, 
				{name:'Предзаказ', id: "26"}, 
				{name:'Отказ/Не согласован', id: "4"}, 
			],
			dateColumns: [
				{name:'все', id: ""}, 
				{name:'созданные', id: "date_ordered"}, 
				{name:'согласованные', id: "date_closed"}, 
			],
			dateColumn: {}, 
			moreAvailable: true,
		} 
	},
	computed: {
		filterStart: {
			get() {
				return moment(this.filter.start).format('YYYY-MM-DD')
			},
			set(d) {
				if (d == null)
					this.filter.start = null
				else
					this.filter.start = moment(d).startOf('day').toDate()
			}
		},
		filterEnd: {
			get() {
				return moment(this.filter.end).format('YYYY-MM-DD')
			},
			set(d) {
				if (d == null)
					this.filter.end = null
				else 
					this.filter.end = moment(d).endOf('day').toDate()
			}
		}

	},
	created() {
		this.dateColumn = this.dateColumns[1]
		this.filter.dateColumn =	this.dateColumn.id
		this.filter.start = new Date(moment().startOf("week"))
		this.filter.end = new Date(moment().endOf("day"))

		window.onpopstate = this.onPopState
		
		setTimeout(this.getOrders, 200)
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
		formatTimeOnly(d) {
			if(d != null)
				return moment(d).format('HH:mm:SS')
			return ""
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
				return this.getOrders()
			}
		},
		onChangeDateColumn() {
			this.filter.dateColumn =	this.dateColumn.id
			this.onChange()
		},
		onChangeSelectedStatuses() {
			if (this.selectedStatuses != null) 
				this.filter.selectedStatuses =	this.selectedStatuses.map(value => value.id)
			else
				this.filter.selectedStatuses = null
			this.onChange()
		},
		onChange() {
			history.pushState( JSON.parse(JSON.stringify(this.filter)), this.getFilterText(), "/" + process.env.VUE_APP_BASE_URL + "/orders?" + this.getFilterText())	
			return this.getOrders()
		},
		getOrders() {
			this.orders = []
			this.filterNormalized = Object.assign({}, this.filter)
			if (! (this.filter.start instanceof Date) || isNaN(this.filter.start)) 
				this.filterNormalized.start = null
			if (! (this.filter.end instanceof Date) || isNaN(this.filter.end)) 
				this.filterNormalized.end = null
			this.filterNormalized.page = 0
			this.filterNormalized.itemsPerPage = 20

			return this.loadOrders()
		},
		loadOrders() {
			return axios({
				method: "GET", 
				url: "/methods/orders",
				params: this.filterNormalized,
			})			
			.then(res => {
				this.error_message = ""
				this.orders = this.orders.concat(res.data.orders)
				this.moreAvailable = res.data.orders.length == this.filterNormalized.itemsPerPage
				if('count' in	res.data && res.data.count > 0)
					this.totalOrders = res.data.count
				if('sum' in res.data && res.data.sum_with_tax > 0)
					this.totalSum = res.data.sum_with_tax
				this.loading = false
				this.$nextTick(doubleRaf(() => this.onScroll()))
			})
			.catch(error => {
				this.error_message = "Не удалось загрузить список заказов: " + this.getAxiosErrorMessage(error)
				this.orders = []
				this.loading = false
			})
		},
		loadMoreOrders() {
			this.loading = true
			this.filterNormalized.page ++

			this.loadOrders()

		},
		onScroll : function () {
			if (!this.loading && this.moreAvailable) {
				let element = this.$refs.ordersTable
				if ( element != null && element.getBoundingClientRect().bottom < window.innerHeight ) {
					this.loadMoreOrders()
				}
			}
		},
		exportExcel() {
			this.gettingExcel = true

			axios({
				method: "GET",
				url: "/methods/orders/excel", 
				responseType: "blob", 
				params: this.filterNormalized,
			}).then(res => {
				const blob = new Blob([res.data], {type: 'application/xlsx'})
				const url = URL.createObjectURL(blob)
				const link = document.createElement('a')
				link.href = url;
				link.download = 'заказы.xlsx'
				link.click();
				link.remove();
				URL.revokeObjectURL(link.href)
				this.gettingExcel = false
			}).catch(error => {
				this.gettingExcel = false
				this.error_message = "Не удалось загрузить excel со списком заказов: " + this.getAxiosErrorMessage(error)
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
		toggleDetails (entry) {
			let order_id = entry.target.parentElement.id
			if (!(order_id in this.order_details)) {
				axios({
					method: "GET", 
					url: "/methods/orders/"+order_id,
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
		truncate : function(str, n, useWordBoundary) {
			if (str.length <= n) { return str; }
				const subString = str.substr(0, n-1); 
				return (useWordBoundary
					? subString.substr(0, subString.lastIndexOf(" "))
					: subString) + "...";
		},
		// eslint-disable-next-line no-unused-vars
		exportTTN : function(order, orderLines) {

			var content_intro = [
							new Paragraph({
								text: "Приложение ТТН от " + (order.shipped_date ? this.formatDateOnly(order.shipped_date) : "_".repeat(12)),
								heading: HeadingLevel.HEADING_1,
								pageBreakBefore: true,
							}),
							new Paragraph({
								text: "Приложение к ТТН является обязательным документом для оформления приема-передачи груза",
							}),
							new Paragraph({
								text: "",
							}),
						]

	
			var introLines = [
					{name:"Номер заказа: ", value: order.contractor_number},
					{name : "Грузоотправитель (Поставщик)", value : order.seller_name + ", " + order.seller_inn + ", " + order.seller_kpp + ", " + order.seller_address},
					{name : "Грузополучатель (Покупатель): ", value : order.customer_name + ", " + order.customer_inn + ", " + order.customer_kpp + ", " + order.customer_address},
			]

			var introRows = []

			introLines.forEach(line => {
					introRows.push(
						new TableRow({
							children: [
								new TableCell({
									children: [new Paragraph({children: [
												new TextRun({text:line.name, bold: true})
											]
										})
									],
									width: { size: 20, type: WidthType.PERCENTAGE, },
									borders: {
										top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
									},
								}),
								new TableCell({
									children: [new Paragraph({children: [
											new TextRun(line.value.length > 0 ? line.value : "_".repeat(28) ),
										]}
									)],
									borders: {
										top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
									},
								}),
							]
						})
					)
				}
			)

			var introTable = new Table({
				rows : introRows,
				borders: {
					top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
					bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
					left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
					right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
				},
				width: {
						size: 100,
						type: WidthType.PERCENTAGE,
				},
			})

			// eslint-disable-next-line no-unused-vars
			var orderLineRows = [
				new TableRow({
					children: [
						new TableCell({
							children: [new Paragraph({children: [new TextRun({text:"#", bold: true})]})],
						}),
						new TableCell({
							children: [new Paragraph({children: [new TextRun({text: "Код", bold:true})]})],
						}),
						new TableCell({
							children: [new Paragraph({children: [new TextRun({text: "Артикул", bold:true})]})],
						}),
						new TableCell({
							children: [new Paragraph({children: [new TextRun({text: "Наименование", bold:true})]})],
						}),
						new TableCell({
							children: [new Paragraph({children: [new TextRun({text: "Кол-во", bold:true})]})],
						}),
						new TableCell({
							children: [new Paragraph({children: [new TextRun({text: "Цена, руб. коп. за единицу", bold:true})]})],
						}),
						new TableCell({
							children: [new Paragraph({children: [new TextRun({text: "Цена, руб. коп. всего (в том числе НДС)", bold:true})]})],
						}),
					]
				})
			]


			// eslint-disable-next-line no-unused-vars
			var id = 1
			// eslint-disable-next-line no-unused-vars
			orderLines.forEach(oi => { 
				orderLineRows.push(
					new TableRow({
						children: [
							new TableCell({
								children: [new Paragraph((id++).toString())],
							}),
							new TableCell({
								children: [new Paragraph(oi.product_id.toString())],
							}),
							new TableCell({
								children: [new Paragraph(oi.code)],
							}),
							new TableCell({
								children: [new Paragraph(oi.name)],
							}),
							new TableCell({
								children: [new Paragraph(oi.count.toString())],
							}),
							new TableCell({
								children: [new Paragraph(oi.price.toString())],
							}),
							new TableCell({
								children: [new Paragraph(oi.sum_with_tax.toString())],
							}),
						]
					})
				)
			})
					
			var orderLinesTable = new Table({
			rows : orderLineRows,
				width: {
						size: 100,
						type: WidthType.PERCENTAGE,
				},
			})

			var outroLines = [
					{
						col1 : {name : "Адрес поставщика", value : [order.seller_address]},
						col2 : {name : "Адрес покупателя", value : [order.customer_address]},
					},
					{
						col1 : {name: "Адрес отгрузки: ", value : [(orderLines[0].warehouse + (orderLines[0].warehouse_adress? ", " + orderLines[0].warehouse_address : ""))]},
						col2 : {name: "Адрес приёма груза: ", value : [order.consignee_city + ", " + order.consignee_address]},
					},
					{
						col1 : {name : "Дата отгрузки", value : [order.shipped_date? this.formatDateOnly(order.shipped_date) : null]},
						col2 : {name : "Дата приёмки груза", value : [order.accepted_date?this.formatDateOnly(order.accepted_date) : null]},
					},
					{
						col1 : {name : "Время отгрузки", value : [order.shipped_date? this.formatTimeOnly(order.shipped_date) : null]},
						col2 : {name : "Время приёмки груза", value : [order.accepted_date?this.formatTimeOnly(order.accepted_date) : null]},
					},
					{
						col1 : {name : "Поставщик: груз передал", value: [null, ' '.repeat(35) + 'ФИО', null, ' '.repeat(33) + 'подпись']},
						col2 : {name : "Клиент:	груз принял:", value: [null, ' '.repeat(35) + 'ФИО', null, ' '.repeat(33) + 'подпись']},
					},
					{
						col1 : {name : "Перевозчик: груз принял", value: [null, ' '.repeat(35) + 'ФИО', null, ' '.repeat(33) + 'подпись']},
						col2 : {name : "", value: ["", "", "", ""]},
					},
			]

			var outroRows = []

			outroLines.forEach(line => {
				var rowspan = line.col1.value.length;
				for(let i=0; i< rowspan; i++) {
					if (i==0) {
						outroRows.push(
							new TableRow({
		
								children: [
									new TableCell({
										children: [new Paragraph({
												children: [
													new TextRun({text:line.col1.name, bold: true, size: 18})
												],
											})
										],
										width: { size: 20, type: WidthType.PERCENTAGE, },
										borders: {
											top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										},
										margins: {
											bottom: convertInchesToTwip(0.05),
											right: convertInchesToTwip(0.05),
										},
										rowSpan: rowspan,
									}),
									new TableCell({
										children: [new Paragraph({children: [
												new TextRun({text: line.col1.value[i] ? line.col1.value[i] : "_".repeat(28), size: 18} ),
											]}
										)],
										width: { size: 30, type: WidthType.PERCENTAGE, },
										borders: {
											top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										},
										margins: {
											bottom: convertInchesToTwip(0.05),
											left: convertInchesToTwip(0.05),
											right: convertInchesToTwip(0.1),
										},
									}),
									new TableCell({
										children: [new Paragraph({children: [
													new TextRun({text:line.col2.name, bold: true, size: 18})
												]
											})
										],
										width: { size: 20, type: WidthType.PERCENTAGE, },
										borders: {
											top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										},
										margins: {
											bottom: convertInchesToTwip(0.05),
											left: convertInchesToTwip(0.1),
											right: convertInchesToTwip(0.05),
										},
										rowSpan: rowspan,
									}),
									new TableCell({
										children: [new Paragraph({children: [
												new TextRun({ text: line.col2.value[i] || !line.col2.name ? line.col2.value[i] : "_".repeat(28), size: 18 }),
											]}
										)],
										width: { size: 30, type: WidthType.PERCENTAGE, },
										borders: {
											top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										},
										margins: {
											bottom: convertInchesToTwip(0.05),
											left: convertInchesToTwip(0.05),
										},
									}),
								],
							})
							)
						}
						else {
						outroRows.push(
							new TableRow({
		
								children: [
									new TableCell({
										children: [new Paragraph({children: [
												new TextRun({text: line.col1.value[i] ? line.col1.value[i] : "_".repeat(28), size: line.col1.value[i] ? 12 : 18} ),
											]}
										)],
										width: { size: 30, type: WidthType.PERCENTAGE, },
										borders: {
											top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										},
										margins: {
											bottom: convertInchesToTwip(0.05),
											left: convertInchesToTwip(0.05),
											right: convertInchesToTwip(0.1),
										},
									}),
									new TableCell({
										children: [new Paragraph({children: [
												new TextRun({ text: line.col2.value[i] || !line.col2.name ? line.col2.value[i] : "_".repeat(28), size : line.col2.value[i] ? 12 : 18}),
											]}
										)],
										width: { size: 30, type: WidthType.PERCENTAGE, },
										borders: {
											top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
											right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
										},
										margins: {
											bottom: convertInchesToTwip(0.05),
											left: convertInchesToTwip(0.05),
										},
									}),
								],
							})
							)
						}
					}
				}
			)

			var outroTable = new Table({
				rows : outroRows,
				borders: {
					top: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
					bottom: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
					left: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
					right: { style: BorderStyle.NIL, size: 0, color: "FFFFFF", },
				},
				width: {
						size: 100,
						type: WidthType.PERCENTAGE,
				},
			})


			const doc = new Document({
				sections: [
					{
						properties: {},
						children: content_intro,
					},
					{
						properties: {type: SectionType.CONTINUOUS},
						children: [introTable]
					},
					{
						properties: {type: SectionType.CONTINUOUS},
						children: [orderLinesTable]
					},
					{
						properties: {type: SectionType.CONTINUOUS},
						children: [outroTable]
					},
				],
			});

			const mimeType =
				"application/vnd.openxmlformats-officedocument.wordprocessingml.document";
			const fileName = "ttn-" + order.id + ".docx";
			Packer.toBlob(doc).then((blob) => {
				const docblob = blob.slice(0, blob.size, mimeType);
				saveAs(docblob, fileName);
			});
		}
	},
}
</script>
