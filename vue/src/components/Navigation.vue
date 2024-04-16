<script setup>
import { ref, onMounted } from 'vue'
import { RouterLink, RouterView } from 'vue-router'

// reactive state
/*
const items = ref([
    {
        label: 'Home',
        link: '/',
        icon: 'pi pi-home'
    },
    {
        label: 'Patrulje',
        link: '/patrulje',
        icon: 'pi pi-star'
    }
]);
*/
const items = ref([]);
</script>

<template>

  <div class="card">
        <Menubar :model="items" class="rounded-none">
            <template #start>
                <RouterLink to="/"><i class="pi pi-moon text-yellow-400 rotate-75" style="font-size: 2rem"></i><span class="font-nathejk text-slate-100 text-xl">NATHEJK</span></RouterLink>
            </template>
            <template #item="{ item, props, hasSubmenu, root }">
                <RouterLink :to="item.link || ''" class="flex items-center" v-bind="props.action">
                    <span :class="item.icon" />
                    <span class="ml-2">{{ item.label }}</span>
                    <Badge v-if="item.badge" :class="{ 'ml-auto': !root, 'ml-2': root }" :value="item.badge" />
                    <span v-if="item.shortcut" class="ml-auto border border-surface-200 dark:border-surface-500 rounded-md bg-surface-100 dark:bg-surface-800 text-xs p-1">{{ item.shortcut }}</span>
                    <i v-if="hasSubmenu" :class="['pi pi-angle-down text-primary-500 dark:text-primary-400-500 dark:text-primary-400', { 'pi-angle-down ml-2': root, 'pi-angle-right ml-auto': !root }]"></i>
                </RouterLink>
            </template>
            <template #_end>
                <div class="flex items-center gap-2">
                    <InputText placeholder="Search" type="text" class="w-[8rem] sm:w-auto" />
                    <Avatar image="https://primefaces.org/cdn/primevue/images/avatar/amyelsner.png" shape="circle" />
                </div>
            </template>
        </Menubar>
    </div>
</template>

<style scoped>
</style>
