<template>
<div class="container py-3">

  <SettingsNavigation active="mail" />

  <div v-if="loading">
    <div class="d-flex justify-content-center"><i class="fas fa-spinner fa-5x fa-spin"></i></div>
  </div>
  <div v-else>

    <div class="row mt-3">
        <div class="col">
          <div class="input-group mb-3">
            <div class="input-group-prepend">
              <div class="input-group-text"><i class="fas fa-envelope-open"></i></div>
            </div>
            <input type="text" class="form-control" placeholder="Emne" v-model="subject">
          </div>
        </div>
    </div>
    <div class="row">
      <div class="col-6">

        <div class="card">
          <div class="card-header">
              <div class="d-flex justify-content-between">
                Skabelon
                <div>
                    <i v-if="error" class="fas fa-fw fa-exclamation-triangle text-danger ml-1" data-toggle="modal" data-target="#errorModal"></i>
                    <i class="fas fa-fw fa-code ml-1" data-toggle="modal" data-target="#exampleModal"></i>
                </div>
              </div>
          </div>
          <div class="form-group m-0"><textarea class="form-control border-0 h-100" v-model="template" rows="10"></textarea></div>
        </div>

      </div>
      <div class="col-6">

        <div class="card">
          <div class="card-header">Forhåndsvisning</div>
          <div class="card-body">
            <p class="card-text" style="white-space: pre-wrap">{{ preview }}</p>
          </div>
        </div>

      </div>
    </div>
    <div class="form-group row">
      <div class="d-flex justify-content-end col mt-3">
        <button type="button" class="btn btn-primary" @click="save">Gem skabelon</button>
      </div>
    </div>
  </div>

<div class="modal fade" id="exampleModal" tabindex="-1" aria-labelledby="exampleModalLabel" aria-hidden="true">
  <div class="modal-dialog modal-dialog-scrollable modal-lg">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title" id="exampleModalLabel">Eksempel på data til brug i skabelon</h5>
        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <pre class="modal-body"><small>{{ JSON.stringify(example, null, 2) }}</small></pre>
    </div>
  </div>
</div>

<div class="modal fade" id="errorModal" tabindex="-1" aria-labelledby="errorModalLabel" aria-hidden="true">
  <div class="modal-dialog modal-dialog-scrollable modal-lg">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title" id="errorModalLabel">Der er følgende fejl i skabelonen</h5>
        <button type="button" class="close" data-dismiss="modal" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <pre class="modal-body"><small>{{ error }}</small></pre>
    </div>
  </div>
</div>

</div>
</template>

<style lang="scss">
</style>

<script>
import SettingsNavigation from '@/components/SettingsNavigation.vue'
import Handlebars from 'handlebars';
import axios from 'axios'

export default {
    components: { SettingsNavigation },
    props: {
    },
    data: () => ({
        error:'',
        loaded: false,
        subject: String,
        template: String,
        example: Object,
    }),
    computed: {
        loading() {
            return !this.loaded
        },
        slug() {
            return this.$route.params.slug
        },
        preview() {
            const template = Handlebars.compile(this.template);
            let preview = '';
            try {
                preview = template(this.example);
            } catch(error) {
                this.error = error.message
                /*.replace(/[\u00A0-\u9999<>\&]/gim, function(i) {
                    return '&#' + i.charCodeAt(0) + ';';
                });*/
                return '...'
            }
            this.error = ''
            return preview;
        },
    },
    methods: {
        async load() {
            try {
                const rsp = await axios.get('/api/settings/mail/' + this.slug)
                if (rsp.status == 200) {
                    this.subject = rsp.data.subject
                    this.template = rsp.data.template
                    this.example = rsp.data.example
                    this.loaded = true
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }
        },
        async save() {
            delete(this.loaded)
                const rsp = await axios.put('/api/settings/mail', {slug:this.slug, subject:this.subject, template:this.template}, { withCredentials: true })
            this.load()
            return rsp.data.teamId
        },
    },
    async mounted() {
        this.load()

        this.$nextTick(function () {
        })
    },
        watch:{
    $route (to, from){
            delete(this.loaded)
            this.load()
    }
}
}
</script>
