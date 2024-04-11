<template>
  <slot :example="example">[]</slot>
</template>

<script setup>
import { ref } from 'vue'

const example = ref('just some data')
</script>
