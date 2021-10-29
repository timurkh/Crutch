<template>
	<header class="navbar navbar-sticky-top" role="banner">
		<div class="col-md-12" style=" background-color: white;">
			<div class="row navbar-container">
				<div class="navbar-header pull-left">
					<div class="logo-wrapper">
						<a href="/" style="display: block;" class="logo_link">

							<div class="logo-text-block ">
								<div class="logo-text-block__img">
									<img src="/static/optima/img/severstal/severstal_logo_image.png" alt="Industrial.Market">
								</div>
								<div class="logo-text-block__text-container">
									<div class="logo-text-block__first-line">Industrial</div>
									<div class="logo-text-block__second-line">.Market</div>
								</div>
							</div>
							<small class="severstal-logo-small-text" style="font-size: 8px;margin-left: 30px; color: black!important; display: block;">При поддержке Северстали</small>
						</a>
					</div>
				</div>

				<div class="collapse navbar-collapse navbar-ex1-collapse pull-left" >
					<ul class="hidden-xs hidden-sm nav navbar-nav">
						<li class="" title="Каталог">
							<a v-if="user.supplier_id>0" href="/catalog_supplier/" class="nav-link">
								<span class="hidden-ntb nav-img-wrapper"><img src="/static/icons/Loading%20Trolley.svg" class="nav-link-img" alt="Иконка телеги"></span>
								<span class="hidden-ntb nav-link-text">Каталог</span>
							</a>
							<a v-else href="/catalog/" class="nav-link">
								<span class="hidden-ntb nav-img-wrapper"><img src="/static/icons/Loading%20Trolley.svg" class="nav-link-img" alt="Иконка телеги"></span>
								<span class="hidden-ntb nav-link-text">Каталог</span>
							</a>
						</li>
						<li class="" title="Поставщики">
							<a href="/suppliers/" class="nav-link">
								<span class="nav-img-wrapper"><img src="/static/icons/Truck.svg" class="nav-link-img" alt="Иконка грузовой машины"></span>
								<span class="hidden-ntb nav-link-text">Поставщики</span>
							</a>
						</li>
						<li class="" title="Новости">
							<a href="/news" class="nav-link">
								<span class="nav-img-wrapper"><img src="/static/icons/Newspaper.svg" class="nav-link-img" alt="Иконка газеты"></span>
								<span class="hidden-ntb nav-link-text"> Новости</span>
							</a>
						</li>
						<li class="" title="Заказы">
							<a href="/orders" class="orders_link nav-link">

								<span class="nav-img-wrapper"><img src="/static/icons/todo_list.svg" class="nav-link-img" alt="Иконка списка дел"></span>
								<span class="hidden-ntb nav-link-text">Заказы</span>
							</a>
						</li>
						<li class="" title="Рекламации">
							<a href="/warranty/user" class="nav-link">
								<span class="nav-img-wrapper"><img src="/static/icons/FileCross.svg" class="nav-link-img" alt="Иконка перечеркнутого списка дел"></span>
								<span class="hidden-ntb nav-link-text"> Рекламации</span>
							</a>
						</li>
						<li class="" title="Чат">
							<a href="/chat/messages/" class="nav-link">
								<span class="nav-img-wrapper"><img src="/static/icons/Comments.svg" class="nav-link-img" alt="Иконка чата"></span>
								<span class="hidden-ntb nav-link-text"> Чат</span>
								<span class="badge unread-message-counter hide umc--js"
											id="unread-message-counter"
											data-value="0">+0</span>
							</a>
						</li>
						<li title="Адреса доставки">
							<a href="/consignees/" class="nav-link">
								<span class="nav-img-wrapper"><img src="/static/icons/Route.svg" class="nav-link-img" alt="Иконка карты"></span>
								<span class="hidden-ntb nav-link-text"> Адреса доставки</span>
							</a>
						</li>
					</ul>
				</div>
				<ul class="nav navbar-nav pull-right right-toolbar ">
					<li class="dropdown js--user-item">
						<a href="#" data-toggle="dropdown" class="dropdown-toggle nav-link" :title="user.name">
							<span class="nav-img-wrapper"><img src="/static/icons/user-icon.svg" class="nav-link-img" alt="Иконка юзера"></span>
							<span class="nav-link-text">{{user.name}}</span>
							<img src="/static/icons/chevron.svg" class="nav-link-img" alt="Иконка стрелки вниз" style="margin-left: 5px;">
						</a>
						<ul class="dropdown-menu userinfo arrow">
							<li class="username">
								<a href="/profile_settings/user-profile-editor">
									<div>
										<h5>{{user.name}}</h5>
										<small>{{user.email}}</small>
									</div>
								</a>
							</li>
							<li class="userlinks">
								<ul class="dropdown-menu">
									<li id="analytics_menu_item">
										<a href="/analytics/">
											<i class="fas fa-chart-line"></i> Аналитика
										</a>
									</li>
									<li>
										<a href="/plugins_settings/">
											<i class="fa fa-cog"></i> Персональные настройки
										</a>
									</li>
									<li>
										<a href="/accounts/portal_contractors/2">
											<i class="fa fa-gears"></i> Юр. лицо
										</a>
									</li>
									<li>
										<a href="/registration/change_password">
											<i class="fa fa-unlock-alt"></i> Сменить пароль
										</a>
									</li>
									<li v-if="user.company_admin">
                    <a href="/company/admin/">
                      <i class="fa fa-users"></i> Админ. компании
                    </a>
                  </li>
									<li>
										<a data-toggle="modal" data-target="#feedback_1" href="#feedback_1">
											<i class="fa fa-ambulance"></i> Тех. поддержка
										</a>
									</li>
									<li class="divider"></li>
									<li>
										<a href="/logout/" class="text-left">
											<i class="fa fa-sign-out"></i> Выйти
										</a>
									</li>
								</ul>
							</li>
						</ul>
					</li>
					<li v-if="currentUserIsCustomer()" class="dropdown">
						<div style="display: flex; padding: 0 15px;">
							<span class="nav-img-wrapper"><img src="/static/icons/briefcase.svg" class="nav-link-img" alt="Иконка чемодана"></span>
							<span class="hidden-md hidden-xs hidden-sm nav-link-text">{{user.contractor}}</span>
						</div>
					</li>
					<li v-if="currentUserIsCustomer()" class="dropdown">
						<Popper v-if="cartContent.itemsCount>0" :hover=true :closeDelay="1000">
							<a id="smallcart" href="/cart/" class="nav-link">
								<span class="nav-img-wrapper"><img src="/static/icons/cart.svg" class="nav-link-img" alt="Иконка карты"></span>
								<span class="badge">{{cartContent.itemsCount}}</span>&nbsp;&nbsp;
								<span class="hidden-xs hidden-sm nav-link-text">{{cartContentSum}}</span>
							</a>
							<template #content>
								<table class="table" style="width:700px;">
									<thead>
										<tr>
											<th>Артикул</th>
											<th>Кол</th>
											<th>Название</th>
										</tr>
									</thead>
									<tr v-for="(ci,i) in cartItems" :key="i">
										<td>{{ci.productCode}}</td>
										<td align="center">{{ci.count}}</td>
										<td><a :href="'/catalog/product/'+ ci.productId">{{ci.productName}}</a></td>
									</tr>
								</table>
							</template>
						</Popper>
						<span v-else id="smallcart" href="/cart/" class="nav-link">
							<span class="nav-img-wrapper"><img src="/static/icons/cart.svg" class="nav-link-img" alt="Иконка карты"></span>
							<span class="hidden-xs hidden-sm hidden-md nav-link-text">Корзина</span>
						</span>
					</li>
					<li v-if="false && currentUserIsCustomer()" class="dropdown">
						<a v-if="false && cartContent.ordersCount>0" href="/cart/" class="nav-link hasnotifications smallcart-popover">
							<span><i class="fa fa-shopping-cart fa-lg" style="color: rgb(117, 117, 117);"></i></span>
							<span class="badge">{{cartContent.ordersCount}}</span>&nbsp;&nbsp;
							<span class="nav-link-text hidden-xs hidden-sm hidden-md">Заказы в корзине</span>
						</a>
						<span v-else class="nav-link hasnotifications smallcart-popover">
							<span><i class="fa fa-shopping-cart fa-lg" style="color: rgb(117, 117, 117);"></i></span>
							<span class="nav-link-text hidden-xs hidden-sm hidden-md">Заказы в корзине</span>
						</span>
					</li>
					<li class="dropdown">
						<a class="go-to-favorite-products-btn nav-link" href="/favorite-products/" title="Избранные товары">
							<span class="nav-img-wrapper"><img src="/static/icons/star.svg" class="nav-link-img" alt="Иконка карты"></span>
							<span class="go-to-favorite-products-btn__count badge hidden"></span>
						</a>
					</li>
					<li class="dropdown">
						<a v-if="user.compareItemsCount>0" :href="'/compare/'+user.compare_list" title="Список сравнения" id="smallcompare" class="nav-link">
							<span class="nav-img-wrapper"><img src="/static/icons/scales.svg" class="nav-link-img" alt="Иконка весов"></span>
							<span class="badge">{{user.compareItemsCount}}</span>
						</a>
						<span v-else title="Список сравнения" id="smallcompare" class="nav-link" style="margin-right:12px;">
							<span class="nav-img-wrapper"><img src="/static/icons/scales.svg" class="nav-link-img" alt="Иконка весов"></span>
						</span>
					</li>


					<li class="hidden-md hidden-lg dropdown">
						<a href="#" data-toggle="dropdown" class="nav-link">
							<span class="nav-img-wrapper"><img src="/static/icons/folded_menu.svg" class="nav-link-img" alt="Иконка меню"></span>
						</a>
						<ul class="dropdown-menu dropdown-menu-left arrow">
							<li class="" title="Каталог">
								<a href="/catalog/" class="nav-link">
									<span class="nav-img-wrapper"><img src="/static/icons/Loading%20Trolley.svg" class="nav-link-img" alt="Иконка телеги"></span>
									<span class="hidden-md  nav-link-text">Каталог</span>
								</a>
							</li>
							<li class="" title="Поставщики">
								<a href="/suppliers/" class="nav-link">
									<span class="nav-img-wrapper"><img src="/static/icons/Truck.svg" class="nav-link-img" alt="Иконка грузовой машины"></span>
									<span class="hidden-md nav-link-text">Поставщики</span>
								</a>
							</li>
							<li class="" title="Новости">
								<a href="/news" class="nav-link">
									<span class="nav-img-wrapper"><img src="/static/icons/Newspaper.svg" class="nav-link-img" alt="Иконка газеты"></span>
									<span class="hidden-md nav-link-text"> Новости</span>
								</a>
							</li>
							<li class="" title="Заказы">
								<a href="/orders" class="orders_link nav-link">

									<span class="nav-img-wrapper"><img src="/static/icons/todo_list.svg" class="nav-link-img" alt="Иконка списка дел"></span>
									<span class="hidden-md nav-link-text">Заказы</span>
								</a>
						</li>
							<li class="" title="Рекламации">
								<a href="/warranty/user" class="nav-link">
									<span class="nav-img-wrapper"><img src="/static/icons/FileCross.svg" class="nav-link-img" alt="Иконка перечеркнутого списка дел"></span>
									<span class="hidden-md nav-link-text"> Рекламации</span>
								</a>
							</li>
							<li class="" title="Чат">
								<a href="/chat/messages/" class="nav-link">
									<span class="nav-img-wrapper"><img src="/static/icons/Comments.svg" class="nav-link-img" alt="Иконка чата"></span>
									<span class="hidden-md nav-link-text"> Чат</span>
									<span class="badge unread-message-counter hide umc--js"
												id="unread-message-counter"
												data-value="0">+0</span>
								</a>
							</li>
							<li title="Адреса доставки">
								<a href="/consignees/" class="nav-link">
									<span class="nav-img-wrapper"><img src="/static/icons/Route.svg" class="nav-link-img" alt="Иконка карты"></span>
									<span class="hidden-md nav-link-text"> Адреса доставки</span>
								</a>
							</li>
						</ul>
					</li>
				</ul>
			</div>
		</div>

		<div class="search-bar col-xs-12">
			<div id="simple-search" style="display: block;">
				<div class="search-box">
					<form v-on:submit.prevent="onSearchSubmit" class="form-inline">
						<div class="form-group" > 
							<div class="input-group"> 
								<input class="search textinput form-control" id="id_query" name="query" placeholder="Поиск товаров" type="text" v-model="searchQuery.text" style="border-bottom-width: 3px;" /> 
								<span class="input-group-btn"> 
									<button  type="submit" class="btn btn-primary btn-lg agora__button-search">
										<i class="fa fa-search"></i> 
										<span class="hidden-xs search-btn-text">Поиск</span>
									</button> 
								</span> 
							</div> 
						</div> 
					</form>
				</div>
			</div>
		</div>

		<div class="col-xs-12" style="background-color: white; padding-top: 10px; padding-bottom: 10px;">
			<form v-on:submit.prevent="onSearchSubmit" class="form-inline">
				<div class="input-group" style="display: flex; flex-direction: row;">
					<div class="form-group" style="display: flex; align-items: center;">
						<input class="form-check-input" v-model="options.showPictures" @change="onChangeShowPictures" type="checkbox" value="" id="flexCheckPictures" style="width:20px;">
						<label class="form-check-label hidden-xxs" style="margin-left: 4px; margin-bottom:0;" for="flexCheckPictures">
							Показывать картинки
						</label>
					</div>
					<div style="flex:1; margin-left:10px">
						<VueMultiselect v-model="searchQuery.city" 
							:options="user.cities" 
							:multiple="false" 
							:searchable="true" 
							track-by="id"
							label="name"
							:close-on-select="true" 
							:show-labels="false"
							placeholder="Город"
							@select="onSearchSubmit">
						</VueMultiselect>
					</div>
					<div style="display: flex; align-items: center;">
						<input class="form-check-input" v-model="searchQuery.inStockOnly" @change="onSearchSubmit" type="checkbox" value="" id="flexCheckDefault" style="margin-left:10px;width:20px;">
						<label class="form-check-label hidden-xxs" style="margin-left: 4px; margin-bottom:0;" for="flexCheckDefault">
							В наличии
						</label>
					</div>
					<input id="searchCategory" type="text" v-model="searchQuery.category" class="search form-control textinput hidden-xs hidden-sm" style="flex:2; margin-left:10px" placeholder="Категория"/>
					<input id="searchSupplier" type="text" v-model="searchQuery.supplier" class="search form-control textinput hidden-xs hidden-sm" style="flex:2; margin-left:10px;" placeholder="Поставщик" :disabled="user.supplier_id>0"/>
					<button  type="submit" class="hidden"/>
				</div>
			</form>
		</div>
  </header>

	<div v-if="sortingEnabled()" class="text-center alert alert-primary text-wrap text-break" style="color:#F48450; margin-bottom:0px; padding-bottom:0px;" role="alert">
		Пока включена сортировка по цене, товары при прокрутке не подгружаются.
	</div>

	<div class="modal fade js-feedback_modal" id="feedback_1" style="position:absolute; top:180px;">
		<div class="modal-dialog">
			<div class="modal-content">
				<div class="modal-header">
					<button type="button" class="close" data-dismiss="modal" aria-hidden="true">&times;</button>
					<h4 class="modal-title">Техническая поддержка</h4>
				</div>

				<form method="post" action="/feedback/send" class="form-horizontal">
					<div class="modal-body">
						<div class="row">
							<input type='hidden' name='csrfmiddlewaretoken' :value="csrfToken" /> 
							<input id="id_type" name="type" type="hidden" value="1" /> 
							<div id="div_id_email" class="form-group">
								<label for="id_email" class="control-label col-sm-4 requiredField"> E-mail<span class="asteriskField">*</span></label>
								<div class="controls col-sm-8 col-md-6">
									<input class="emailinput form-control" id="id_email" maxlength="75" name="email" required="required" type="email" value="ta.khakimianov@severstal.com" /> 
								</div>
							</div>
							<div id="div_id_phone" class="form-group">
								<label for="id_phone" class="control-label col-sm-4"> Телефон </label>
								<div class="controls col-sm-8 col-md-6">
									<input class="textinput textInput form-control" id="id_phone" maxlength="20" name="phone" type="text" /> 
								</div>
							</div>
							<div id="div_id_message" class="form-group">
								<label for="id_message" class="control-label col-sm-4 requiredField"> Текст сообщения<span class="asteriskField">*</span></label>
								<div class="controls col-sm-8 col-md-6">
									<textarea class="textarea form-control" cols="40" id="id_message" name="message" required="required" rows="10"></textarea>
								</div>
							</div>
							<input id="id_user" name="user" type="hidden" value="538" />

						</div>
					</div>
					<div class="modal-footer">
						<button class="btn btn-primary" type="submit">
							<i class="fa fa-ok"></i> Отправить
						</button>
						<button type="button" class="btn btn-default-alt" data-dismiss="modal">
							Закрыть
						</button>
					</div>

				</form>
			</div>
		</div>
	</div>

	<div class="container-fluid">
		<div style="">
			<div v-if="error_message.length > 0" class="alert alert-danger mx-1 my-2 p-1 text-wrap text-break" role="alert">
				{{ error_message }}
			</div>

				<div class="table-responsive-lg p-0 pr-1 mr-1">
					<table class="table table-sm table-borderless" ref="productsTable" style="table-layout: fixed;">
						<thead class="thead-dark text-truncate">
							<tr class="text-wrap">
								
								<th v-if="devMode" class="col-1 col-code">Score</th>
								<th class="col-1 col-category">Категория</th>
								<th class="col-1 col-code">Артикул</th>
								<th class="col-1">Название</th>
								<th class="col-1 col-img-responsive" v-if="options.showPictures">Картинка</th>
								<th class="col-1 hidden-xxs">Описание</th>
								<th class="col-1 col-rest">Остаток</th>
								<th class="col-1 col-price" role="button" @click="switchSorting">
										Цена<br><span style="font-size:xx-small;">без ндс</span>
								</th>
								<th class="col-1 col-add-to-cart">
										<div class="mx-1" v-if="(options.priceSorting==='up')"><i class="fas fa-sort-up"></i></div>
										<div v-if="(options.priceSorting==='down')" class="mx-1"><i class="fas fa-sort-down"></i></div>
								</th>
								<th class="col-1 pr-1 col-category">Поставщик</th>
							</tr>
						</thead>
						<tbody > 
							<tr class="d-flex text-wrap text-break" v-for="(product, index) in searchResults" :key="index">

								<td v-if="devMode" class="col-1 col-code"> {{product.score}}</td>
								<td class="col-1 col-category"> 
									<span v-if="product.category.length==0" style="color:orange"><b>Не задана</b></span>
									<span v-else>{{product.category}}</span> 
								</td>
								<td class="col-1 col-code"> {{product.code}} </td>
								<td class="col-1"> 
										<a  target="_blank" :href="'/catalog/product/'+product.id" >{{product.name}}</a>
								</td>
								<td v-if="options.showPictures" class="col-1">  
										<img v-if="product.image.length>0" :src="'/media/' + product.image" class="img-responsive">
								</td>
								<td class="col-1 hidden-xxs" data-toggle="tooltip" :title="product.description">  {{ truncate(stripHTML(product.description), 100, true)}} </td>
								<td class="col-1 col-rest"> 
									<span v-if="product.rest==0 && product.warehouse_id==0"> <i class="fas fa-exclamation-circle" style="color:orange" title="Наличие не задано"></i></span>
									<span v-else>{{product.rest}}</span> 
								</td>
								<td class="col-1"> {{product.price}} </td>
								<td class="col-1">  
									<button v-if="product.id in cartContent.cartItems" class="btn btn-primary" disabled="true">
										<i class="fa fa-check"></i>
									</button>
									<button v-else class="btn btn-primary" @click="addProductToCart(product)">
										<i class="fa fa-shopping-cart"></i>
									</button>
								</td>
								<td class="col-1 col-category"> {{product.supplier}} </td>

							</tr>
						</tbody>
					</table>
				</div>
		</div>
	</div>

	<div v-if="loading">
		<div class="mt-1" align="center">
			<div class="loader"> </div>
		</div>
	</div>
