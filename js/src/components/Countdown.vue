<template>

    <section class="container pt-3">
        <div class="jumbotron text-center rounded-lg mt-10">
            <p>Tilmeldingen til <strong>{{ title }}</strong> åbner om</p>
            <h1 id="countdown" :class="{'sunset':countdownDate.diff(today, 'seconds') <= 10}">
                  <span>{{ days }}</span><small>dage</small>
                  <span>{{ hours }}</span><small>timer</small>
                  <span>{{ minutes }}</span><small>minut</small>
                  <span class="second">{{ seconds }}</span><small>sekunder</small>
              </h1>
        </div>
    </section>
</template>

<style lang="scss">
.jumbotron {
    p {
        font-size:30px;
        font-weight:300;
        font-family: "Helvetica Neue",Helvetica,Arial,sans-serif;
    }
    strong {
        font-weight:bold;
    }
    #countdown {
        font-size:4rem;
        font-weight:700;
        small {
            font-size:1.5rem;
            font-weight:normal;
            color:#999;
            padding-right:1rem;
        }
        span {
            display: inline-block;
            text-align:right;
            width:80px;
        }
        &.sunset span.second {
            color: #cc0000;
        }
    }

    backgroundimage: linear-gradient(to bottom,#e8e8e8 0,#f5f5f5 100%);
        background-color:rgba(240,240,240,0.85) ! important;
    margin-bottom:1rem;
}
</style>

<script>
import moment from 'moment'

export default {
    data: () => ({
        today: moment(),
        tick: null,
    }),
    props: {
        title: null,
        countdownDate: Object
    },
    computed: {
        days() {
            return Math.max(0, this.countdownDate.diff(this.today, 'days'))
        },
        hours() {
            return Math.max(0, this.countdownDate.diff(this.today, 'hours')) % 24
        },
        minutes() {
            return Math.max(0, this.countdownDate.diff(this.today, 'minutes')) % 60
        },
        seconds() {
            return Math.max(0, this.countdownDate.diff(this.today, 'seconds')) % 60
        }
    },
    mounted: function () {
        this.tick = setInterval(function () {
            this.today = moment();
        }.bind(this), 1000)
    },
    beforeDestroy() {
        clearInterval(this.tick);
    }
}
</script>
