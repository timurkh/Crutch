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
                <a href="/catalog/" class="nav-link">
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
                <span class="  nav-link-text">{{user.name}}</span>
                <img src="/static/icons/chevron.svg" class="nav-link-img" alt="Иконка стрелки вниз" style="margin-left: 5px;">
              </a>
              <ul class="dropdown-menu userinfo arrow">
                <li class="username">
                  <a href="/profile_settings/user-profile-editor">
                    <div>
                      <h5>{{user.name}}</h5>
                      <small>Логин <span>{{user.email}}</span></small>
                    </div>
                  </a>
                </li>
                <li class="userlinks">
                  <ul class="dropdown-menu">
                    <li id="analytics_menu_item">
                      <a href="/analytics/">
                        <i class="fa fa-line-chart"></i> Аналитика
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
                    <li>
                      <a data-toggle="modal"
                         data-target="#feedback_1"
                         href="#feedback_1">
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
            <li class="dropdown">
              <div style="display: flex; padding: 0 15px;">
                <span class="nav-img-wrapper"><img src="/static/icons/briefcase.svg" class="nav-link-img" alt="Иконка чемодана"></span>
                <span class="hidden-md hidden-xs hidden-sm nav-link-text">ПАО &quot;Северсталь&quot;</span>
              </div>
            </li>
            <li class="dropdown">
              <a id="smallcart" href="/cart/" class="nav-link" v-tooltip="{content:getCartContent, html: true, delay: { show: 100, hide: 500 }}">
                <span class="nav-img-wrapper"><img src="/static/icons/cart.svg" class="nav-link-img" alt="Иконка карты"></span>
                <!-- <span class="hidden-xs hidden-sm hidden-md nav-link-text">Корзина</span> -->
                <span class="badge">1</span>&nbsp;&nbsp;
                <span class="hidden-xs hidden-sm nav-link-text">15 083,33 P</span>
              </a>
            </li>
            <li class="dropdown">
              <a href="/cart/" class="nav-link hasnotifications smallcart-popover">
                <span><i class="fa fa-shopping-cart fa-lg" style="color: rgb(117, 117, 117);"></i></span>
                <span class="badge">5</span>&nbsp;&nbsp;
                <span class="nav-link-text hidden-xs hidden-sm hidden-md">Заказы в корзине</span>
              </a>
            </li>
            <li class="dropdown">
              <a class="go-to-favorite-products-btn nav-link" href="/favorite-products/" title="Избранные товары">
                <span class="nav-img-wrapper"><img src="/static/icons/star.svg" class="nav-link-img" alt="Иконка карты"></span>
                <span class="go-to-favorite-products-btn__count badge hidden"></span>
              </a>
            </li>
            <li class="dropdown">
              <a href="#" title="Список сравнения" id="smallcompare" class="nav-link">
                <span class="nav-img-wrapper"><img src="/static/icons/scales.svg" class="nav-link-img" alt="Иконка весов"></span>
                <span class="badge">1</span>
              </a>
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
                  <input autocomplete="off" class="search input-lg textinput textInput form-control" id="id_query" name="query" placeholder="Поиск товаров" type="text" /> 
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
						<div style="flex:1;">
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
						<div style="display: flex; align-items: center;">
							<input class="form-check-input" v-model="searchQuery.inStockOnly" type="checkbox" value="" id="flexCheckDefault" style="margin-left:10px;">
							<label class="form-check-label" style="margin-left: 4px; margin-bottom:0;" for="flexCheckDefault">
								В наличии
							</label>
						</div>
						<input id="searchCategory" v-model="searchQuery.category" class="search form-control textinput" style="flex:2; margin-left:10px;" placeholder="Категория"/>
						<input id="searchSupplier" v-model="searchQuery.supplier" class="search form-control textinput" style="flex:2; margin-left:10px;" placeholder="Поставщик"/>
					</div>
				</form>
			</div>
    </header>


		<div class="modal fade js-feedback_modal" id="feedback_1">
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
		<div style="margin-top:10px;">
			<div v-if="error_message.length > 0" class="alert alert-danger mx-1 my-2 p-1 text-wrap text-break" role="alert">
				{{ error_message }}
			</div>
			Here goes search output
		</div>
	</div>

</template>

<script>

import axios from 'axios'
axios.defaults.baseURL = '/' + process.env.VUE_APP_BASE_URL
import { ref } from 'vue'
import {VueCookieNext} from 'vue-cookie-next'
import 'v-tooltip/dist/v-tooltip.css'
import VueMultiselect from 'vue-multiselect'

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
			url: "/methods/currentUser"
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
		return { error_message, user, csrfToken, searchQuery }
	},
  methods : {
		getCartContent : function () {
			return axios.get(
					"/api/rest/v1/cart_preview/", {
						headers: {
							'Catalog-Direct-Purchase': '1'
						}
					}
				)      
				.then(res => {
						console.log(res.data)
						return res.data.template
					}
				)
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
.btn-primary {
	color: #fff;
	background-color: #f48450;
	border-color: #f48450;
}

.btn-primary:hover {
   background-color: #f48450ab !important;
}

.v-popper--theme-tooltip .v-popper__inner {
  background: rgb(240, 240, 240);
  color: black;
  border-radius: 16px;
  padding: 5px 10px 4px;
}

input[type="checkbox"] {
  width: 15px;
  height: 15px;
}

</style>