</template>

<script>

import axios from 'axios'
import { ref, computed } from 'vue'
import {VueCookieNext} from 'vue-cookie-next'
import 'v-tooltip/dist/v-tooltip.css'
import VueMultiselect from 'vue-multiselect'
import 'uri'

var baseUrl = '/' + process.env.VUE_APP_BASE_URL

function doubleRaf (callback) {
	requestAnimationFrame(() => {
		requestAnimationFrame(callback)
	})
}

function loadFromCookie(val, name) {
	if(VueCookieNext.getCookie(name)) {
		var v = VueCookieNext.getCookie(name)

		if(v != null && v != undefined) {
			val[name] = v
		}
	}
}

export default {
  name: 'App',
	components: { VueMultiselect },
	setup() {

		let error_message = ref("")
		let user = ref({
			cities : [],
			name : "",
		}) 
		let csrfToken = ref(VueCookieNext.getCookie("csrftoken"))
		let searchQuery = ref({
			cityId : 0,
		}) 

		axios({
			method: "GET", 
			url: baseUrl + "/methods/current-user"
		})      
		.then(res => {
			user.value = res.data
		})
		.catch(error => {

			error_message.value = "Ошибка во время проверки сессии: " + error.response.data
			console.log(error_message.value);
			
			if (error.response.status == 401) {
				window.location.href = "/login";
			} else if (error.response.status == 403) {
				window.location.href = "/";
			}
		})

		let cartContent = ref("")
		const cartContentSum = computed(()=>{
			if(cartContent.value.totalSum != undefined)
				return cartContent.value.totalSum.toFixed(2)
			return null
		})
		const cartItems = computed(()=>{
			return cartContent.value.cartItems
		})

		let loading = ref(false)
		let currentSearchQuery = ref({})
		let searchResults = ref([])
		let page = ref(0)
		let totalPages = ref(0)
		let devMode = ref(process.env.NODE_ENV  === "development")
		let options = ref({showPictures:true})

		//loadFromCookie(searchQuery.value, "supplier")
		loadFromCookie(searchQuery.value, "inStockOnly")
		loadFromCookie(searchQuery.value, "city")
		loadFromCookie(options.value, "showPictures")

		return {
			loading,
			error_message, 
			user, 
			csrfToken, 
			searchQuery, 
			cartContent,
			currentSearchQuery,
			searchResults,
			page,
			totalPages,
			devMode,
			options,
			cartContentSum,
			cartItems,
		}
	},
	created () {
		window.onpopstate = this.onPopState
		let uri = window.location.href.split('?');
    if(uri.length == 2) {
      let vars = uri[1].split('&');
      let getVars = {};
      let tmp = '';
      vars.forEach(function(v) {
        tmp = v.split('=');
        if(tmp.length == 2)
          getVars[tmp[0]] = decodeURIComponent(tmp[1].replaceAll('+', '%20'));
      });
			if (getVars.query != undefined)
				this.searchQuery.text = getVars.query
    }

		this.updateCartContent()

		if(this.searchQueryNotEmpty())
			this.searchProducts()
	}, 
	mounted() {

		this.$nextTick(function() {
			window.addEventListener('scroll', this.onScroll)
		});
	},
	beforeUnmount() {
		window.removeEventListener('scroll', this.onScroll)
	},  
  methods : {
		updateCartContent() {
			axios({
				method: "GET", 
				url: baseUrl + "/methods/cart-preview"
			})      
			.then(res => {
				this.cartContent = res.data
			})
			.catch(error => {

				this.error_message = "Не удалось загрузить корзину: " + error.response.data
				console.log(this.error_message);
			})
		},
		currentUserIsCustomer() {
			return typeof(this.user.contractor_id) === 'number' && this.user.contractor_id !== 0
		}, 
		saveToCookie(val, name) {
			if(name in val && val[name] !== undefined)
				VueCookieNext.setCookie(name, val[name])
		},
		sortingEnabled() {
			return this.options.priceSorting === 'up' || this.options.priceSorting === 'down'
		},
		switchSorting() {
			if (this.options.priceSorting == "up") {
				this.options.priceSorting = "down"
			}
			else if (this.options.priceSorting == "down") {
				this.options.priceSorting = ""
			}
			else {
				this.options.priceSorting = "up"
			}
			this.sortResults()
		},
		sortResults() {
			if (this.options.priceSorting == "down") {
				this.searchResults.sort((b, a) => (a.price > b.price) ? 1 : ((b.price > a.price) ? -1 : 0))
			}
			else if (this.options.priceSorting == "up") {
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
				history.pushState( JSON.parse(JSON.stringify(this.searchQuery)), this.searchQuery.text, baseUrl + "/search?query=" + this.searchQuery.text)  
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
					url: baseUrl + "/methods/products",
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
		},
		onChangeShowPictures : function() {
			this.saveToCookie(this.options, "showPictures")
		},
		addProductToCart : function(product) {
			var formData = new FormData();
			formData.append("form-TOTAL_FORMS", 1)
			formData.append("form-INITIAL_FORMS", 1)
			formData.append("form-MAX_NUM_FORMS", 1000)
			formData.append("form-0-count", 1)
			formData.append("form-0-warehouse", product.warehouse_id)
			formData.append("form-0-modification", product.modification_id)
			formData.append("form-0-id", product.modification_id)
			axios({
				method: "POST",
				url: "/api/rest/v1/old_api_add_multi_to_cart",
				data: formData,
				headers: { 
						"Accept": "*/*",
						"Content-Type": "application/x-www-form-urlencoded",
						"X-CSRFToken": this.csrfToken, 
						"X-Requested-With": "XMLHttpRequest",
						"Catalog-Direct-Purchase":1,
					},
			})      
			.then(() => {
				this.updateCartContent()
			})
			.catch(res => {
				console.log(res);
			});
		}
  }
}
</script>

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

<style>
@media (max-width:1620px) {
    .hidden-ntb {
        display: none !important;
    }
}

.col-category {
		width:200;
}


.col-code {
	width:150px;
	text-wrap:normal;
	word-wrap:break-word
}

@media (max-width:1199px) {
	.col-category {
    display: none !important;
	}
	.col-code {
    display: none !important;
	}

}


.col-rest {
		width:70px;
}

.col-add-to-cart {
		width:50px;
}

.col-price {
		cursor: pointer;
		width:70px;
}

.col-img-responsive {
		width: 110px;
}

.img-responsive {
		width: 100px;
		height: auto;
}

@media (max-width:500px) {
	.col-rest {
    display: none !important;
	}

	.hidden-xxs {
    display: none !important;
	}

	.img-responsive {
		width: 70px;
		height: auto;
	}
	.col-img-responsive {
		width: 80px;
	}
}

.btn-primary  {
	color: #fff;
	background-color: #f48450;
	border-color: #f48450;
}

.btn-primary:focus  {
	color: #fff;
	background-color: #f48450;
	border-color: #f48450;
}

.btn-primary:hover {
   background-color: #f48450ab !important;
}

.btn-primary[disabled] {
   background-color: #F48450 !important;
}

.btn-success {
    color: #fff;
    background-color: #f48450;
    border-color: #f48450;
}

input[type="checkbox"] {
  width: 15px;
  height: 15px;
	filter: hue-rotate(190deg) saturate(200%);
}

.loader {
  border: 12px solid #f3f3f3; /* Light grey */
  border-top: 12px solid #F48450; /* orange */
  border-radius: 50%;
  width: 80px;
  height: 80px;
  animation: spin 2s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.v-tooltip__content {
  pointer-events: auto;
}

.modal-open {
	overflow: scroll;
}

:root {
    --popper-theme-background-color: rgb(240, 240, 240);
    --popper-theme-background-color-hover: rgb(240, 240, 240);
    --popper-theme-text-color: black;
    --popper-theme-border-width: 0px;
    --popper-theme-border-style: solid;
    --popper-theme-border-radius: 4px;
    --popper-theme-padding: 10px;
  }
</style>
