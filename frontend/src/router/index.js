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
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "about" */ '../views/Orders.vue')
  }
]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes
})

export default router
