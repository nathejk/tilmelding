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
    teamId: {type: String, required: false},
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

const config = ref({});
const team = ref({});
const contact = ref({});
const members = ref(new List());
const tshirtCount = computed(() => members.value.filter(v => !v.deleted && v.tshirtSize && v.tshirtSize != "").length)
const expenses = computed(() => {
console.log('tshort', tshirtCount.value*config.value.tshirtPrice, tshirtCount, config.value.tshirtPrice)
  return new List(
    { "text": "Deltagere", "count": members.value.length, "unitPrice": config.value.memberPrice, "amount": members.value.length*config.value.memberPrice },
    { "text": "Års t-shirt", "count": tshirtCount, "unitPrice": config.value.tshirtPrice, "amount": tshirtCount.value*config.value.tshirtPrice }
  );
})
const payments = ref(new List());
const payableAmount = computed(() => Math.max(0, expenses.value.sum('amount') - payments.value.sum('amount')));

onMounted(async () => {
  try {
    const response = await fetch("/api/patrulje/" + props.teamId);
    if (!response.ok) {
        throw new Error("HTTP status " + response.status);
    }
    const data = await response.json();
    config.value = data.config
    team.value = data.team
    contact.value = data.contact
    members.value = new List(...data.members);
    payments.value = new List(...data.payments.filter(p => p.status != 'requested'));

    console.log("found", data)
  } catch (error) {
    console.log("mounted load failed", error);
  }
})

const member = ref({});

const isLoading = ref(false);
const memberDialog = ref(false);
const deleteMemberDialog = ref(false);
const paymentDialog = ref(false);
const teamSubmitted = ref(false);
const memberSubmitted = ref(false);

const confirmDeleteMember = (prod) => {
    member.value = prod;
    deleteMemberDialog.value = true;
};
const editMember = (prod) => {
    member.value = {...prod};
    memberDialog.value = true;
};
const openNew = () => {
    member.value = {};
    memberSubmitted.value = false;
    memberDialog.value = true;
};
const hideDialog = () => {
    memberDialog.value = false;
    memberSubmitted.value = false;
};
const sleep = ms => new Promise(r => setTimeout(r, ms));

const canSave = computed(() => members.value.length >= 3 && members.value.length <= 7)
const save = async () => {
    const headers = {
        "Content-Type": "application/json",
    }
    try {
        const body = JSON.stringify({
            team: team.value,
            contact: contact.value,
            members: members.value,
        })
        console.log('body', body)
        const response = await fetch("/api/patrulje/" + props.teamId, { method: 'PUT', body: body, headers: headers });
        if (!response.ok) {
            throw new Error("HTTP status " + response.status);
        }
        const data = await response.json();
        contact.value = data.team
        //router.replace({ name: 'indskrivning', params: { id: data.teamId } })
        //router.replace({ path: '/indskrivning/'+ data.team.teamId  })

        if (data.paymentLink && data.paymentLink != "") {
            location.href = data.paymentLink
        } else {
            router.push({ name: 'thankyou' })
        }
        //const vendor = data.content
        //next()
    } catch (error) {
        console.log("team signup failed", error);
    }
    //isLoading.value=true
    //await sleep(2000)
    //isLoading.value=false
    //paymentDialog.value = true;
}

const mobilepay = ref('');
const pay = async () => {
    const headers = {
        "Content-Type": "application/json",
    }
    try {
        const body = JSON.stringify({
            phone: mobilepay.value,
            amount: payableAmount.value,
        })
        const response = await fetch("/api/pay/" + props.teamId, { method: 'PUT', body: body, headers: headers });
        if (!response.ok) {
            throw new Error("HTTP status " + response.status);
        }
        const data = await response.json();
        //contact.value = data.team
        //router.replace({ name: 'indskrivning', params: { id: data.teamId } })
        //router.replace({ path: '/indskrivning/'+ data.team.teamId  })

        //if (data.team.status =="HOLD") {
            router.push({ name: 'thankyou' })
        //}

    //paymentDialog.value = true;
        //const vendor = data.content
        //next()
    } catch (error) {
        console.log("team signup failed", error);
    }
}

