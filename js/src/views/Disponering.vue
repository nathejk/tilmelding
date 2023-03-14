<template>
    <div class="container py-3">
        <div class="row">
            <div class="col-6">


        <div class="card mb-3" >
          <div class="card-header d-flex justify-content-between">
              <span><i class="fas fa-fw fa-lg fa-car-side"></i> Igangværende sager</span>
          </div>
          <div  class="card-body bg-gradient-secondary">
<table class="table table-hover table-sm">
  <thead>
    <tr>
      <th scope="col-8">Vogn</th>
      <th scope="col-8">Beskrivelse</th>
      <th scope="col">sidste action</th>
    </tr>
  </thead>
  <tbody>
      <tr v-for="h in hold">
        <td><small>{{ h.text }}</small></td>
        <td><div v-if="h.sos > 0">
                <i class="fas fa-fw fa-circle text-success"></i> Patrulje 75-4 har mistet en sovepose</div></td>
      <td><small>lør kl 23:45</small></td>
    </tr>
  </tbody>
</table>

          </div>
        </div>

            </div>
            <div class="col-6">

        <div class="card mb-3">
          <div class="card-header d-flex justify-content-between">
              <span>Ledig kapacitet</span>
              <small>
                <span class="pl-3"><i class="fas fa-fw fa-lg fa-car-side"></i> 2</span>
                <span class="pl-3"><i class="fas fa-fw fa-lg fa-ambulance"></i> 3</span>
                <span class="pl-3"><i class="fas fa-fw fa-lg fa-walking"></i> 5</span>
                <span class="pl-3"><i class="fas fa-fw fa-lg fa-utensils"></i> 1</span>
              </small>
          </div>
        </div>

        <div class="card mb-3">
          <div class="card-header"><i class="fas fa-fw fa-lg fa-clipboard-check"></i> Udisponerede sager</div>
          <div class="card-body bg-gradient-secondary">
                <table class="table table-hover table-sm">
  <thead>
    <tr>
      <th scope="col-8">Beskrivelse</th>
      <th scope="col-3">Oprettet</th>
      <th scope="col">sidste action</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><i class="fas fa-fw fa-circle text-success"></i> Patrulje 34-6 har ondt i fødderne</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="far fa-fw fa-circle"></i> Kaffe til LOK4</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="fas fa-fw fa-circle text-danger"></i> Post 2A er løbet tør for kort til spejderne</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="fas fa-fw fa-circle text-success"></i> Patrulje 34-6 har ondt i fødderne</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="far fa-fw fa-circle"></i> Kaffe til LOK4</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="fas fa-fw fa-circle text-danger"></i> Post 2A er løbet tør for kort til spejderne</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="fas fa-fw fa-circle text-success"></i> Patrulje 34-6 har ondt i fødderne</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="far fa-fw fa-circle"></i> Kaffe til LOK4</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="fas fa-fw fa-circle text-danger"></i> Post 2A er løbet tør for kort til spejderne</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="fas fa-fw fa-circle text-success"></i> Patrulje 34-6 har ondt i fødderne</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="far fa-fw fa-circle"></i> Kaffe til LOK4</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
    <tr>
      <td><i class="fas fa-fw fa-circle text-danger"></i> Post 2A er løbet tør for kort til spejderne</td>
      <td><small>lør kl 23:30</small></td>
      <td><small>lør kl 23:30</small></td>
    </tr>
  </tbody>
</table>
<button type="button" class="btn btn-outline-success">Ny sag</button>

          </div>
        </div>
            </div>
        </div>
    </div>
</template>

<style lang="scss">
</style>

<script>
export default {
    data: () => ({
        hold: [
                { text: 'Rover 1', sos:1, members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Rover 2', sos:1,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Rover 3', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Rover 4', sos:1,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Sam 1', sos:1,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Sam 2', sos:1,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Sam 3', sos:1,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Madbil', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Guide 1', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Guide 2', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Guide 3', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Guide 4', sos:1,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Guide 5', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Guide 6', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Teknik 1', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Teknik 2', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Skadestuebil', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
                { text: 'Bigfoot', sos:0,members:[{name:"A. And", tlf:"+45 12345678"}, {name:"Jens", tlf:"+45 12345678"}] },
        ]
    }),
    computed: {
        state() {
            return this.$route.params.state
        },
        teamId() {
            return this.$route.params.id
        },
        type() {
            return this.$route.params.type
        },

    },
}
</script>
