<template>
  <header>
    <div class="navbar navbar-expand-lg navbar-dark bg-dark shadow-sm">
      <div class="container d-flex flex-row-reverse">
        <a class="navbar-brand" href="/" style="background:url(/img/logomoonsmall.png) no-repeat; padding-left:60px;"><strong>{{ title }}</strong></a>

        <button class="navbar-toggler" type="button" data-toggle="collapse" data-target="#navbarText" aria-controls="navbarText" aria-expanded="false" aria-label="Toggle navigation">
    <span class="navbar-toggler-icon"></span>
  </button>
  <!--
        <div class="collapse navbar-collapse" id="navbarNav">
    <ul class="navbar-nav ">
      <li class="nav-item dropdown">
        <a class="nav-link dropdown-toggle" href="#" id="navbarDropdown" role="button" data-toggle="dropdown" aria-expanded="false">
          Patrulje
        </a>
        <div class="dropdown-menu" aria-labelledby="navbarDropdown">
            <a class="dropdown-item" href="#"><i class="fas fa-fw fa-users"></i> Spætterne</a>
          <a class="dropdown-item" href="#"><i class="fas fa-fw fa-users"></i>Panterne</a>
          <div class="dropdown-divider"></div>
          <a class="dropdown-item" href="#"><i class="fas fa-fw fa-plus"></i> Tilmeld ny patrulje</a>
        </div>
      </li>
      <li class="nav-item active">
        <a class="nav-link" href="#">Stamoplysninger! xmllint --format - <span class="sr-only">(current)</span></a>
      </li>
      <li class="nav-item">
        <a class="nav-link" href="#">E-mails</a>
      </li>
      <li class="nav-item">
        <a class="nav-link" href="#">Betalingsoverblik</a>
      </li>
      <li class="nav-item">
        <a class="nav-link disabled">Diplom</a>
      </li>
    </ul>
  </div>
  -->
      </div>
    </div>
  </header>
</template>

<style lang="scss">
</style>

<script>
export default {
    props: {
        title: null
    }
}
</script>
