import { createRouter, createWebHistory } from 'vue-router'
import Search from '../views/Search.vue'

const routes = [
  {
    path: '/',
		alias: '/search',
    name: 'Поиск товаров',
    component: Search
  },
  {
    path: '/orders',
    name: 'Заказы',
    component: () => import('../views/Orders.vue')
  },
  {
    path: '/counterparts',
    name: 'Контрагенты',
    component: () => import('../views/Counterparts.vue')
  }
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes
})

export default router
