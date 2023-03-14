<template>
  <div class="card">
    <div class="card-header" @click="toggle">
      <slot name="actions" />

      <Transition name="slide-down">
        <slot v-if="isOpen" name="title">{{ title }}</slot>
        <slot v-if="!isOpen" name="summary"><slot name="title">{{ title }}</slot></slot>
      </Transition>
    </div>

    <Transition name="slide-down">
      <div v-if="isOpen" class="card-body">
        <slot>empty</slot>
      </div>
    </Transition>
    </div>
  </div>
</template>

<style lang="scss">
.fade-enter-active, .fade-leave-active {
  transition: opacity .3s;
}
.fade-enter, .fade-leave-to  {
  opacity: 0;
}

.slide-down-enter-active,
.slide-down-leave-active {
  transition: max-height .3s ease-in-out;
}

.slide-down-enter-to,
.slide-down-leave {
  overflow: hidden;
  max-height: 1000px;
}

.slide-down-enter,
.slide-down-leave-to {
  overflow: hidden;
  max-height: 0;
}
</style>

<script>

export default {
    props: {
        title: String,
        open: Boolean,
    },
    data: () => ({
      isOpen: false,
    }),
    methods: {
        toggle() {
            this.isOpen = !this.isOpen
            this.$emit('toggled', this.isOpen)
        }
    },
    mounted() {
        this.$nextTick(function () {
            this.isOpen = this.open
        })
    },
    watch: {
        open: function(newValue, oldValue) {
                console.log("open changed", newValue, oldValue)
          this.isOpen = newValue;
        },
    },
}
</script>
