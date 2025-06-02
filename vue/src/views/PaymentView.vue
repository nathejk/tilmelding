<script setup>
import { ref, computed, onMounted } from 'vue'
import Navigation from '@/components/Navigation.vue'

const props = defineProps({
    reference: {type: String, required: true},
})

onMounted(async () => load())

const payment = ref({})
const loaded = ref(false)

const load = async () => {
    try {
        const response = await fetch("/api/payment/" + props.reference);
        if (!response.ok) {
            throw new Error("HTTP status " + response.status);
        }
        const data = await response.json();
        payment.value = data.payment
        loaded.value = true
    } catch (error) {
        console.log("Failed loading config", error);
    }
}
</script>

<template>
    <Navigation class="dark" />

    <div v-if="loaded" class="container mx-auto">
        <div v-if="payment.status == 'received'">
            <h1 class="text-2xl font-nathejk">Betaling registreret</h1>
            <p>Hvis I har rettelser til tilmeldingen kan I <a :href="payment.returnUrl" class="text-blue-500">rette her</a>.</p>
            <p class="pt-5">Vi ses i mørket...</p>
        </div>
        <div v-else>
            <h1 class="text-2xl font-nathejk">Noget gik galt</h1>
            <p>Betaling fejlede, prøv igen...</p>
        </div>
    </div>
</template>

<style>
</style>
