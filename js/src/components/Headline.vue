<template>
    <div class="headline">
        <h1>{{ title }} <small>{{ subtitle }}</small></h1>
        <hr />
    </div>
</template>

<style lang="scss">
h1 {
    font-size: 2.3rem;
    font-family: "Helvetica Neue",Helvetica,Arial,sans-serif;
}
h1 small {
    font-size:65%;
    color:#999;
}
</style>

<script>
export default {
    props: {
        title: String,
        subtitle: String,
    },
}
</script>
