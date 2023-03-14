<template>

<ul class="nav nav-tabs mt-2">
  <li class="nav-item">
      <router-link class="nav-link" :class="{active:active=='signup'}" :to="{ name: 'settings-signup' }">Deltagerbegrænsninger</router-link>
      <!--a class="nav-link" :class="{active:active=='signup'}" href="">Deltagerbegrænsninger</a-->
  </li>
  <li class="nav-item">
      <router-link class="nav-link" :to="{ name: 'settings-signup' }">Indbydelser</router-link>
  </li>
  <!--li class="nav-item">
    <a class="nav-link disabled">Disabled</a>
  </li-->
  <li class="nav-item dropdown">
    <a class="nav-link dropdown-toggle" :class="{active:active=='mail'}" data-toggle="dropdown" href="#" role="button" aria-expanded="false">Emailskabeloner</a>
    <div class="dropdown-menu">
      <router-link class="dropdown-item" :to="{ name: 'settings-mail', params: { slug: 'signup' } }"><i class="fas fa-fw fa-envelope-open"></i> Tilmelding begyndt</router-link>
      <router-link class="dropdown-item" :to="{ name: 'settings-mail', params: { slug: 'updated' } }"><i class="fas fa-fw fa-envelope-open"></i> Deltageroplysninger opdateret</router-link>
      <router-link class="dropdown-item" :to="{ name: 'settings-mail', params: { slug: 'payment' } }"><i class="fas fa-fw fa-envelope-open"></i> Betaling modtaget</router-link>
      <div class="dropdown-divider"></div>
      <a class="dropdown-item" href="#"><i class="fas fa-fw fa-mobile-alt"></i> SMS</a>
    </div>
  </li>
</ul>

</template>

<style lang="scss">
</style>

<script>

export default {
    props: {
        active: String,
    },
}
</script>
