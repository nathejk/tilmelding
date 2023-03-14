<template>
    <div class="card" :class="[layout, color]">
        <div class="card-body">
            <h5 v-if="title" class="card-title">{{ title }}</h5>
            <h6 v-if="subtitle" class="card-subtitle mb-2 text-muted">{{ subtitle }}</h6>
            <p class="card-text">
                <slot />
            </p>
        </div>
        <img v-if="img" :src="img" lass="card-img-bottom">
    </div>
</template>

<style lang="scss">
.card.compact {
    .card-title {
        font-size:36px;
        font-weight:500;
        display:inline-block;
        padding-right:1rem;
    }
    .card-subtitle {
        font-size:23px;
        font-weight:400;
        display:inline-block;
    }
    .card-text {
        font-size:14px;
    }
    background-image: linear-gradient(to bottom,#e8e8e8 0,#f5f5f5 100%);
}
.card.borderless {
    border:0;
}
.card.blue {
    color: rgb(58, 135, 173);
}
.card.red {
    color: #b94a48;
}
</style>

<script>

export default {
    props: {
        img: String,
        title: String,
        subtitle: String,

        layout: String,
        color: String,
    },
}
</script>
