<script setup>
const model = defineModel()
const props = defineProps({
  // The sizes on offer, from config.tshirtSizes.
  options: { type: Array, default: null },
  // False once the year t-shirt is closed for sale, i.e. when
  // config.closedProducts contains 'tshirt.adult'. Defaults to closed so a slow
  // first load cannot flash a picker that will not work.
  open: { type: Boolean, default: false },
  // The size this owner actually has on an order, from helpers/order.orderedSize.
  // Read from the order rather than from the member record: while the sale is
  // closed an unpaid selection has been cancelled, and the member record would
  // still promise a shirt nobody will produce.
  orderedSize: { type: String, default: '' }
})

const sizeLabel = (slug) => {
  if (!slug) return ''
  for (const option of props.options || []) {
    if (option.slug === slug) return option.label
  }
  return slug.toUpperCase()
}
</script>

<template>
  <Fieldset class="mt-3" legend="Merchandise">
    <div class="grid grid-flow-col auto-cols-auto">
      <div>
        <p class="m-0">
          Vi sælger vores fulde sortiment af merchandise om søndagen på selve Nathejk, men hvis I
          vil have den helt særlige og eksklusive <span class="font-bold">års t-shirt</span> med et
          særligt tryk på ryggen som relaterer til årets Nathejk så har I kun chancen online.
          <span class="font-bold">Års t-shirten</span> skal bestilles og betales her inden August
          måned, hvorefter den vil blive produceret og udleveret sammen med det øvrige salg af
          merchandise om søndagen på Nathejk,
        </p>

        <div v-if="props.open" class="flex flex-col">
          <FloatLabel class="mt-7">
            <label for="shop-tshirt">Køb t-shirt</label>
            <Dropdown
              v-model="model"
              inputId="shop-tshirt"
              :options="props.options"
              optionValue="slug"
              optionLabel="label"
              class="w-full filled md:w-14rem"
            />
          </FloatLabel>
        </div>

        <!--
          Closed: no control at all rather than a disabled one — there is nothing
          left to choose.

          Only the banner is unconditional. The "you ordered X" line appears when
          this owner is a single person with a shirt on an order (crew, gøgler);
          the team views render one Shop for a whole roster, where per-member sizes
          belong in the member table, so there is deliberately no "you ordered
          nothing" counterpart — it would be a claim this component cannot make.
        -->
        <div v-else class="mt-7">
          <Message severity="info" :closable="false">
            Salget af årets t-shirt er lukket — t-shirtene er sendt i produktion.
          </Message>
          <p v-if="props.orderedSize" class="mt-3 m-0">
            Du har bestilt en t-shirt i størrelse
            <span class="font-bold">{{ sizeLabel(props.orderedSize) }}</span
            >. Den udleveres sammen med det øvrige merchandise om søndagen på Nathejk.
          </p>
        </div>
      </div>
      <Image src="/tshirt_2026.jpg" alt="t-shirt" width="250" preview />
    </div>
  </Fieldset>
</template>

<style scoped></style>
