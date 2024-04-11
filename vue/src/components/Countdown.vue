<script setup>
import { ref, computed, defineProps, onMounted, onBeforeUnmount } from "vue";

const props = defineProps({
  time: { type: String, required: true},
  'class': { type: String },
})

const time = {
    day: 24 * 60 * 60 * 1000,
    hour: 60 * 60 * 1000,
    minute: 60 * 1000,
    second: 1000,
}
const targetTime = computed(() => new Date(props.time).getTime());
const currentTime = ref(new Date().getTime());
const timeRemaining = computed(() => targetTime.value - currentTime.value);

const days = computed(() => Math.floor( timeRemaining.value / time.day ));
const hours = computed(() => Math.floor(timeRemaining.value / time.hour ) % 24 );
const minutes = computed(() => Math.floor(timeRemaining.value / time.minute) % 60 );
const seconds = computed(() => Math.floor(timeRemaining.value / time.second ) % 60 );

let intervalId;

onMounted(() => {
    intervalId = setInterval(() => { currentTime.value = new Date().getTime() }, 1000);
});

onBeforeUnmount(() => {
    clearInterval(intervalId);
});

</script>

<template>
    <div :class="class">
        <slot :days="days" :hours="hours" :minutes="minutes" :seconds="seconds">{{ days }}:{{ hours }}:{{ minutes }}:{{ seconds }}</slot>
    </div>
</template>

