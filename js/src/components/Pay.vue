<template>
<div class="modal fade" id="payModal" data-backdrop="static" data-keyboard="false" tabindex="-1" aria-labelledby="staticBackdropLabel" aria-hidden="true">
  <div class="modal-dialog">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title" id="staticBackdropLabel"><i class="fas fa-money-bill-alt"></i> Betaling</h5>
        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <div v-if="onHold" class="modal-body">
          <h1>Øv!</h1>
          <p>I er på venteliste, hvis der kommer et afbud kontakter vi jer.</p>
      </div>
      <div v-if="paymentDue" class="modal-body">
          <p>Jeres plads i køen er nu sikret, når I har betalt er I tilmeldt. Betaling til Nathejk sker med MobilePay.</p>
          <div class="form-group row">
            <label for="mobilepayPhone" class="col-sm-4 col-form-label">Telefonnummer</label>
            <div class="col-sm-8">
              <input type="phone" class="form-control" id="mobilepayPhone" v-model="team.phone">
            </div>
          </div>
          <small class="text-muted">Der kan gå op til 1 døgn før jeres betaling er registreret.</small>
      </div>
      <div v-if="paymentDue" class="modal-footer">
        <button type="button" class="btn btn-secondary" data-dismiss="modal" @click="send">Send betalingslink</button>
      </div>
    </div>
  </div>
</div>
</template>

<style lang="scss">
</style>

<script>
import axios from 'axios'

export default {
    props: {
        teamId: String,
        phone: String,
    },
    data: () => ({
        team: Object,
    }),
    computed: {
        onHold() {
            return this.team.status == 'HOLD'
        },
        paymentDue() {
            return (this.team.status == 'PAY' || this.team.status == 'PAID') && (this.team.unpaidMemberCount > 0)
        },
        allFine() {
            return this.team.status == 'PAID' && this.team.UnpaidMemberCount == 0
        },
    },
    methods: {
        async load() {
            try {
                const rsp = await axios.get('/api/checkout/' + this.teamId)
                if (rsp.status == 200) {
                    this.team = rsp.data
                    this.team.phone = this.phone
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }
        },
        async send() {
            const rsp = await axios.put('/api/mobilepay', Object.assign({}, this.team), { withCredentials: true })
            this.load()
            return rsp.data.teamId
        },
    },
    async mounted() {
        $('#payModal').on('shown.bs.modal', function (event) {
            this.load()
            // do something...
        }.bind(this))

        this.$nextTick(function () {
        })
    },
}
</script>
