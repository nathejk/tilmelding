<template>
    <div class="container py-3">
        <Headline title="Tilmelding" subtitle="indtast deltageroplysninger" />
        <form enctype="multipart/form-data" action="" method="post" @submit="submit">

        <div class="row">
            <div class="col-8">
                <div class="p-3 bg-light border rounded">
                    <fieldset id="team">
                        <legend><i class="fas fa-map-signs"></i> Patruljeoplysninger</legend>
                        <TextInput label="Patrulje" v-model="team.name" />
                        <TextInput label="Gruppe og division" v-model="team.groupName" />
                        <div class="control-group form-group row">
                          <label class="col-sm-3 col-form-label col-form-label-sm">Korps</label>
                          <div class="col-sm-9">
                              <select v-model="team.korps" class="form-control form-control-sm">
                                <option v-for="option in korps" v-bind:value="option.value">{{ option.text }}</option>
                              </select>
                          </div>
                        </div>
                        <TextInput label="Evt. Liga-ID" v-model="team.advspejdNumber">
                            Læs mere om LigaID og tilmeld jer Adventurespejdligaen her: <a href="http://adventurespejd.dk/faq/#tilmeldingsguide">adventurespejd.dk</a>.
                        </TextInput>
                    </fieldset>
                </div>
            </div>
        </div>

        <div class="row">
            <div class="col-8">
                <div class="p-3 my-3 bg-light border rounded">
                    <fieldset id="contact">
                        <legend><i class="fas fa-user-tie"></i> Kontaktperson under Nathejk</legend>
                        <TextInput label="Navn" v-model="team.contactName" />
                        <TextInput label="Adresse" v-model="team.contactAddress" />
                        <TextInput label="Postnummer / by" v-model="team.contactPostalCode" />
                        <TextInput label="E-mail" v-model="team.contactMail" placeholder="din@e-mailadresse.dk" type="email" />
                        <TextInput label="Telefonnummer" v-model="team.contactPhone" />
                        <TextInput label="Rolle ift. patrulje" v-model="team.contactRole" />
                    </fieldset>
                </div>
            </div>
            <div class="col">
                <div class="p-3 my-3 alert alert-warning  border rounded">
                        <h4><i class="fas fa-exclamation-triangle"></i> Vigtigt</h4>
                        <p>Kontaktpersonen er meget vigtig og skal være en person, som kender patruljen godt (fx tropslederen). Nathejks team skal kunne få fat i kontaktpersonen undervejs på løbet, hvis situationen kræver det.</p>
                </div>
            </div>
        </div>
        <div class="row">
            <div class="col">
                <div class="p-3 bg-light border rounded">
                    <fieldset>
                        <h4 style="display:inline-block"><i class="fas fa-users"></i> Deltagere</h4>

                        <div class="btn-group mx-4" role="group" aria-label="First group">
                            <button type="button" class="btn" :class="{'btn-secondary':teamSize != members.length, 'btn-warning':isMemberCount(teamSize)}"  v-for="teamSize in allowedTeamSizes" @click="setMemberCount(teamSize)">{{ teamSize }}</button>
                        </div>

                        <div v-for="member, i in members" class="row border  m-1 pt-3 bg-white">
                            <div class="col-sm-1 text-center pt-2">
                                <span class="fa-stack fa-lg">
                                    <i class="fa fa-user fa-stack-2x text-warning"></i>
                                    <strong class="fa-stack-1x pt-2">{{ i + 1 }}</strong>
                                </span>
                            </div>
                            <div class="col bg-white">
                                <div class="row">
                                    <div class="col-4">
                        <TextInput layout="material" label="Navn" v-model="member.name" />
                        <TextInput layout="material" label="Adresse" v-model="member.address" />
                        <TextInput layout="material" label="Postnummer" v-model="member.postalCode" />
                                    </div>
                                    <div class="col-4">
                        <TextInput layout="material" label="E-mailadresse" v-model="member.mail" type="email" />
                        <TextInput layout="material" label="Eget telefonnummer" v-model="member.phone" />
                        <div class="alert alert-warning p-1" role="alert"><i class="fas fa-mobile-alt"></i> <small> Kun hvis telefon medbringes på Nathejk</small></div>
                                    </div>
                                    <div class="col-4">
                                        <div class="row">
                                            <div class="col">
                                                
                        <!--
                        <DateInput layout="material" label="Fødselsdag" v-model="member.birthday" />
                        -->
                        <v-date-picker v-model="member.birthday" mode="date" class="flex-grow"  :model-config="modelConfig">
  <template v-slot="{ inputValue, inputEvents }">
                        <!-- TextInput layout="material" label="Fødselsdag" :value="inputValue" v-on="inputEvents" type="text"  /-->
    <input
      class="px-2 py-1 border rounded focus:outline-none focus:border-blue-300 flex-grow"
      :value="inputValue"
      v-on="inputEvents"
      placeholder="fødselsdag"
    />
  </template>
