<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Navigation from '@/components/Navigation.vue'
import Shop from '@/components/Shop.vue'
import FloatLabel from 'primevue/floatlabel'
import Calendar from 'primevue/calendar'
import InputGroup from 'primevue/inputgroup'
import InputGroupAddon from 'primevue/inputgroupaddon'
import {
  aggregateOrderLines,
  orderTotalDkk,
  orderDueDkk,
  orderShortLines,
  orderDateShort,
  totalPaidDkk
} from '@/helpers/order'

const props = defineProps({
  teamId: { type: String, required: false }
})

const router = useRouter()

const config = ref({})
const team = ref({})
const contact = ref({})
const members = ref([])
const order = ref(null)
const paidOrders = ref([])

// Expenses, payments, totals are now derived from the server-side order
// envelope. The backend computes prices and aggregates per-member lines;
// the frontend just groups by SKU for display and converts øre → DKK.
const expenses = computed(() => aggregateOrderLines(order.value))
const expensesTotal = computed(() => orderTotalDkk(order.value))
// paymentsTotal is the cumulative paid amount across the open order and
// every paid order in the history — the answer to "how much have I paid
// in total".
const paymentsTotal = computed(() => totalPaidDkk(order.value, paidOrders.value))
const payableAmount = computed(() => orderDueDkk(order.value))
// orderStatus is "OPEN" or "PAID" — see order.MarshalJSON on the
// backend. Used by the template to render a status badge and to disable
// the form once the order is locked.
const orderStatus = computed(() => (order.value && order.value.status) || 'OPEN')
// showOpenOrder gates the open-order block (status badge, expense
// rows, "I alt", "At betale", refund text). When the open order has
// nothing due there's no actionable content there — only the paid
// history (if any) is worth showing.
const showOpenOrder = computed(() => payableAmount.value > 0)
// showPaymentsSection gates the entire Betalinger fieldset: render it
// when there's either a current bill (open order with due > 0) or any
// historical paid orders to display.
const showPaymentsSection = computed(() => showOpenOrder.value || paidOrders.value.length > 0)

onMounted(async () => {
  try {
    const response = await fetch('/api/patrulje/' + props.teamId)
    if (!response.ok) {
      throw new Error('HTTP status ' + response.status)
    }
    const data = await response.json()
    config.value = data.config
    team.value = data.team
    contact.value = data.contact
    members.value = data.members ? [...data.members] : []
    order.value = data.order || null
    paidOrders.value = data.paidOrders ? [...data.paidOrders] : []

    console.log('found', data)
  } catch (error) {
    console.log('mounted load failed', error)
  }
})

const member = ref({})

const isLoading = ref(false)
const memberDialog = ref(false)
const deleteMemberDialog = ref(false)
const paymentDialog = ref(false)
const teamSubmitted = ref(false)
const memberSubmitted = ref(false)

const confirmDeleteMember = (prod) => {
  member.value = prod
  deleteMemberDialog.value = true
}
const editMember = (prod) => {
  member.value = { ...prod }
  memberDialog.value = true
}
const openNew = () => {
  member.value = {}
  memberSubmitted.value = false
  memberDialog.value = true
}
const hideDialog = () => {
  memberDialog.value = false
  memberSubmitted.value = false
}
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

const canSave = computed(() => activeMembers.value.length >= 3 && activeMembers.value.length <= 7)

// putState is the shared HTTP PUT helper used by both syncOrder (silent
// background save after every member edit) and save (final "Gem" button
// which redirects to MobilePay if there's anything to pay). It mutates
// team/contact/order refs from the server response so the UI stays in
// sync with the projection.
const putState = async () => {
  const headers = { 'Content-Type': 'application/json' }
  const body = JSON.stringify({
    team: team.value,
    contact: contact.value,
    members: members.value
  })
  const response = await fetch('/api/patrulje/' + props.teamId, {
    method: 'PUT',
    body: body,
    headers: headers
  })
  if (!response.ok) {
    throw new Error('HTTP status ' + response.status)
  }
  const data = await response.json()
  if (data.team) team.value = data.team
  if (data.order) order.value = data.order
  if (data.paidOrders) paidOrders.value = [...data.paidOrders]
  return data
}

