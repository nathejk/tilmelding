<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Navigation from '@/components/Navigation.vue'
import Stepper from 'primevue/stepper';
import StepperPanel from 'primevue/stepperpanel';
import InputIcon from 'primevue/inputicon';
import IconField from 'primevue/iconfield';
import InputOtp from 'primevue/inputotp';

const props = defineProps({
    teamId: {type: String, required: false},
    teamType: {type: String, required: false, default:'spejder'},
})

const router = useRouter()
const contact = ref({})
const step = ref(0);

onMounted(async () => {
    if (!props.teamId) {
        return
    }
    try {
        const response = await fetch("/api/signup/" + props.teamId);
        if (!response.ok) {
            throw new Error("HTTP status " + response.status);
        }
        const data = await response.json();
        contact.value = data.signup
        if (contact.value.email == contact.value.emailPending) {
            step.value = 2
        }
    } catch (error) {
        console.log("mounted load failed", error);
    }
})

const complete = computed(() => !!contact.value.name && /^[^@]+@\w+(\.\w+)+\w$/.test(contact.value.emailPending) && !!contact.value.phonePending)
const emailValidated = computed(() => !!contact.value.email && contact.value.email == contact.value.emailPending)

const signup = async (next) => {
    const headers = {
        "Content-Type": "application/json",
    }
    try {
        const body = JSON.stringify({
            teamId: contact.value.teamId,
            type: props.teamType,
            emailPending: contact.value.emailPending,
            phonePending: contact.value.phonePending,
            name: contact.value.name,
        })
        const response = await fetch("/api/signup", { method: 'POST', body: body, headers: headers });
        if (!response.ok) {
            throw new Error("HTTP status " + response.status);
        }
        const data = await response.json();
        contact.value = data.team
        //router.replace({ name: 'indskrivning', params: { id: data.teamId } })
        router.replace({ path: '/indskrivning/'+ data.team.teamId  })

        //const vendor = data.content
        next()
    } catch (error) {
        console.log("team signup failed", error);
    }
}
const pincode = ref();
const validatePincode = async () => {
    const headers = {
        "Content-Type": "application/json",
    }
    console.log('pincode', pincode)
    try {
        const body = JSON.stringify({
            teamId: contact.value.teamId,
            pincode: contact.value.pincode,
        })
        const response = await fetch("/api/signup/pincode", { method: 'POST', body: body, headers: headers });
        if (!response.ok) {
            throw new Error("HTTP status " + response.status);
        }
        const data = await response.json();
        router.replace({ path: data.team.teamPage })
    } catch (error) {
        console.log("team signup failed", error);
        contact.value.pincode = ''
    }

}
</script>

<template>
    <Navigation class="dark" />

    <div class="container mx-auto py-5 max-w-screen-md">
        <h1 class="font-nathejk text-4xl text-slate-700 pb-5">
            <span v-if="props.teamType == 'spejder'">Tilmelding af spejderpatrulje</span>
            <span v-if="props.teamType == 'senior'">Tilmelding af seniorklan</span>
        </h1>
        <Stepper linear :activeStep="step">
    <StepperPanel header="Kontaktoplysninger">
        <template #content="{ nextCallback }">
            <div class="flex flex-col gap-2 mx-auto" style="min-height: 16rem; max-width: 20rem">
                <div class="mb-4">
                    <IconField>
                        <InputIcon><i class="pi pi-user" /></InputIcon>
                        <InputText v-model="contact.name" type="text" placeholder="Navn" required />
                    </IconField>
                </div>
                <div class="mb-4">
                    <IconField>
                        <InputIcon><i class="pi pi-envelope" /></InputIcon>
                        <InputText v-model="contact.emailPending" type="email" placeholder="E-mailadresse" required />
                    </IconField>
                </div>
                <div class="mb-4">
                    <IconField>
                        <InputIcon><i class="pi pi-mobile" /></InputIcon>
                        <InputText v-model="contact.phonePending" type="text" placeholder="Telefonnummer" required />
                    </IconField>
                </div>
            </div>
            <div class="flex pt-4 justify-end">
                <Button label="Videre" icon="pi pi-arrow-right" iconPos="right" @click="signup(nextCallback)" :disabled="!complete"/>
            </div>
        </template>
    </StepperPanel>
    <StepperPanel header="Bekræft e-mailadresse">
        <template #content="{ prevCallback, nextCallback }">
            <div class="flex flex-col h-[12rem]">
                <p>For at sikre at vi kan komme i kontakt med jer er det vigtigt at den indtastede e-mailadresse er korrekt. Vi har sendt en e-mail med et bekræftigelseslink til:</p>
                <div class="text-center py-5"><p class="text-xl font-bold">{{ contact.emailPending }}</p></div>
                <p>For at komme videre skal du klikke på linket i mailen.</p>
            </div>
            <div class="flex pt-4 justify-between">
                <Button label="Tilbage" severity="secondary" icon="pi pi-arrow-left" @click="prevCallback" />
                <Button label="Videre" icon="pi pi-arrow-right" iconPos="right" @click="nextCallback" :disabled="contact.email != contact.emailPending"/>
            </div>
        </template>
    </StepperPanel>
    <StepperPanel header="Bekræft telefonnummer">
        <template #content="{ prevCallback }">
            <div class="flex flex-col h-[12rem]">
                <p>For at bekræfte dit telefonnummer har vi sendt en SMS med en pinkode - indtast pinkoden herunder</p>
                <div class="mx-auto py-5"><InputOtp v-model="contact.pincode" /></div>
            </div>
            <div class="flex pt-4 justify-between">
                <Button label="Back" severity="secondary" icon="pi pi-arrow-left" @click="prevCallback" />
                <Button label="Videre" icon="pi pi-arrow-right" iconPos="right" @click="validatePincode" />
            </div>
        </template>
    </StepperPanel>
</Stepper>

    </div>
</template>

<style>
</style>
