<script setup>
const model = defineModel()
const props = defineProps(['options'])


function update() {
  model.value++
}
</script>

<template>
    <Fieldset class="mt-3" legend="Merchandise">
        <div class="grid grid-flow-col auto-cols-auto">
            <div>
                <p class="m-0">Vi sælger vores fulde sortiment af merchandise om søndagen på selve Nathejk, men hvis I vil have den helt særlige og eksklusive <span class="font-bold">års t-shirt</span> med et særligt tryk på ryggen som relaterer til årets Nathejk så har I kun chancen online. <span class="font-bold">Års t-shirten</span> skal bestilles og betales her inden August måned, hvorefter den vil blive produceret og udleveret sammen med det øvrige salg af merchandise om søndagen på Nathejk,</p>
            <div class="flex flex-col" v-if="props.options">
                <FloatLabel class="mt-7">
                    <label for="shop-tshirt">Køb t-shirt</label>
                    <Dropdown v-model="model" inputId="shop-tshirt" :options="props.options" optionValue="slug" optionLabel="label"  class="w-full filled md:w-14rem" />
                </FloatLabel>
            </div>
            </div>
            <Image src="/tshirts-web-2025.png" alt="t-shirt" width="250" preview />
        </div>
    </Fieldset>
</template>

<style scoped>
</style>
