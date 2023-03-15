<template>
    <div>
        <div class="bg">
            <Countdown v-if="now.isBefore(countdownDate)" :title="title" :countdownDate="countdownDate" />
            <div v-else class="container v-center">
                <div class="row v-center">
                <div class="col text-right">
                    <a href="/spejder" class="btn btn-lg btn-orange p-3 px-5">Tilmeld patrulje <i class="fas fa-angle-double-right pl-1"></i></a>


                </div>
                <div class="col text-left">
                    <a href="/senior" class="btn btn-lg btn-orange p-3 px-5">Tilmeld seniorklan <i class="fas fa-angle-double-right pl-1"></i></a>
                </div>
                </div>
            </div>
        </div>
            <div class="container py-3">
                <div class="card-deck">

                    <Card layout="compact" color="blue" title="SPEJDERE" subtitle="12-16 år">
                    <p><strong>REGLER FOR PATRULJER</strong></p>
                    <ul class="pl-3">
                        <li>Der skal mindst være minimum 3 deltagere på holdet - og max 7</li>
                        <li>Ingen er under 12 år.</li>
                        <li>Ingen er fyldt 17 år.</li>
                        <li>Holdets gennemsnitsalder skal være mindst 13 år.</li>
                        <li>Tilmelding er først gældende, når beløbet er registreret på Nathejks konto.</li>
                    </ul>
                    <p></p>

                    </Card>
    <div class="card borderless">
        <div class="card-body p-0">
            <p class="card-text">
                    <div class="d-flex justify-content-center">
                   <iframe width="350" height="200"  src="https://www.youtube.com/embed/pb5-Be3IwTA" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe> 
                    </div>
                    <div class="mt-3 alert alert-warning " role="alert">
      <strong>Spørgsmål?</strong> Har I spørgsmål til jeres tilmelding, så skriv til tilmeld@nathejk.dk.
    </div>
            </p>
        </div>
    </div>
                    <Card layout="compact" color="red" title="SENIORER" subtitle="+16 år">
                    <p><strong>REGLER FOR SENIORER</strong></p>
                    <ul class="pl-3">
                    <li>
                        Et sjak kan være op til 5 personer.</li>
                    <li>Seniorer skal være fyldt 16 år.</li>
                    <li>Prisen for deltagelse er 250 kr. inkl forplejning.</li>
                    <li>Alle seniorer skal medbringe cykler.</li>
                    <li>Tilmelding er først gældende, når beløbet er registreret på Nathejks konto.</li>
                    </ul>
                    <p>Nødvendige oplysninger ved tilmelding er klan navn, gruppe navn samt antal seniorer - navne og telefonnumre på de deltagende seniorer kan indtastes efterfølgende.</p>
                    </Card>
                </div>
            </div>
    </div>
</template>

<style lang="scss">
.v-center {
    margin-top: auto;
margin-bottom: auto;
}
.btn-orange {
    background: #ff7032;
    backgroundimage: linear-gradient(to bottom,#ff7032 0, #ee6025 100%);
    color:#eee;
    font-weight:700;
    &:hover {
        background:#dd6025;
    }
}
.bg {
    background: url("/img/bg.jpg") no-repeat center center ;
      background-size: cover;
    min-height:310px;
        display:flex;
}
</style>

<script>
//  import '@/assets/main.scss'
import Countdown from '@/components/Countdown.vue'
import Card from '@/components/Card.vue'
import moment from 'moment'
import axios from 'axios'

export default {
    data: () => ({
      title: 'Nathejk 2023',
      now: moment(),
      api: Object,
      tick: null,
    }),
    components: { Countdown, Card },
    computed: {
        state() {
            return this.$route.params.state
        },
        countdownDate() {
            return moment(this.api.signupStart)
        },
        countdownActive() {
            //return this.countdownDate.isBefore(moment());
            return moment().isBefore(this.countdownDate);
        }
    },
    methods: {
      afterEnter() {
        // Scroll to top on page switch
        window.scrollTo({top: 0, behavior: 'smooth'})
      },
      async load() {
            try {
                const rsp = await axios.get('/api/frontpage')
                if (rsp.status == 200) {
                    this.api = rsp.data;
                }
            } catch(error) {
                console.log("error happend", error)
                throw new Error(error.response.data)
            }

      },
    },
    async mounted() {
        this.load()
        this.tick = setInterval(function () {
            this.now = moment();
        }.bind(this), 1000)
    },
    beforeDestroy() {
        clearInterval(this.tick);
    }
}
</script>
