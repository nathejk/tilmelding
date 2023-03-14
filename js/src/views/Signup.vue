<template>
    <div class="container pt-3">
        <Headline v-if="signup" title="Tilmelding" subtitle="dine kontaktoplysninger" />
        <Headline v-else title="Tilmelding" subtitle="bekræft dine kontaktoplysninger" />
        <div class="row">
            <div class="col-8">
                <div v-if="signup" class="p-3 bg-light border rounded">
                    <p class="lead">Indtast først dine (tilmelderens) kontaktoplysninger.</p>
                    <form @submit="submit" class="needs-validation" novalidate>
                      <div class="form-group row">
                        <label for="staticName" class="col-3 col-form-label text-right">Dit navn</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                                <div class="input-group-text"><i class="fas fa-fw fa-user"></i></div>
                            </div>
                            <input type="text" v-model="form.name" class="form-control" id="staticName" placeholder="navn" required>
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticPhone" class="col-3 col-form-label text-right">Dit telefonnummer</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-mobile"></i></div>
                            </div>
                            <input type="text" v-model="form.phone" class="form-control" id="staticPhone" placeholder="mobiltelefonnummer">
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticEmail" class="col-3 col-form-label text-right">Din e-mailadresse</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-at"></i></div>
                            </div>
                            <input type="email" v-model="form.email" class="form-control" id="staticEmail" placeholder="e-mailadresse" required>
                        </div>
                      </div>
                      <div class="form-group row">
                        <div class="col-3"></div>
                        <div class="col">
                            <button type="submit" class="btn btn-success">Videre &raquo;</button>
                        </div>
                      </div>
                    </form>
                </div>
                <div v-else class="p-3 bg-light border rounded">
                    <p class="lead">E-mail eller SMS</p>
                    <p>Vi har sendt dig en e-mail med et link og en SMS med en aktiveringskode, benyt en af delene til at bekræfte dine kontaktoplysninger og komme videre med tilmeldingen.</p>
                    <form @submit="confirm">
                      <div class="form-group row">
                        <label for="staticPhone" class="col-3 col-form-label text-right">Mobiltelefon</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-mobile"></i></div>
                            </div>
                            <input type="text" v-model="form.phone" readonly class="form-control" id="staticPhone" placeholder="mobiltelefonnummer">
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticEmail" class="col-3 col-form-label text-right">Aktivering</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-hashtag"></i></div>
                            </div>
                            <input type="text" v-model="pincode" class="form-control" id="staticEmail" placeholder="pinkode" required>
                        </div>
                      </div>
                      <div class="form-group row">
                        <div class="col-3"></div>
                        <div class="col">
                            <button type="submit" class="btn btn-success">Bekræft &raquo;</button>
                        </div>
                      </div>
                    </form>
                </div>
            </div>
            <div class="col">
                <Card v-if="team == 'patrulje'" layout="compact" color="blue" title="PATRULJER" subtitle="12-16 år">
                    <p><strong>REGLER FOR PATRULJER</strong></p>
                    <ul>
                        <li>Der skal mindst være min 3 deltagere på holdet - og max 7</li>
                        <li>Ingen er under 12 år.</li>
                        <li>Ingen er fyldt 17 år.</li>
                        <li>Gennemsnitsalderen skal være mindst 13 år.</li>
                    </ul>
                </Card>
                <Card v-if="team == 'klan'" layout="compact" color="red" title="SENIORER" subtitle="+16 år">
                    <p><strong>REGLER FOR SENIORER</strong></p>
                    <ul>
                        <li>Der må max være 5 senioer i en klan.</li>
                        <li>Alle skal være fyldt 16 år.</li>
                    </ul>
                </Card>
            </div>
        </div>


    </div>
</template>

<style lang="scss">
ul { padding-inline-start: 20px; }
</style>

<script>
import Card from '@/components/Card.vue'
import Headline from '@/components/Headline.vue'
import moment from 'moment'
import axios from 'axios'

export default {
    data: () => ({
      countdownDate: moment('2021-05-04 20:21'),
      now: moment(),
      tick: null,
      form: {
        type: '',
        name: '',
        phone: '',
        email: ''
      },
      //confirmation: {

      teamId: '',
      pincode: '',
      signup: true,
    }),
    props: {
        team: String
    },
    components: { Card, Headline },
    computed: {
        countdownActive: function() {
            //return this.countdownDate.isBefore(moment());
            return moment().isBefore(this.countdownDate);
        }
    },
    methods: {
        submit: async function(e) {
            e.preventDefault();
            e.stopPropagation();
            const form = e.target || e.currentTarget

            if (form && form.checkValidity() === false) {
                form.classList.add('was-validated');
                return
            }
            this.form.type = this.team

            try {
                const rsp = await axios.post(window.envConfig.API_BASEURL + '/api/signup', this.form)
                this.teamId = rsp.data.teamId
            } catch(error) {
                throw new Error(error.response.data)
            }
            this.signup = !this.signup;
        },
        confirm: async function(e) {
            e.preventDefault();
            e.stopPropagation();
            try {
                this.form.pincode = this.pincode
                const rsp = await axios.post(window.envConfig.API_BASEURL + '/api/confirm', {phone:this.form.phone, pincode:this.pincode, teamId:this.teamId})
                if (rsp.status == 200 && rsp.data.OK) {
                    this.$router.replace({ name: this.team+"-view", params: { id: this.teamId } });
                }
            } catch(error) {
                this.pincode = ''
                throw new Error(error.response.data)
            }
        },
    },
}
</script>