// syncOrder pushes the current form state to the server so the order is
// recomputed (lines, totalAmount, dueAmount). Called whenever a member
// is added, removed or edited so the displayed totals always match the
// form, without requiring the user to click "Gem". Errors are swallowed
// silently — the next sync (or the final save) will retry.
const syncOrder = async () => {
  try {
    await putState()
  } catch (error) {
    console.log('syncOrder failed', error)
  }
}

const save = async () => {
  try {
    const data = await putState()
    if (data.paymentLink && data.paymentLink != '') {
      location.href = data.paymentLink
    } else {
      router.push({ name: 'thankyou' })
    }
  } catch (error) {
    console.log('team signup failed', error)
  }
}

const mobilepay = ref('')
const pay = async () => {
  const headers = {
    'Content-Type': 'application/json'
  }
  try {
    const body = JSON.stringify({
      phone: mobilepay.value,
      amount: payableAmount.value
    })
    const response = await fetch('/api/pay/' + props.teamId, {
      method: 'PUT',
      body: body,
      headers: headers
    })
    if (!response.ok) {
      throw new Error('HTTP status ' + response.status)
    }
    const data = await response.json()
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
    console.log('team signup failed', error)
  }
}

const saveMember = () => {
  memberSubmitted.value = true

  if (member.value.name.trim() == '') {
    return
  }
  if (member.value.id) {
    //member.value.inventoryStatus = product.value.inventoryStatus.value ? product.value.inventoryStatus.value : product.value.inventoryStatus;
    members.value[findIndexById(member.value.id)] = member.value
    //toast.add({severity:'success', summary: 'Successful', detail: 'Product Updated', life: 3000});
  } else {
    member.value.id = createId()
    members.value.push(member.value)
    //    toast.add({severity:'success', summary: 'Successful', detail: 'Product Created', life: 3000});
  }
  memberDialog.value = false
  member.value = { name: '' }
  // Recalculate the order on the server now that the member set has
  // changed (added, edited, or t-shirt size flipped). Fire-and-forget
  // — the user keeps editing while the order refreshes in the
  // background.
  syncOrder()
}
const activeMembers = computed(() => members.value.filter((i) => !i.deleted))
const deleteMember = () => {
  //members.value = members.value.filter(val => val.id !== member.value.id);
  members.value[findIndexById(member.value.id)].deleted = true
  deleteMemberDialog.value = false
  member.value = {}
  // Same rationale as saveMember — sync the order so totals reflect
  // the removal immediately.
  syncOrder()
  //toast.add({severity:'success', summary: 'Successful', detail: 'Product Deleted', life: 3000});
}
const findIndexById = (id) => {
  let index = -1
  for (let i = 0; i < members.value.length; i++) {
    if (members.value[i].id === id) {
      index = i
      break
    }
  }

  return index
}
const createId = () => {
  let id = ''
  var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  for (var i = 0; i < 5; i++) {
    id += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  return id
}
const tshirtSizeLabel = (slug) => {
  if (slug == '') return ''
  for (const tshirt of config.value.tshirtSizes) {
    if (tshirt.slug == slug) return tshirt.label
  }
  return ''
}
</script>

<template>
  <Navigation class="dark" />

  <div class="container mx-auto">
    <div class="grid grid-cols-2 gap-4">
      <Fieldset class="mt-3" legend="Patruljeoplysninger">
        <p class="m-0">
          Spejderpatruljen skal bestå af mellem 3 og 7 spejdere, Når I tilmelder patruljen skal i
          derfor mindst tilmelde 3 spejdere, I kan når som helst eftertilmelde ekstra spejdere
          sålænge patruljen ikke overstiger 7 spejdere.
        </p>
        <div class="flex flex-col">
          <FloatLabel class="mt-7">
            <InputText
              id="team-name"
              v-model.trim="team.name"
              size="small"
              class="w-full"
              required="true"
              autofocus
              :invalid="teamSubmitted && !team.name"
            />
            <label for="team-name">Patruljenavn</label>
          </FloatLabel>
          <small class="p-error mb-2" v-if="teamSubmitted && !team.name"
            >Patruljenavn skal indtastes.</small
          >
          <!--small id="member-help">Enter your username to reset your password.</small-->
        </div>
        <div class="flex flex-col">
          <FloatLabel class="mt-7">
            <InputText
              id="team-group"
              v-model.trim="team.group"
              size="small"
              class="w-full"
              required="true"
              :class="{ 'p-invalid': teamSubmitted && !team.group }"
            />
            <label for="team-group">Gruppe og division</label>
          </FloatLabel>
          <small class="p-error mb-2" v-if="teamSubmitted && !member.name"
            >Gruppe og division skal indtastes.</small
          >
          <!--small id="member-help">Enter your username to reset your password.</small-->
        </div>
        <div class="flex flex-col">
          <FloatLabel class="mt-7">
            <Dropdown
              v-model="team.korps"
              inputId="team-korps"
              :options="config.korps"
              optionValue="slug"
              optionLabel="label"
              class="filled w-full md:w-14rem"
            />
            <label for="team-korps">Spejderkorps</label>
          </FloatLabel>
        </div>
        <div class="flex flex-col">
          <FloatLabel class="mt-7">
            <InputText id="team-liga" v-model.trim="team.liga" size="small" class="w-full" />
            <label for="team-liga">Adventurespejdliga nummer</label>
          </FloatLabel>
          <small id="team-liga-help"
            >Læs mere om LigaID og tilmeld jer Adventurespejdligaen her:
            <a href="">adventurespejd.dk</a>.</small
          >
        </div>
      </Fieldset>

      <Fieldset class="mt-3" legend="Kontaktperson">
        <p class="m-0">
          Kontaktpersonen er meget vigtig og skal være en person, som kender patruljen godt (fx
          tropslederen). Nathejks team skal kunne få fat i kontaktpersonen undervejs på løbet, hvis
          situationen kræver det.
        </p>
        <div class="flex flex-col">
          <FloatLabel class="mt-7">
            <InputText
              id="contact-name"
              v-model.trim="contact.name"
              size="small"
              class="w-full"
              required="true"
              :class="{ 'p-invalid': teamSubmitted && !contact.name }"
            />
            <label for="team-name">Navn</label>
          </FloatLabel>
          <small class="p-error mb-2" v-if="teamSubmitted && !contact.name"
            >Kontaktperson skal indtastes.</small
          >
          <!--small id="member-help">Enter your username to reset your password.</small-->
        </div>
        <div v-if="false" class="flex flex-col">
          <FloatLabel class="mt-7">
            <InputText
              id="contact-address"
              v-model.trim="contact.address"
              size="small"
              class="w-full"
            />
            <label for="contact-address">Adresse</label>
          </FloatLabel>
        </div>
        <div v-if="false" class="flex flex-col">
          <FloatLabel class="mt-7">
            <InputText
              id="contact-postal"
              v-model.trim="contact.postal"
              size="small"
              class="w-full"
            />
            <label for="contact-postal">Postnummer og by</label>
          </FloatLabel>
        </div>
        <div class="flex flex-col">
          <FloatLabel class="mt-7">
            <InputText
              id="contact-phone"
              v-model.trim="contact.phone"
              size="small"
              class="w-full"
            />
            <label for="contact-phone">Telefonnummer</label>
          </FloatLabel>
        </div>
        <div class="flex flex-col">
          <FloatLabel class="mt-7">
            <InputText
              id="contact-email"
              v-model.trim="contact.email"
              size="small"
              class="w-full"
            />
            <label for="contact-email">E-mail</label>
          </FloatLabel>
        </div>
        <div class="flex flex-col">
          <FloatLabel class="mt-7">
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
              <p v-if="row.data.email" class="m-0 font-thin">
                <i class="pi pi-envelope"></i> {{ row.data.email }}
              </p>
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
              <p v-if="row.data.phone" class="m-0 font-thin">
                <i class="pi pi-mobile"></i> {{ row.data.phone }}
              </p>
              <p v-if="row.data.phoneContact" class="m-0 font-thin">
                <i class="pi pi-phone"></i> {{ row.data.phoneContact }}
              </p>
            </template>
          </Column>
          <Column field="birthday" header="Fødselsdag"></Column>
          <Column field="tshirt" header="T-Shirt">
            <template #body="row" style="font-size: 0.8rem">
              {{ tshirtSizeLabel(row.data.tshirtSize) }}
            </template>
          </Column>
          <Column style="min-width: 3rem">
            <template #body="row">
              <div class="text-end">
                <Button
                  icon="pi pi-pencil"
                  outlined
                  rounded
                  class="mr-2"
                  @click="editMember(row.data)"
                />
                <Button
                  v-if="members.length > config.minMemberCount"
                  icon="pi pi-trash"
                  outlined
                  rounded
                  severity="danger"
                  @click="confirmDeleteMember(row.data)"
                />
              </div>
            </template>
          </Column>
          <template #footer>
            <div class="text-end">
              <Button
                icon="pi pi-plus"
                outlined
                rounded
                @click="openNew"
                :disabled="activeMembers.length >= config.maxMemberCount"
              />
            </div>
          </template>
        </DataTable>
      </div>
    </Fieldset>
    <Fieldset class="mt-3" legend="Betalinger" v-if="showPaymentsSection">
      <div class="card">
        <div v-if="showOpenOrder" class="text-right text-sm pb-2">
          Status:
          <span
            class="font-bold ml-1"
            :class="orderStatus === 'PAID' ? 'text-green-600' : 'text-orange-500'"
            >{{ orderStatus }}</span
          >
        </div>
        <div class="grid grid-cols-6 gap-4">
          <template v-if="showOpenOrder">
            <div class="col-start-4 text-center">Antal</div>
            <div class="text-center">Pris</div>
            <div class="text-center">Total</div>
            <template v-for="expense in expenses">
              <div class="col-start-1 col-span-3">{{ expense.text }}</div>
              <div class="text-right">{{ expense.count }}</div>
              <div class="text-right">{{ expense.unitPrice }},-</div>
              <div class="text-right">{{ expense.amount }},-</div>
            </template>
            <div class="col-start-1 col-span-5 font-bold">I alt</div>
            <div class="font-bold text-right">{{ expensesTotal }},-</div>
            <Divider class="col-start-1 col-end-7" />
          </template>
          <template v-if="paidOrders.length">
            <div class="col-start-1 col-span-6 text-sm font-bold">Tidligere betalte ordrer</div>
            <template v-for="po in paidOrders" :key="po.orderId">
              <div class="col-start-1 col-span-2 text-sm">{{ orderDateShort(po) }}</div>
              <div class="col-span-3 text-sm">{{ orderShortLines(po) }}</div>
              <div class="col-end-7 text-right text-sm">
                {{ Math.round(po.paidAmount / 100) }},-
              </div>
            </template>
            <Divider class="col-start-1 col-end-7" />
          </template>
          <div class="col-start-1 col-span-3 font-bold">Indbetalt i alt</div>
          <div class="col-end-7 font-bold text-right">{{ paymentsTotal }},-</div>
          <template v-if="showOpenOrder">
            <Divider class="col-start-1 col-end-7" />
            <div class="col-start-1 col-span-5 font-bold">At betale</div>
            <div class="font-bold text-right">{{ payableAmount }},-</div>
          </template>
          <Divider class="col-end-7" />
        </div>
        <p v-if="showOpenOrder">
          Deltagerbetalingen bliver ikke refunderet ved afbud uanset grund - vi kan have brugt
          pengene ud fra en forventning om, at du kommer. Det er dog helt frem til ganske kort før
          løbsstart muligt at skifte ud blandt deltagerne. Betalingen bliver naturligvis refunderet,
          hvis holdet ikke deltager, fordi Nathejks team har besluttet det.
        </p>
      </div>
    </Fieldset>

    <div class="card flex justify-end">
      <Button
        class="my-5"
        :disabled="!canSave"
        :label="payableAmount ? 'Gem ændringer og betal' : 'Gem ændringer'"
        @click="save"
      />
    </div>
  </div>

  <Dialog v-model:visible="memberDialog" :style="{ width: '450px' }" header="Spejder" :modal="true">
    <div class="flex flex-col">
      <FloatLabel class="mt-4">
        <InputText
          id="member-fullname"
          v-model.trim="member.name"
          size="small"
          class="w-full"
          required="true"
          autofocus
          :invalid="memberSubmitted && !member.name"
        />
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
        <InputText
          type="email"
          id="member-email"
          v-model="member.email"
          size="small"
          class="w-full"
        />
        <label for="member-email">E-mail adresse</label>
      </FloatLabel>
    </div>
    <div class="flex flex-col">
      <FloatLabel class="mt-7">
        <InputText id="member-phone" v-model="member.phone" size="small" class="w-full" />
        <label for="member-phone">Telefonnummer</label>
      </FloatLabel>
      <small id="member-phone-help" class="text-slate-400"
        >Mobilnummer på Nathejk (kun hvis telefon medbringes).</small
      >
    </div>
    <div class="flex flex-col">
      <FloatLabel class="mt-7">
        <InputText
          id="member-phoneContact"
          v-model="member.phoneContact"
          size="small"
          class="w-full"
        />
        <label for="member-phoneContact">Telefonnummer på pårørende</label>
      </FloatLabel>
      <small id="member-phoneContact-help" class="text-slate-400"
        >Nathejk skal kunne kontakte dette nummer undervejs på løbet, hvis situationen kræver
        det.</small
      >
    </div>
    <div class="flex flex-col">
      <FloatLabel class="mt-7">
        <Calendar
          inputId="member-birthday"
          v-model="member.birthday"
          size="small"
          class="w-full filled"
          dateFormat="yy-mm-dd"
          showIcon
          iconDisplay="input"
        />
        <label for="member-birthday">Fødselsdato</label>
      </FloatLabel>
    </div>
    <div class="flex flex-col">
      <FloatLabel class="mt-7">
        <Dropdown
          v-model="member.tshirtSize"
          inputId="member-tshirt"
          :options="config.tshirtSizes"
          optionValue="slug"
          optionLabel="label"
          class="w-full filled md:w-14rem"
        />
        <label for="member-tshirt">Vælg t-shirt</label>
      </FloatLabel>
    </div>

    <template #footer>
      <Button label="Afbryd" icon="pi pi-times" text @click="hideDialog" />
      <Button label="Gem" icon="pi pi-check" text @click="saveMember" />
    </template>
  </Dialog>

  <Dialog
    v-model:visible="deleteMemberDialog"
    :style="{ width: '450px' }"
    header="Bekræft"
    :modal="true"
  >
    <div>
      <i class="pi pi-exclamation-triangle mr-3" style="font-size: 2rem" />
      <span v-if="member"
        >Er det rigtigt at <b>{{ member.name }}</b> ikke skal deltage på Nathejk?</span
      >
    </div>
    <template #footer>
      <Button label="Nej" icon="pi pi-times" text @click="deleteMemberDialog = false" />
      <Button label="Ja" icon="pi pi-check" text @click="deleteMember" />
    </template>
  </Dialog>

  <Dialog
    v-model:visible="paymentDialog"
    :style="{ width: '500px' }"
    header="Betaling"
    :modal="true"
  >
    <div class="confirmation-content">
      <p class="m-0 mb-5">
        Vi sender en SMS med et MobilePay betalingslink på DKK {{ payableAmount }},- til det
        indtastede telefonnummer.
      </p>
      <InputGroup size="small">
        <InputGroupAddon>+45</InputGroupAddon>
        <InputText size="small" placeholder="Telefonnummer" v-model="mobilepay" />
      </InputGroup>
      <Message severity="warn" :closable="false"
        >registrering af indbetalinger sker manuelt.</Message
      >
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
  position: fixed !important;
  top: 0;
  left: 0;
  width: 100% !important;
  height: 100% !important;
  z-index: 100; /* this seems to work for me but may need to be higher*/
}
</style>
