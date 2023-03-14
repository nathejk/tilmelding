<template>
    <CollapsibleCard :title="title" :open=isOpen @toggled="v => isOpen = v">
    <template #actions>
        <div class="d-flex flex-row-reverse float-right">
            <button type="button" :class="{'btn-outline-danger':!isDeleted, 'btn-danger':isDeleted}" class="btn btn-sm py-0 px-1 ml-1" :disabled="!deletable" @click.stop="deleted"><i class="fas fa-fw fa-xs fa-trash"></i></button>
            <button type="button" :class="{'btn-outline-secondary':!isOpen, 'btn-secondary':isOpen}" class="btn btn-sm py-0 px-1 ml-1"><i class="fas fa-fw fa-xs fa-edit"></i></button>
        </div>
    </template>
    <template #title>
        <div class="row">
            <div class="col-1 px-0"><i class="fas fa-fw fa-lg fa-user"></i></div>
            <div class="col" :class="{deleted:isDeleted}">{{ member.name }}</div>
        </div>
    </template>
    <template #summary>
        <div class="row">
            <div class="col-1 px-0"><i class="fas fa-fw fa-lg fa-user"></i></div>
            <small class="col-4" :class="{deleted:isDeleted}">{{ member.name }} ({{ member.birthday | age }} år)<br>{{ member.email }}<br>+45 {{ member.phone }}</small>
            <small class="col-4" :class="{deleted:isDeleted}">{{ member.address }}<br>{{ member.postalCode }} {{ member.city }}<br>+45 {{ member.contactPhone }}</small>
        </div>
    </template>
    <form>
      <div class="row">
        <div class="col">
          <div class="form-group mb-1">
            <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Navn</label>
            <input type="text" class="form-control form-control-sm py-0" v-model="member.name">
          </div>
          <div class="form-group mb-1">
            <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">E-mail</label>
            <input type="text" class="form-control form-control-sm py-0" v-model="member.email">
          </div>
          <div class="form-group mb-1">
            <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Telefonnummer</label>
            <input type="text" class="form-control form-control-sm py-0" v-model="member.phone">
          </div>
          <div class="form-group mb-1">
            <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Fødselsdag</label>
            <DatePicker v-model="member.birthday" initial-view="year" first-day-of-week="mon" :bootstrap-styling=true input-class="form-control form-control-sm py-0"></DatePicker>
          </div>
        </div>
        <div class="col">
          <div class="form-group mb-1">
            <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Adresse</label>
            <input type="text" class="form-control form-control-sm py-0" v-model="member.address">
          </div>
          <div class="row">
            <div class="col-4">
              <div class="form-group mb-1">
                <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Postnr.</label>
                <input type="text" class="form-control form-control-sm py-0" v-model="member.postalCode">
              </div>
            </div>
            <div class="col pl-0">
              <div class="form-group mb-1">
                <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">By</label>
                <input type="text" class="form-control form-control-sm py-0" v-model="member.city">
              </div>
            </div>
          </div>
          <div class="form-group mb-1">
            <label class="mb-0 py-0 text-uppercase" style="font-size:0.7rem">Forældres telefonnummer</label>
            <input type="text" class="form-control form-control-sm py-0" v-model="member.contactPhone">
          </div>
          <div class="form-check mt-4">
              <input class="form-check-input" type="checkbox" value="" :id="'member-returning-'+_uid" v-model="member.returning">
              <label class="form-check-label mb-0 py-0 text-uppercase" style="font-size:0.7rem" :for="'member-returning-'+_uid">Har deltaget før</label>
          </div>
        </div>
      </div>
    </form>
  </CollapsibleCard>
</template>

<style lang="scss">
.deleted {
      text-decoration: line-through;
      color: #999;
}
</style>

<script>
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import DatePicker from '@sum.cumo/vue-datepicker'

export default {
    components: { CollapsibleCard, DatePicker },
    props: {
        title: String,
        open: Boolean,
        member: Object,
        deletable: Boolean,
    },
    data: () => ({
        isOpen: false,
        isDeleted: false,
    }),
    methods: {
        deleted() {
            this.isDeleted = !this.isDeleted
            this.$emit('deleted', this.member, this.isDeleted)
        }
    },
    filters: {
        age: function (value) {
            if (!value) return '-'
            return new Date(new Date() - new Date(value)).getFullYear() - 1970
        },
    },
    mounted() {
        this.$nextTick(function () {
            this.isOpen = this.open
        })
    },
    watch: {
        open: function(newValue, oldValue) {
            this.isOpen = newValue;
        },
    },
}
</script>
