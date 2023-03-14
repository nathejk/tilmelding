<template>
    <div class="container py-3">
        <div v-if="loading">
            <Headline title="Tilmelding" subtitle="henter klanoplysninger" />
            <div class="d-flex justify-content-center"><i class="fas fa-spinner fa-5x fa-spin"></i></div>
        </div>
        <div v-else>
        <Headline title="Tilmelding" subtitle="indtast klanoplysninger" />

        <div class="row">
            <div class="col-8">

        <div class="card mb-3">
          <div class="card-header"><i class="fas fa-fw fa-lg fa-award"></i> Klan</div>
          <div class="card-body bg-gradient-secondary">
              <div class="form-group mb-1">
                <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Klannavn</label>
                <input type="text" class="form-control form-control-sm py-0" v-model="team.name">
              </div>
              <div class="form-group mb-1">
                <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Gruppe og division</label>
                <input type="text" class="form-control form-control-sm py-0" v-model="team.groupName">
                <!-- small id="emailHelp" class="form-text text-muted">We'll never share your email with anyone else.</small -->
              </div>
              <div class="form-group mb-1">
                <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Korps</label>
                <select v-model="team.korps" class="form-control form-control-sm py-0">
                    <option v-for="option in korps" v-bind:value="option.value">{{ option.text }}</option>
                </select>
              </div>
          </div>
        </div>

            </div>
        </div>
        <div class="row">
            <div class="col-8">

        <div class="card">
            <div class="card-header"><i class="fas fa-fw fa-lg fa-users"></i> Deltagere: {{ memberCount }}</div>
          <div class="card-body">
            <CollapsibleMember v-for="member, i in team.spejdere" :key="'collapsible-member-'+member.id" :open="fieldsMissing(member)" :member="member" :deletable="deletable(member)" @deleted="deleteMember" />
            <button type="button" class="btn btn-outline-success m-2" @click="addMember" :disabled="!canAddMember"><i class="fas fa-fw fa-lg fa-user-plus"></i> Tilføj deltager</button>
          </div>
        </div>

            </div>
            <div class="col">

        <div class="card mb-3 alert-warning" v-if="false">
          <div class="card-header"><i class="fas fa-fw fa-lg fa-star"></i> Vigtigt</div>
          <div class="card-body">
            <p><strong>Spejderens telefonnummer</strong> Mobilnummer på Nathejk (kun hvis telefon medbringes)</p>
            <p><strong>Forældres telefonnummer</strong> – Nathejk skal kunne kontakte dette nummer undervejs på løbet, hvis situationen kræver det.</p>
          </div>
        </div>

            </div>
        </div>

        <div class="row">
            <div class="col">
                <div class="d-flex flex-row-reverse">
                    <button type="button" class="btn btn-success ml-3" v-if="unpaidMemberCount > 0" @click="pay" :disabled="!payable">Gem og betal ({{ unpaidMemberCount }} seniorer)</button>
                    <button type="button" class="btn btn-outline-success ml-3" @click="save" :disabled="!saveable">Gem ændringer</button>
                </div>
            </div>
        </div>
        <PayModal :teamId="teamId" :phone="team.contactPhone" />
    </div>
    </div>
</template>

<style lang="scss">
</style>

<script>
import Headline from '@/components/Headline.vue'
import PayModal from '@/components/Pay'
import CollapsibleMember from '@/components/CollapsibleSenior.vue'
import DatePicker from '@sum.cumo/vue-datepicker'
import '@sum.cumo/vue-datepicker/dist/Datepicker.css'
import axios from 'axios'

export default {
    components: { Headline, DatePicker, CollapsibleMember, PayModal },
    data: () => ({
        korps: [
          { text: 'Det Danske Spejderkorps', value: 'dds' },
          { text: 'KFUM-Spejderne', value: 'kfum' },
          { text: 'De grønne pigespejdere', value: 'kfuk' },
          { text: 'Danske Baptisters Spejderkorps', value: 'dbs' },
          { text: 'De Gule Spejdere', value: 'dgs' },
          { text: 'Dansk Spejderkorps Sydslesvig', value: 'dss' },
          { text: 'FDF / FPF', value: 'fdf' },
          { text: 'Andet', value: 'andet' },
        ],
        team: Object,
        phone: "",
    }),
    computed: {
        loading() {
                console.log("teamID:", this.team.teamId, !!this.team.teamId)
            return !this.team.teamId
        },
        teamId() {
            return this.$route.params.id
        },
        memberCount() {
            if (!this.team || !this.team.spejdere) return 0
            let c = 0
            for (const m of this.team.spejdere) {
                if (!m.deleted) {
                        console.log("counting member", m)
                    c++;
                }
            }
            return c
        },
        canAddMember() {
            return this.memberCount < 5
        },
        unpaidMemberCount() {
            return this.memberCount - this.team.paidMemberCount
        },
        saveable() {
            return this.team.name != "" && this.memberCount >= 1 && this.memberCount <= 5
        },
        payable() {
            return this.saveable && this.unpaidMemberCount > 0
        },
        
    },
    methods: {
        addMember() {
            this.team.spejdere.push({id: ''+this.team.spejdere.length, deleted:false, name: "Deltager "+(this.memberCount + 1)})
        },
        deleteMember(member, deleted) {
            for (const m of this.team.spejdere) {
                if (m.id == member.id) {
                    m.deleted = deleted;
                }
            }
        },
        deletable(member) {
            return member.deleted || this.memberCount > 1
        },
        fieldsMissing(member) {
            return false
        },
        async load() {
            try {
                const rsp = await axios.get('/api/klan/' + this.teamId)
                if (rsp.status == 200) {
                    for (const m of rsp.data.spejdere) {
                        m.deleted = false
                    }
                    this.team = rsp.data
                    this.phone = rsp.data.contactPhone
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }
        },
        async save() {
            const rsp = await axios.put('/api/klan', Object.assign({}, this.team), { withCredentials: true })
            this.load()
            return rsp.data.teamId
        },
        async pay() {
            this.save()
            $('#payModal').modal('show')
        },
    },
    async mounted() {
        let ids =this.teamId.split(":")
        if (ids.length == 2) {

            try {
                const rsp = await axios.get('/api/bridge/'+ids[0])
                if (rsp.status == 200) {
                    this.$router.replace({ name: "klan-view", params: { id: rsp.data.teamId } });
                    return
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }
        }
        this.load()

        this.$nextTick(function () {
        })
    },
}

const safeAsync = func => {
  return async (args, defaultValue) => {
    try {
      return await func(args)
    } catch (error) {
      return defaultValue ? defaultValue : null
    }
  }
}
</script>
