import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView
    },
    {
      path: '/about',
      name: 'about',
      // route level code-splitting
      // this generates a separate chunk (About.[hash].js) for this route
      // which is lazy-loaded when the route is visited.
      component: () => import('../views/AboutView.vue')
    },
    { path: '/patrulje/:teamId', name: 'patrulje', component: () => import('../views/PatruljeView.vue'), props: true },
    { path: '/klan/:teamId', name: 'klan', component: () => import('../views/KlanView.vue'), props: true },
    { path: '/badut/:userId', name: 'gøgler', component: () => import('../views/StaffView.vue'), props: true },
    { path: '/friend/:userId', name: 'friend', component: () => import('../views/FriendView.vue'), props: true },
    { path: '/verificer', name: 'verify', component: () => import('../views/VerifyView.vue') },
    { path: '/indskrivning/patrulje', component: () => import('../views/IndskrivningView.vue'), props: { teamType: 'patrulje' } },
    { path: '/indskrivning/klan', component: () => import('../views/IndskrivningView.vue'), props: { teamType: 'klan' } },
    { path: '/indskrivning/badut', component: () => import('../views/IndskrivningView.vue'), props: { teamType: 'gøgler' } },
    { path: '/indskrivning/friend', component: () => import('../views/IndskrivningView.vue'), props: { teamType: 'friend' } },
    { path: '/indskrivning/:teamId', component: () => import('../views/IndskrivningView.vue'), props: true },
    { path: '/venteliste', name: 'onhold', component: () => import('../views/VentelisteView.vue') },
    { path: '/tak', name: 'thankyou', component: () => import('../views/ThankyouView.vue') },
    { path: '/betaling/:reference', name: 'payment', component: () => import('../views/PaymentView.vue'), props: true },
  ]
})

export default router
