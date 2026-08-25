import { createRouter, createWebHashHistory } from 'vue-router'

const HomeView = () => import('./views/HomeView.vue')
const ChartsView = () => import('./views/ChartsView.vue')
const ListView = () => import('./views/ListView.vue')
const SearchView = () => import('./views/SearchView.vue')
const DetailView = () => import('./views/DetailView.vue')
const PlayerView = () => import('./views/PlayerView.vue')
const WatchView = () => import('./views/WatchView.vue')
const RewardsView = () => import('./views/RewardsView.vue')
const PlaylistView = () => import('./views/PlaylistView.vue')
const ProfileView = () => import('./views/ProfileView.vue')
const WalletView = () => import('./views/WalletView.vue')
const RechargeView = () => import('./views/RechargeView.vue')
const LoginView = () => import('./views/LoginView.vue')

export const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView, meta: { tab: 'home' } },
    { path: '/charts', name: 'charts', component: ChartsView, meta: { tab: 'home' } },
    { path: '/list', name: 'list', component: ListView, meta: { tab: 'home' } },
    { path: '/search', name: 'search', component: SearchView },
    { path: '/drama/:id', name: 'drama', component: DetailView },
    { path: '/player/:dramaId/:episodeId', name: 'player', component: PlayerView },
    { path: '/watch', name: 'watch', component: WatchView, meta: { tab: 'watch' } },
    { path: '/rewards', name: 'rewards', component: RewardsView, meta: { tab: 'rewards' } },
    { path: '/playlist', name: 'playlist', component: PlaylistView, meta: { tab: 'playlist' } },
    { path: '/profile', name: 'profile', component: ProfileView, meta: { tab: 'profile' } },
    { path: '/wallet', name: 'wallet', component: WalletView },
    { path: '/recharge', name: 'recharge', component: RechargeView },
    { path: '/login', name: 'login', component: LoginView },
    { path: '/history', name: 'history', component: () => import('./views/HistoryView.vue') },
    { path: '/privacy', name: 'privacy', component: () => import('./views/LegalView.vue') },
    { path: '/terms', name: 'terms', component: () => import('./views/LegalView.vue') },
  ],
})
