<template>
    <div class="information">
        <div class="row align-items-center pt-5 clearfix">
            <div class="day col-5 text-right">{{selectedDate.format('D')}}</div>
            <div class="col-7">
                <div class="month">{{selectedDate.format('MMMM')}}</div>
                <div class="weekday">{{selectedDate.format('dddd')}}</div>
            </div>
        </div>
        <div class="week text-right mx-3 border-top">{{selectedDate.format('[uge] w, YYYY')}}</div>
    </div>
</template>

<style lang="scss">
.information {
    color:#2a4d69;
    background-color: #adcbe3;
    border-radius: 1.2rem 0 0 1.2rem;
    height: 100%;
    .day {
        font-size: 6em;
        font-weight: 200;
        line-height: 1;
    }
    .month{
        font-size: 2.5em;
        font-weight: 200;
        line-height: 1;
    }
    .weekday {
        font-size: 2.5em;
        font-weight: 600;
        line-height: 1;
    }
    .week {
        margin-top:2em;
        border-color:#2a4d69 ! important;
        font-size: 0.75em;
    }
}

@media (max-width: 768px) {
    .information {
        min-height: auto;
        padding-top: 2rem;
        padding-bottom: 2rem;
        border-radius: 2.5rem 2.5rem 0 0;
        .today {
            padding-top: 0;
        }
    }
}


</style>

<script>
import moment from 'moment-timezone'

export default {
    props: {
        selectedDate: Object,
        dateContext: Object,
        today: Object,
    },
    filters: {
        capitalize: function(value) {
            if (!value) return "";
            value = value.toString();
            return value.charAt(0).toUpperCase() + value.slice(1);
        }
    }
}
</script>
