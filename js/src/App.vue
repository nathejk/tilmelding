<template>
  <div class="d-flex flex-column h-100">
    <navigation title="Nathejk 2023"/>
    <main role="main">
      <router-view></router-view>
    </main>
    <foooter />
  </div>
</template>

<style lang="scss">
@import "~bootstrap/scss/bootstrap";
html, body { height:100%; }
</style>

<script>
  import Navigation from '@/components/Navigation.vue'
  import Foooter from '@/components/Footer.vue'

  export default {
    components: { Navigation, Foooter },
  }
</script>
