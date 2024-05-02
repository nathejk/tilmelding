import { defineStore } from 'pinia';

import { fetchWrapper, router } from '@/helpers';

const baseUrl = 'https://getwhy.io/';

export const useAuthStore = defineStore({
    id: 'auth',
    state: () => ({
        // initialize state from local storage to enable user to stay logged in
        user: JSON.parse(localStorage.getItem('user')),
        returnUrl: null,
        loginUrl: "https://getwhy.io/auth/login?redirect_url=",
    }),
    actions: {
        async login(username, password) {
            const user = await fetchWrapper.post(`${baseUrl}/authenticate`, { username, password });

            // update pinia state
            this.user = user;

            // store user details and jwt in local storage to keep user logged in between page refreshes
            localStorage.setItem('user', JSON.stringify(user));

            // redirect to previous url or default to home page
            router.push(this.returnUrl || '/');
        },
        logout() {
            this.user = null;
            localStorage.removeItem('user');
            router.push(this.loginUrl);
        }
    }
});
