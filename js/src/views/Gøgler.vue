<template>
    <div class="container pt-3">
        <Headline title="Gøglertilmelding" subtitle="dine kontaktoplysninger" />
        <div class="row">
            <div class="col-8">
                <div v-if="signup" class="p-3 bg-light border rounded">
                    <p class="lead">Indtast først dine oplysninger.</p>
                    <form @submit="submit">

                      <div class="form-group row">
                        <label for="staticName" class="col-3 col-form-label text-right">Fulde navn</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                                <div class="input-group-text"><i class="fas fa-fw fa-user"></i></div>
                            </div>
                            <input type="text" v-model="contact.name" class="form-control" id="staticName" placeholder="navn">
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticPhone" class="col-3 col-form-label text-right">Dit telefonnummer</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-mobile"></i></div>
                            </div>
                            <input type="text" v-model="contact.phone" class="form-control" id="staticPhone" placeholder="mobiltelefonnummer">
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticEmail" class="col-3 col-form-label text-right">Din e-mailadresse</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-at"></i></div>
                            </div>
                            <input type="email" v-model="contact.email" class="form-control" id="staticEmail" placeholder="e-mailadresse">
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticAddress" class="col-3 col-form-label text-right">Din adresse</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-home"></i></div>
                            </div>
                            <input type="text" v-model="contact.address" class="form-control" id="staticAddress" placeholder="adresse">
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticPostalCode" class="col-3 col-form-label text-right">Dit postnummer</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-home"></i></div>
                            </div>
                            <input type="text" v-model="contact.postalCode" class="form-control" id="staticPostalCode" placeholder="postnummer">
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticTeam" class="col-3 col-form-label text-right">Gøglerhold</label>
                        <div class="input-group col">
                            <div class="input-group-prepend">
                              <div class="input-group-text"><i class="fas fa-fw fa-users"></i></div>
                            </div>
                            <select v-model="contact.group" class="form-control" id="staticTeam" placeholder="gøglerhold">
                                <option value="martin og dennis">Martin og Dennis</option>
                                <option value="tommy og kristian">Tommy og Kristian</option>
                                <option value="niels erik">Niels Erik</option>
                                <option value="jonathan">Jonathan</option>
                            </select>
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticPostalCode" class="col-3 col-form-label text-right">Deltager</label>
                        <div class="input-group col">
                            <div class="form-check form-check-inline">
                              <input v-model="contact.friday" class="form-check-input" type="checkbox" id="inlineCheckbox1" value="friday">
                              <label class="form-check-label" for="inlineCheckbox1">fredag</label>
                            </div>
                            <div class="form-check form-check-inline">
                              <input v-model="contact.saturday" class="form-check-input" type="checkbox" id="inlineCheckbox2" value="saturday">
                              <label class="form-check-label" for="inlineCheckbox2">lørdag</label>
                            </div>
                            <div class="form-check form-check-inline">
                              <input v-model="contact.sunday" class="form-check-input" type="checkbox" id="inlineCheckbox3" value="sunday">
                              <label class="form-check-label" for="inlineCheckbox3">søndag</label>
                            </div>
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticPostalCode" class="col-3 col-form-label text-right">Fototilladelse</label>
                        <div class="input-group col">
                            <div class="form-check form-check-inline">
                              <input v-model="contact.photo" class="form-check-input" type="checkbox" id="gdpr" value="friday">
                              <label class="form-check-label" for="gdpr">Ja</label>
                            </div>
                            <small class="form-text text-muted">Gøglerne er en vigtig del af Nathejks ansigt udadtil, vi vil derfor gerne kunne bruge billeder i forbindelse med markedsføring af Nathejk på sociale medier og lignende.</small>
                        </div>
                      </div>
                      <div class="form-group row">
                        <label for="staticPostalCode" class="col-3 col-form-label text-right">Bemærkninger</label>
                        <div class="input-group col">
                            <textarea v-model="contact.comment" class="form-control" id="exampleFormControlTextarea1" rows="3"></textarea>
                        </div>
                      </div>
                      <div class="form-group row">
                        <div class="col-3"></div>
                        <div class="col">
                            <button type="submit" class="btn btn-success">Tilmeld og betal &raquo;</button>
                        </div>
                      </div>
                    </form>
                </div>
                <div v-else class="p-3 bg-light border rounded">
                    <p class="lead">Betaling</p>
                    <p>Det koster 50,- kr at deltage som gøgler på Nathejk, Vi har sendt dig en SMS (+45 {{ contact.phone }}) med et mobilepay betalingslink - når der er betalt er du tilmeldt.</p>
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
import TextInput from '@/components/TextInput.vue'
import moment from 'moment'
import axios from 'axios'

export default {
    data: () => ({
      title: 'Nathejk 2020',
      countdownDate: moment('2019-04-26 22:32'),
      now: moment(),
      tick: null,
      phone: 'i1',
      signup: true,
      contact: {
          name: '',
          phone: '',
          email: '',
          address: '',
          postalCode: '',
          group:'',
          friday: false,
          saturday: false,
          sunday: false,
          photo: false,
          comment:'',
      }
    }),
    props: {
        team: String
    },
    components: { Card, Headline, TextInput },
    computed: {
        countdownActive: function() {
            //return this.countdownDate.isBefore(moment());
            return moment().isBefore(this.countdownDate);
        }
    },
    methods: {
        submit: async function(e) {
            e.preventDefault();
            try {
                const rsp = await axios.post('/api/gogler', this.contact)
            } catch(error) {
                throw new Error(error.response.data)
            }
            this.signup = !this.signup
            console.log(this.contact);
        },
    },
}
</script>
