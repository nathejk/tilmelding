<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Navigation from '@/components/Navigation.vue'
import Shop from '@/components/Shop.vue'
import FloatLabel from 'primevue/floatlabel';
import Calendar from 'primevue/calendar';
import InputGroup from 'primevue/inputgroup';
import InputGroupAddon from 'primevue/inputgroupaddon';

const props = defineProps({
    userId: {type: String, required: false},
})

const router = useRouter()

class List extends Array {
    totalAmount() {
        return this.reduce((a, b) => a + (b["amount"] || 0), 0);
    }
    sum(field) {
        return this.reduce((a, b) => a + (b[field] || 0), 0);
    }
    group(field) {
        return 0;
    }
}

const config = ref({
    memberPrice: 0,
    tshirtPrice: 175,
});
/*
        { slug: "bandit", label: "Banditter" },
        { slug: "guide", label: "Guide" },
        { slug: "post", label: "Postmandskab" },
        { slug: "logistik", label: "Logistik" },
        { slug: "teknik", label: "Teknisk Tjeneste" },
        { slug: "pr", label: "PR og kommunikation" },
        { slug: "kok", label: "Køkkenhold" },
        { slug: "andet", label: "Andet" },
        */
const departments = {
  'bandit': 'Banditter',
  'guide': 'Guide',
  'post': 'Postmandskab',
  'logistik': 'Logistik',
  'teknik': 'Teknisk Tjeneste',
  'pr': 'PR og kommunikation',
  'kok': 'Køkkenhold',
  'andet': 'Andet'
}
const questions = [
    {slug:'department',title:'I hvilken funktion deltager du?', type:'radio', options:departments, defaults:[]},
    {slug:'diet',title:'Har du brug for vegetarmad?', type:'radio', options:['Ja', 'Mej'], defaults:[]},
];
const staffer = ref({ tshirtSize: '', additionals: {}});
const tshirtCount = computed(() => staffer.value.tshirtSize != '' ? 1 : 0)
const expenses = computed(() => {
  return new List(
    { "text": "Deltager", "count": 1, "unitPrice": config.value.memberPrice, "amount": config.value.memberPrice },
    { "text": "Års t-shirt", "count": tshirtCount, "unitPrice": config.value.tshirtPrice, "amount": tshirtCount.value*config.value.tshirtPrice }
  );
})
const payments = ref(new List());
const payableAmount = computed(() => Math.max(0, expenses.value.sum('amount') - payments.value.sum('amount')));

onMounted(async () => load(props.userId))

const load = async (userId) => {
  try {
    const response = await fetch("/api/personnel/" + userId);
    if (!response.ok) {
        throw new Error("HTTP status " + response.status);
    }
    const data = await response.json();
    config.value = data.config
    staffer.value = data.person
    payments.value = new List(...data.payments.filter(p => p.status != 'requested'));

    console.log("found", data)
  } catch (error) {
    console.log("mounted load failed", error);
  }
}

const member = ref({});

const isLoading = ref(false);
const memberDialog = ref(false);
const deleteMemberDialog = ref(false);
const paymentDialog = ref(false);
const teamSubmitted = ref(false);
const memberSubmitted = ref(false);

const mobilepay = ref('');


const save = async () => {
    const headers = {
        "Content-Type": "application/json",
    }
    try {
        const body = JSON.stringify({
            person: staffer.value,
        })
        console.log('body', body)
        const response = await fetch("/api/personnel/" + props.userId, { method: 'PUT', body: body, headers: headers });
        if (!response.ok) {
            throw new Error("HTTP status " + response.status);
        }
        const data = await response.json();

        if (data.paymentLink && data.paymentLink != "") {
            location.href = data.paymentLink
        } else {
            router.push({ name: 'thankyou' })
        }
    } catch (error) {
        console.log("team signup failed", error);
    }
}

const tshirtSizeLabel = slug => {
    if (slug == "") return ""
    for (const tshirt of config.value.tshirtSizes) {
        if (tshirt.slug == slug) return tshirt.label
    }
    return '';
}
</script>

