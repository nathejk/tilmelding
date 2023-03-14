<template>
<div class="container py-3">
  <div v-if="loading">
    <div class="d-flex justify-content-center"><i class="fas fa-spinner fa-5x fa-spin"></i></div>
  </div>
  <div v-else>

    <SettingsNavigation active="signup" />

    <div class="row">
      <div class="col">

        <div class="custom-control custom-switch pt-3 pb-1">
          <input type="checkbox" class="custom-control-input" id="openSpejder" v-model="settings.openSpejder">
          <label class="custom-control-label" for="openSpejder">Åben for spejdertilmeldinger</label>
        </div>
        <fieldset :disabled="!settings.openSpejder" class="border p-3 bg-light">
          <div class="form-group row">
            <label for="maxSeniorCount" class="col-sm-3 col-form-label">Maks. antal patruljer</label>
            <div class="col-sm-3">
              <input type="number" class="form-control" id="maxSeniorCount" v-model.number="settings.maxPatruljeCount">
            </div>
          </div>
          <div class="form-group row">
            <label for="maxSeniorCount" class="col-sm-3 col-form-label">Pris per spejder</label>
            <div class="col-sm-3">
              <input type="number" class="form-control" id="maxSeniorCount" v-model.number="settings.spejderPrice">
            </div>
          </div>
        </fieldset>

        <div class="custom-control custom-switch pt-3 pb-1">
          <input type="checkbox" class="custom-control-input" id="openSenior" v-model="settings.openSenior">
          <label class="custom-control-label" for="openSenior">Åben for seniortilmeldinger</label>
        </div>
        <fieldset :disabled="!settings.openSenior" class="border p-3 bg-light">
          <div class="form-group row">
            <label for="inputEmail3" class="col-sm-3 col-form-label">Starttidspunkt</label>
            <div class="col-sm-3">
              <VueCtkDateTimePicker format="YYYY-MM-DD HH:mm" formatted="D. MMM. YYYY [kl.] HH:mm" output-format="YYYY-MM-DDTHH:mm:ssZ" :disabled="!settings.openSenior" locale="da" :firstDayOfWeek="1" :no-label="true" v-model="settings.signupStart" />
            </div>
          </div>
          <div class="form-group row">
            <label for="maxSeniorCount" class="col-sm-3 col-form-label">Maks. antal seniorer</label>
            <div class="col-sm-3">
              <input type="number" class="form-control" id="maxSeniorCount" v-model.number="settings.maxSeniorCount">
            </div>
          </div>
          <div class="form-group row">
            <label for="maxSeniorCount" class="col-sm-3 col-form-label">Pris per senior</label>
            <div class="col-sm-3">
              <input type="number" class="form-control" id="maxSeniorCount" v-model.number="settings.seniorPrice">
            </div>
          </div>
        </fieldset>

        <div class="form-group row">
          <div class="col">
            <button type="button" @click="save" class="btn btn-primary">Gem indstillinger</button>
          </div>
        </div>

      </div>
    </div>

  </div>

</div>
</template>

<style lang="scss">
</style>

<script>
import SettingsNavigation from '@/components/SettingsNavigation.vue'
import VueCtkDateTimePicker from 'vue-ctk-date-time-picker';
import 'vue-ctk-date-time-picker/dist/vue-ctk-date-time-picker.css';
import axios from 'axios'

export default {
    components: { SettingsNavigation, VueCtkDateTimePicker },
    props: {
        teamId: String,
    },
    data: () => ({
        settings: Object,
    }),
    computed: {
        loading() {
            return !this.settings.loaded
        },
    },
    methods: {
        async load() {
            try {
                const rsp = await axios.get('/api/settings')
                if (rsp.status == 200) {
                    this.settings = rsp.data
                    this.settings.loaded = true
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }
        },
        async save() {
            delete(this.settings.loaded)
            const rsp = await axios.put('/api/settings', Object.assign({}, this.settings), { withCredentials: true })
            this.load()
            return rsp.data.teamId
        },
    },
    async mounted() {
        this.load()

        this.$nextTick(function () {
        })
    },
}
</script>