const saveMember = () => {
    memberSubmitted.value = true;

    if (member.value.name.trim() == '') {
        return
    }
    if (member.value.id) {
        //member.value.inventoryStatus = product.value.inventoryStatus.value ? product.value.inventoryStatus.value : product.value.inventoryStatus;
        members.value[findIndexById(member.value.id)] = member.value;
        //toast.add({severity:'success', summary: 'Successful', detail: 'Product Updated', life: 3000});
    }
    else {
        member.value.id = createId();
        members.value.push(member.value);
    //    toast.add({severity:'success', summary: 'Successful', detail: 'Product Created', life: 3000});
    }
    memberDialog.value = false;
    member.value = { name: '' };
};
const activeMembers = computed(() => members.value.filter(i => !i.deleted))
const deleteMember = () => {
    //members.value = members.value.filter(val => val.id !== member.value.id);
    members.value[findIndexById(member.value.id)].deleted = true
    deleteMemberDialog.value = false;
    member.value = {};
    //toast.add({severity:'success', summary: 'Successful', detail: 'Product Deleted', life: 3000});
};
const findIndexById = (id) => {
    let index = -1;
    for (let i = 0; i < members.value.length; i++) {
        if (members.value[i].id === id) {
            index = i;
            break;
        }
    }

    return index;
};
const createId = () => {
    let id = '';
    var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    for ( var i = 0; i < 5; i++ ) {
        id += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return id;
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
        <Fieldset class="mt-3" legend="Patruljeoplysninger">
            <p class="m-0">Spejderpatruljen skal bestå af mellem 3 og 7 spejdere, Når I tilmelder patruljen skal i derfor mindst tilmelde 3 spejdere, I kan når som helst eftertilmelde ekstra spejdere sålænge patruljen ikke overstiger 7 spejdere.</p>
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <InputText id="team-name" v-model.trim="team.name" size="small" class="w-full" required="true" autofocus :invalid="teamSubmitted && !team.name" />
                    <label for="team-name">Patruljenavn</label>
                </FloatLabel>
                <small class="p-error mb-2" v-if="teamSubmitted && !team.name">Patruljenavn skal indtastes.</small>
                <!--small id="member-help">Enter your username to reset your password.</small-->
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <InputText id="team-group" v-model.trim="team.group" size="small" class="w-full" required="true" :class="{'p-invalid': teamSubmitted && !team.group}" />
                    <label for="team-group">Gruppe og division</label>
                </FloatLabel>
                <small class="p-error mb-2" v-if="teamSubmitted && !member.name">Gruppe og division skal indtastes.</small>
                <!--small id="member-help">Enter your username to reset your password.</small-->
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <Dropdown v-model="team.korps" inputId="team-korps" :options="config.korps" optionValue="slug" optionLabel="label" class="filled w-full md:w-14rem" />
                    <label for="team-korps">Spejderkorps</label>
                </FloatLabel>
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <InputText id="team-liga" v-model.trim="team.liga" size="small" class="w-full" />
                    <label for="team-liga">Adventurespejdliga nummer</label>
                </FloatLabel>
                <small id="team-liga-help">Læs mere om LigaID og tilmeld jer Adventurespejdligaen her: <a href="">adventurespejd.dk</a>.</small>
            </div>
        </Fieldset>

        <Fieldset class="mt-3" legend="Kontaktperson">
            <p class="m-0">Kontaktpersonen er meget vigtig og skal være en person, som kender patruljen godt (fx tropslederen). Nathejks team skal kunne få fat i kontaktpersonen undervejs på løbet, hvis situationen kræver det.</p>
            <div class="flex flex-col">
                <FloatLabel class="mt-7">
                    <InputText id="contact-name" v-model.trim="contact.name" size="small" class="w-full" required="true" :class="{'p-invalid': teamSubmitted && !contact.name}" />
                    <label for="team-name">Navn</label>
                </FloatLabel>
                <small class="p-error mb-2" v-if="teamSubmitted && !contact.name">Kontaktperson skal indtastes.</small>
                <!--small id="member-help">Enter your username to reset your password.</small-->
            </div>
            <div v-if="false" class="flex flex-col">
                <FloatLabel class="mt-7" >
                    <InputText id="contact-address" v-model.trim="contact.address" size="small" class="w-full" />
                    <label for="contact-address">Adresse</label>
                </FloatLabel>
            </div>
            <div v-if="false" class="flex flex-col">
                <FloatLabel class="mt-7" >
                    <InputText id="contact-postal" v-model.trim="contact.postal" size="small" class="w-full" />
                    <label for="contact-postal">Postnummer og by</label>
                </FloatLabel>
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7" >
                    <InputText id="contact-phone" v-model.trim="contact.phone" size="small" class="w-full" />
                    <label for="contact-phone">Telefonnummer</label>
                </FloatLabel>
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7" >
                    <InputText id="contact-email" v-model.trim="contact.email" size="small" class="w-full" />
                    <label for="contact-email">E-mail</label>
                </FloatLabel>
            </div>
            <div class="flex flex-col">
                <FloatLabel class="mt-7" >
                    <InputText id="contact-role" v-model.trim="contact.role" size="small" class="w-full" />
                    <label for="contact-role">Rolle i forhold til patruljen</label>
                </FloatLabel>
            </div>

        </Fieldset>
        </div>

        <Shop /> 

<Fieldset class="mt-3" legend="Spejdere">
    <div class="card">
        <DataTable :value="activeMembers" size="small" tableStyle="min-width: 50rem">
            <Column field="name" header="Navn">
                <template #body="row">
                    <p class="m-0 font-medium">{{ row.data.name }}</p>
                    <p v-if="row.data.email" class="m-0 font-thin"><i class="pi pi-envelope"></i> {{ row.data.email }}</p>
                </template>
            </Column>
            <Column field="address" header="Adresse">
                <template #body="row">
                    <p class="m-0 font-thin">{{ row.data.address }}</p>
                    <p class="m-0 font-thin">{{ row.data.postal }}</p>
                </template>
            </Column>
            <Column field="phone" header="Telefon">
                <template #body="row">
                    <p v-if="row.data.phone" class="m-0 font-thin"><i class="pi pi-mobile"></i> {{ row.data.phone }}</p>
                    <p v-if="row.data.phoneContact" class="m-0 font-thin"><i class="pi pi-phone"></i> {{ row.data.phoneContact }}</p>
                </template>
            </Column>
            <Column field="birthday" header="Fødselsdag"></Column>
            <Column field="tshirt" header="T-Shirt">
                <template #body="row" style="font-size:0.8rem">
                    {{ tshirtSizeLabel(row.data.tshirtSize) }}
                </template>
            </Column>
            <Column style="min-width:3rem">
            <template #body="row">
                <div class="text-end">
                    <Button icon="pi pi-pencil" outlined rounded class="mr-2" @click="editMember(row.data)" />
                    <Button v-if="members.length > config.minMemberCount" icon="pi pi-trash" outlined rounded severity="danger" @click="confirmDeleteMember(row.data)" />
                </div>
                </template>
            </Column>
            <template #footer>
                <div class="text-end	">
                    <Button icon="pi pi-plus" outlined rounded @click="openNew" :disabled="members.length >= config.maxMemberCount" />
                </div>
            </template>
        </DataTable>
    </div>
</Fieldset>
<Fieldset class="mt-3" legend="Betalinger">
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
        <p>Deltagerbetalingen bliver ikke refunderet ved afbud uanset grund - vi kan have brugt pengene ud fra en forventning om, at du kommer. Det er dog helt frem til ganske kort før løbsstart muligt at skifte ud blandt deltagerne. Betalingen bliver naturligvis refunderet, hvis holdet ikke deltager, fordi Nathejks team har besluttet det.</p>
    </div>
</Fieldset>

    <div class="card flex justify-end">

<Button class="my-5" :disabled="!canSave" :label="payableAmount ? 'Gem ændringer og betal' : 'Gem ændringer'" @click="save" />

    </div>
    </div>

    <Dialog v-model:visible="memberDialog" :style="{width: '450px'}" header="Spejder" :modal="true">
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
                <InputText id="member-postal" v-model="member.postalCode" size="small" class="w-full" />
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
                <InputText id="member-phoneContact" v-model="member.phoneContact" size="small" class="w-full" />
                <label for="member-phoneContact">Telefonnummer på pårørende</label>
            </FloatLabel>
            <small id="member-phoneContact-help" class="text-slate-400">Nathejk skal kunne kontakte dette nummer undervejs på løbet, hvis situationen kræver det.</small>
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
                <Message severity="warn" :closable="false">registrering af indbetalinger sker manuelt.</Message>
            </div>
            <template #footer>
                <Button label="Send betalingslink" icon="pi pi-mobile" text @click="deleteMember" />
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