</v-date-picker>
                        <!--v-date-picker v-model="dato" class="flex-grow" odel-config="modelConfig">
                                                    <template v-slot="{ inputValue, inputEvents }">
                        <TextInput layout="material" label="Fødselsdag" v-model="member.birthday" type="text" format="date" />
                                                    </template>
                                                </v-date-picker -->
                                            </div>
                                            <div class="col">
                                                <div class="form-check">
    <input type="checkbox" class="form-check-input" :id="member.memberId+'-returning'" v-model="member.returning">
    <label class="form-check-label" :for="member.memberId+'-returning'">har deltaget før</label>
  </div>
                                            </div>
                                        </div>
<!--
                                                <DateInput />
                                                <v-date-picker v-model="date" class="flex-grow">
                        <TextInput layout="material" label="Fødselsdag"  />
       </v-date-picker>
                                        <v-date-picker v-model="date" :popover="{ placement: 'bottom', visibility: 'click' }"><button>hb</button></v-date-picker>
                                         <label
      class="block text-gray-700 text-sm font-bold mb-2"
      for="date">
      Select Date Range
    </label>
    <div class="flex w-full">
      <v-date-picker v-model="date" class="flex-grow">
        <input slot-scope="{ inputProps, inputEvents }" v-bind="inputProps" v-on="inputEvents">
      </v-date-picker>
      <button type="button" @click="date = null">Clear</button>
    </div>
-->
                        <TextInput layout="material" label="Forældre telefon" v-model="member.phoneParent" />
                        <div class="alert alert-warning p-1" role="alert"><i class="fas fa-user-tie"></i> <small> Nathejk skal kunne kontakte dette forældre telefon på løbet, hvis situationen kræver det</small></div>
                                    </div>
                                </div>
                            </div>
                        </div>

                    </fieldset>
        
    </div>
    </div>
    </div>
    <div class="row pt-2">
        <div class="col d-flex flex-row-reverse">
                  <button type="submit" class="btn btn-warning">videre &raquo;</button>
        </div>
    </div>
    </form>
    </div>
</template>

<style lang="scss">
</style>

<script>
import Card from '@/components/Card.vue'
import Headline from '@/components/Headline.vue'
import TextInput from '@/components/TextInput.vue'
import DateInput from '@/components/DateInput.vue'
import moment from 'moment'
import axios from 'axios'

const member = {
    name: 'Anders And',
    address: 'Paradisæblevej 13',
    postalCode: '1313 Andeby',
    phone: '12345678',
    phoneParent: '98765432',
    birthday: '1999-01-01',
    returning: true,
}

export default {
    components: { Card, Headline, TextInput, DateInput },
    data: () => ({
        modelConfig: {
            type: 'string',
            mask: 'YYYY-MM-DD', // Uses 'iso' if missing
        },
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
        allowedTeamSizes: [3,4,5,6,7],
        team:null,
        members:[],
        team1: {
            teamId: '',
            name: 'Team Smølf',
            groupName: 'grp',
            korps: '',
            advspejdNumber: '',
            contactName: '',
            contactAdresse: '',
            contactPostalCode: '',
            contactPhone: '',
            contactMail: '',
            contactRole: '',
            members: [member, member, member],
        },
        date: moment(),
        dato: new Date(),
    }),
        /*
    async mounted() {
        try {
            const rsp = await axios.get(window.envConfig.API_BASEURL + '/api/team/' + this.$route.params.id)
            if (rsp.status == 200) {
                this.team = rsp.data
                if (rsp.data.members) {
                    this.members = rsp.data.members
                }
            }

        } catch(error) {
            console.log("error happend", error)
            throw new Error(error.response.data)
        }
    },
        */
    filters: {
        capitalize: function (value) {
            if (!value) return ''
            value = value.toString()
            return value.charAt(0).toUpperCase() + value.slice(1)
        },
        dateFormat: function(value) {
            return moment(String(value)).format('MM/DD/YYYY')
        },
    },
    methods: {
        setMemberCount: function(c) {
            while (this.members.length < c) {
                this.members.push({member});
            }
            while (this.members.length > c) {
                this.members.pop();
            }
            console.log(this.members)
        },
        isMemberCount(c) {
            //if (!this.team || !this.team.members) return false
            return c == this.members.length;
        },
        submit: async function(e) {
            e.preventDefault();
            e.stopPropagation();
            console.log('saving', this.team)
            const form = e.target || e.currentTarget
            try {
                this.team.members = this.members
                const rsp = await axios.post(window.envConfig.API_BASEURL + '/api/team', this.team)
                if (rsp.status == 200 && rsp.data.OK) {
                    this.$router.replace({ name: "thankyou", params: { id: this.team.teamId, state:'pay', type: this.team.type } });
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }
            console.log("done")
        },
    },
}
</script>
