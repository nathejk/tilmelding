<template>
    <div>
        <router-link :to="{ name: type+'-view', params: { id: teamId } }">Tilbage</router-link>
        State {{ state }} {{ $route.params.id }}
    </div>
</template>

<style lang="scss">
</style>

<script>
export default {

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