<template>

    <Navigation class="dark" />

    <div class="container mx-auto">
        <div class="grid grid-cols-2 gap-4">
        <Fieldset class="mt-3" legend="Kontaktinformation">
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <InputText id="team-name" v-model.trim="staffer.name" size="small" class="w-full" required="true" autofocus :invalid="teamSubmitted && !team.name" />
                    <label for="team-name">Deltagernavn</label>
                </FloatLabel>
                <small class="p-error mb-2" v-if="teamSubmitted && !team.name">Klannavn skal indtastes.</small>
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7 select-none">
                    <InputText id="h" v-model.trim="staffer.phone" size="small" class="w-full select-none" required="true" :invalid="teamSubmitted && !team.phone" />
                    <label for="h" class="select-none">T<span>el</span>efon<span>numm</span>er</label>
                </FloatLabel>
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <InputText id="personneail" v-model.trim="staffer.email" type="email" size="small" class="w-full" required="true" :invalid="teamSubmitted && !team.name" />
                    <label for="personneail">E-mailadresse</label>
                </FloatLabel>
                <small class="p-error mb-2" v-if="teamSubmitted && !team.name">Klannavn skal indtastes.</small>
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <InputText id="team-group" v-model.trim="staffer.group" size="small" class="w-full" required="true" :class="{'p-invalid': teamSubmitted && !team.group}" />
                    <label for="team-group">Gruppe og division</label>
                </FloatLabel>
                <small class="p-error mb-2" v-if="teamSubmitted && !member.name">Gruppe og division skal indtastes.</small>
                <!--small id="member-help">Enter your username to reset your password.</small-->
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <Dropdown v-model="staffer.korps" inputId="team-korps" :options="config.korps" optionValue="slug" optionLabel="label" class="filled w-full md:w-14rem" />
                    <label for="team-korps">Spejderkorps</label>
                </FloatLabel>
            </div>
            <p class="mt-5">Medlemsnummer fra medlemsservice.</p>
            <div class="flex flex-col">
                <FloatLabel class="mt-3">
                    <InputText id="medlemsservice" v-model.trim="staffer.number" size="small" class="w-full" />
                    <label for="medlemsservice">Medlemsnummer</label>
                </FloatLabel>
            </div>
        </Fieldset>
        <Fieldset class="mt-3" legend="Hjælper">
            <div v-for="q in questions" class="flex flex-col gap-2 pb-3">
                <span>{{ q.title}}</span>
                <div v-if="q.type=='radio'" v-for="(o, i) in q.options" :key="q.slug + i" class="flex items-center pl-5">
                    <RadioButton v-model="staffer.additionals[q.slug]" :inputId="q.slug + i" :name="q.slug" :value="o" size="small" />
                    <label :for="q.slug + i" class="text-sm pl-2">{{ o }}</label>
                </div>
                <div v-if="q.type=='checkbox'" v-for="(option, i) in q.options" :key="q.slug + i" class="flex items-center pl-5">
                    <Checkbox v-model="staffer.additionals[q.slug]" :inputId="q.slug + i" :name="q.slug" :value="option" size="small" />
                    <label :for="q.slug + i" class="text-sm pl-2">{{ option }}</label>
                </div>
                <div v-if="q.type=='number'" class="flex items-center pl-5">
                    <InputNumber v-model="staffer.additionals[q.slug]" :inputId="q.slug" :min="0" :max="100" fluid size="small" />
                </div>
            </div>
        </Fieldset>
        </div>

        <Shop v-model="staffer.tshirtSize" :options="config.tshirtSizes" /> 

<Fieldset class="mt-3" legend="Betalinger" >
    <div class="card">
        <div class="grid grid-cols-6 gap-4">
          <div class="col-start-4 text-center">Antal</div><div class="text-center">Pris</div><div class="text-center">Total</div>
          <template v-for="expense in expenses">
            <div class="col-start-1 col-span-3">{{ expense.text }}</div><div class="text-right">{{ expense.count }}</div><div class="text-right">{{ expense.unitPrice }},-</div><div class="text-right">{{ expense.amount }},-</div>
          </template>
          <div class="col-start-1 col-span-5 font-bold">I alt</div><div class="font-bold text-right">{{ expenses.sum('amount') }},-</div>
          <Divider class="col-start-1 col-end-7" />
          <div class="col-start-1 col-span-3">Indbetalinger</div><div class="text-center">Dato</div>
          <template v-for="payment in payments">
            <div class="col-start-1 col-span-3">{{ payment.text }}</div><div>{{ payment.date }}</div><div class="col-end-7 text-right">{{ payment.amount }},-</div>
          </template>
          <div class="col-start-1 col-span-5 font-bold">I alt</div><div class="font-bold text-right">{{ payments.sum('amount') }},-</div>
          <Divider class="col-start-1 col-end-7" />
          <div class="col-start-1 col-span-5 font-bold">At betale</div><div class="font-bold text-right">{{ payableAmount }},-</div>
          <Divider class=" col-end-7" />
        </div>
        <p>Deltagerbetalingen bliver ikke refunderet ved afbud uanset grund - vi kan have brugt pengene ud fra en forventning om, at du kommer.</p>
    </div>
</Fieldset>

    <div class="card flex justify-end" >

