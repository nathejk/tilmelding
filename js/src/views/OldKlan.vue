<template>
    <div class="container py-3">
        <Headline title="Tilmelding" subtitle="indtast oplysninger" />
        <form enctype="multipart/form-data" action="" method="post">

        <div class="row">
            <div class="col-8">
                <div class="p-3 bg-light border rounded">
                    <fieldset id="team">
                        <legend><i class="fas fa-map-signs"></i> Klanoplysninger</legend>
                        <TextInput label="Klan" v-model="team.name" />
                        <TextInput label="Gruppe og division" v-model="team.gruppe" />
                        <div class="control-group form-group row">
                          <label class="col-sm-3 col-form-label col-form-label-sm">Korps</label>
                          <div class="col-sm-9">
                              <select v-model="team.korps" class="form-control form-control-sm">
                                <option v-for="option in korps" v-bind:value="option.value">{{ option.text }}</option>
                              </select>
                          </div>
                        </div>
                        <div class="control-group form-group row">
                            <label class="col-sm-3 col-form-label col-form-label-sm">Antal medlemmer</label>
                            <div class="col-sm-9">
                                <div class="form-check form-check-inline" v-for="c in memberCounts">
                                    <input class="form-check-input" type="radio" name="memberCount" :id="'memberCount'+c" v-model="memberCount" :value="c">
                                    <label class="form-check-label" :for="'memberCount'+c">{{ c }} <i v-for="i in c" class="fas fa-user"></i></label>
                                </div>
                          </div>
                        </div>
                    </fieldset>
                </div>
            </div>
        </div>

        <div class="row">
            <div class="col">
                <button v-if="memberCountChanged" :disabled="!teamValid" class="btn btn-primary float-right" type="submit">Videre til betaling</button>
            </div>
        </div>

        <div class="row" v-if="false">
            <div class="col">
                <div class="p-3 bg-light border rounded">
                    <fieldset>
                        <h4 style="display:inline-block"><i class="fas fa-users"></i> Deltagere</h4>

                        <div class="btn-group mx-4" role="group" aria-label="First group">
                            <button type="button" class="btn" :class="{'btn-secondary':memberCount != members.length, 'btn-warning':isMemberCount(memberCount)}"  v-for="memberCount in memberCounts" @click="setMemberCount(memberCount)">{{ memberCount }}</button>
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
                        <TextInput layout="material" label="Adresse" v-model="adresse" />
                        <TextInput layout="material" label="Postnummer" v-model="adresse" />
                                    </div>
                                    <div class="col-4">
                        <TextInput layout="material" label="E-mailadresse" type="email" v-model="member.name" />
                        <TextInput layout="material" label="Eget telefonnummer" v-model="adresse" />
                        <div class="alert alert-warning p-1" role="alert"><i class="fas fa-mobile-alt"></i> <small> Kun hvis telefon medbringes på Nathejk</small></div>
                                    </div>
                                    <div class="col-4">
                                        <div class="row">
                                            <div class="col">
                                                <v-date-picker v-model="date" class="flex-grow">
                        <TextInput layout="material" label="Fødselsdag" v-model="date" />
       </v-date-picker>
                                            </div>
                                            <div class="col">
                                                <div class="form-check">
    <input type="checkbox" class="form-check-input" id="exampleCheck1">
    <label class="form-check-label" for="exampleCheck1">har deltaget før</label>
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
                        <TextInput layout="material" label="Forældre telefon" v-model="adresse" />
                        <div class="alert alert-warning p-1" role="alert"><i class="fas fa-user-tie"></i> <small> Nathejk skal kunne kontakte dette forældre telefon på løbet, hvis situationen kræver det</small></div>
                                    </div>
                                </div>
                            </div>
                        </div>

                    </fieldset>
        
    </div>
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
        memberCount: 1,
        memberCounts: [1,2,3,4,5],
        team: {
            name: 'Team Smølf',
            gruppe: 'grp',
            selected: 'dds',
            contactName: '',
            contactAdresse: '',
            contactPostalCode: '',
            contactPhone: '',
            contactMail: '',
            contactRole: '',
            members: [member, member, member],
        },
        date: moment(),
    }),
    computed: {
        // get only
        teamValid: function () {
            return this.klan != "" && this.gruppe != ""
        },
        memberCountChanged: function () {
            return this.memberCount != this.members.length
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
            return c == this.members.length;
        },
    },

}
</script>