<Button class="my-5" :label="payableAmount ? 'Gem ændringer og betal' : 'Gem ændringer'" @click="save" />

    </div>
    </div>

    <Dialog v-model:visible="memberDialog" :style="{width: '450px'}" header="Bandit" :modal="true">
        <div class="flex flex-col">
            <FloatLabel class="mt-4">
                <InputText id="member-fullname" v-model.trim="member.name" size="small" class="w-full" required="true" autofocus :invalid="memberSubmitted && !member.name" />
                <label for="member-fullname">Navn</label>
            </FloatLabel>
            <small class="p-error mb-2" v-if="memberSubmitted && !member.name">Navn skal udfyldes.</small>
            <!--small id="member-help">Enter your username to reset your password.</small-->
        </div>
        <div class="flex flex-col"> 
            <FloatLabel class="mt-7">
                <InputText id="member-address" v-model="member.address" size="small" class="w-full" />
                <label for="member-address">Adresse</label>
            </FloatLabel>
        </div>
        <div class="flex flex-col">
            <FloatLabel class="mt-7">
                <InputText id="member-postal" v-model="member.postal" size="small" class="w-full" />
                <label for="member-postal">Postnummer</label>
            </FloatLabel>
        </div>
        <div class="flex flex-col">
            <FloatLabel class="mt-7">
                <InputText type="email" id="member-email" v-model="member.email" size="small" class="w-full" />
                <label for="member-email">E-mail adresse</label>
            </FloatLabel>
        </div>
        <div class="flex flex-col">
            <FloatLabel class="mt-7">
                <InputText id="member-phone" v-model="member.phone" size="small" class="w-full" />
                <label for="member-phone">Telefonnummer</label>
            </FloatLabel>
            <small id="member-phone-help" class="text-slate-400	">Mobilnummer på Nathejk (kun hvis telefon medbringes).</small>
        </div>
        <div class="flex flex-col">
            <FloatLabel class="mt-7">
                <InputSwitch id="member-diet" v-model="member.vegitarian" class="filled" />
                <label for="member-diet">Ønsker vegetarmad (er også gluten og laktosefri)</label>
            </FloatLabel>
        </div>
        <div class="flex flex-col">
            <FloatLabel class="mt-7">
                <Calendar inputId="member-birthday" v-model="member.birthday" size="small" class="w-full filled" dateFormat="yy-mm-dd" showIcon iconDisplay="input" />
                <label for="member-birthday">Fødselsdato</label>
            </FloatLabel>
        </div>
        <div class="flex flex-col">
            <FloatLabel class="mt-7">
                <Dropdown v-model="member.tshirtSize" inputId="member-tshirt" :options="config.tshirtSizes" optionValue="slug" optionLabel="label"  class="w-full filled md:w-14rem" />
                <label for="member-tshirt">Vælg t-shirt</label>
            </FloatLabel>
        </div>

        <template #footer>
            <Button label="Afbryd" icon="pi pi-times" text @click="hideDialog"/>
            <Button label="Gem" icon="pi pi-check" text @click="saveMember" />
        </template>
    </Dialog>

        <Dialog v-model:visible="deleteMemberDialog" :style="{width: '450px'}" header="Bekræft" :modal="true">
            <div>
                <i class="pi pi-exclamation-triangle mr-3" style="font-size: 2rem" />
                <span v-if="member">Er det rigtigt at <b>{{member.name}}</b> ikke skal deltage på Nathejk?</span>
            </div>
            <template #footer>
                <Button label="Nej" icon="pi pi-times" text @click="deleteMemberDialog = false"/>
                <Button label="Ja" icon="pi pi-check" text @click="deleteMember" />
            </template>
        </Dialog>

        <Dialog v-model:visible="paymentDialog" :style="{width: '500px'}" header="Betaling" :modal="true">
            <div class="confirmation-content">
                <p class="m-0 mb-5">Vi sender en SMS med et MobilePay betalingslink på DKK {{ payableAmount }},- til det indtastede telefonnummer.</p>
                <InputGroup size="small">
                    <InputGroupAddon>+45</InputGroupAddon>
                    <InputText size="small" placeholder="Telefonnummer" v-model="mobilepay" />
                </InputGroup>
                <Message severity="warn" :closable="false">registrering af indbetalinger sker manuelt, der kan derfor gå noget tid inden betalingen er registreret.</Message>
            </div>
            <template #footer>
                <Button label="Send betalingslink" icon="pi pi-mobile" text @click="pay" />
            </template>
        </Dialog>

        <!-- BlockUI :blocked="isLoading" :fullScreen="true"></BlockUI >
<ProgressSpinner v-show="isLoading" class="flex items-center justify-center z-[100]" iclass="absolute right-1/2 bottom-1/2 transform translate-x-1/2 translate-y-1/2" lass="overlay"/>
        -->
</template>

<style scoped>
.overlay {
    position:fixed !important;
    top: 0;
    left: 0;
    width: 100% !important;
    height: 100% !important;
    z-index: 100; /* this seems to work for me but may need to be higher*/
}
</style>
