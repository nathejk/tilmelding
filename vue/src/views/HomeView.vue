<script setup>
import { ref, onMounted } from 'vue'
import Navigation from '@/components/Navigation.vue'
import Countdown from '@/components/Countdown.vue'

const config = ref({ timeCountdown: "2024" });
const videoplayer = ref(null);

async function loadConfig() {
    try {
        const response = await fetch("/api/home");
        if (!response.ok) {
            throw new Error("HTTP status " + response.status);
        }
        const data = await response.json();
        return data.config
    } catch (error) {
        console.log("Failed loading config", error);
    }
    return {}
}

function playRandom(videos) {
    if (videos.length == 0) return
    videoplayer.value.src = videos[videos.length * Math.random() | 0];
    videoplayer.value.load();
    videoplayer.value.play();
    videoplayer.value.classList.remove('fading');
    setTimeout(() => {
        videoplayer.value.classList.add('fading');
    }, (videoplayer.value.duration / videoplayer.value.playbackRate - 1) * 1000)
}


onMounted(async () => {
    config.value = await loadConfig()
    videoplayer.value.addEventListener("ended", () => playRandom(config.value.videos), true);
    videoplayer.value.addEventListener("error", () => playRandom(config.value.videos), true);
    playRandom(config.value.videos);
})

function download(url) {
      // create element <a> for download PDF
      const link = document.createElement('a');
      link.href = url;
      link.target = '_blank';
      //link.download = this.pdfFileName;

      // Simulate a click on the element <a>
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    }
</script>

<template>
  <Navigation class="dark" />
  <main class="">
    <div class="bg-slate-900">
      <div class="container mx-auto relative aspect-video">
        <video ref="videoplayer" autoplay muted clas="transition fading"></video>
        <Countdown class="absolute inset-0 text-slate-200 p-10 text-xl" :time="config.timeCountdown" v-slot="{days, hours, minutes, seconds}">
            Vi vågner om
            <span v-if="days"><span class="inline-block w-8 text-right">{{ days }}</span> dage</span>
            <span v-if="days || hours"><span class="inline-block w-8 text-right">{{ hours }}</span> timer</span>
            <span class="inline-block w-8 text-right">{{ minutes }}</span> minutter og
            <span class="inline-block w-8 text-right">{{ seconds }}</span> sekunder
            og åbner tilmeldingen.
        </Countdown>
      </div>
    </div>

    <div class="bg-yellow-500">
      <div class="container mx-auto py-5">
        <h1 class="mb-5 font-nathejk font-bold text-5xl text-slate-700">REGLER</h1>

        <div class="grid grid-cols-2">
          <Card class="col mr-5 bg-yellow-400">
            <template #title>Patruljer <small class="font-thin pl-2">12-16 år</small></template>
            <template #content>
              <ul class="list-disc mx-5 mb-5">
                <li>Der skal være minimum 3 deltagere på holdet - og max 7</li>
                <li>Ingen er under 12 år.</li>
                <li>Ingen er fyldt 17 år.</li>
                <li>Holdets gennemsnitsalder skal være mindst 13 år.</li>
                <li>Tilmelding er først gældende, når beløbet er registreret på Nathejks konto.</li>
              </ul>
              <p>Pris 200 kroner per spejder.</p>
            </template>
          </Card>

          <Card class=" col ml-5 bg-yellow-400">
            <template #title>Seniorer <small class="font-thin pl-2">+15 år</small></template>
            <template #content>
              <ul class="list-disc mx-5">
                <li>Et sjak kan være op til 4 personer.</li>
                <li>Seniorer skal være fyldt 16 år.</li>
                <li>Prisen for deltagelse er 250 kr. inkl forplejning.</li>
                <li>Alle seniorer skal medbringe cykler.</li>
                <li>Tilmelding er først gældende, når beløbet er registreret på Nathejks konto.</li>
              </ul>
            </template>
          </Card>
        </div>

      </div>
    </div>
    

    <div class="bg-slate-100">
      <div class="container mx-auto py-5">
        <h1 class="mb-5 font-nathejk font-bold text-5xl text-slate-700">INVITATIONER</h1>
        <Card class="">
          <template #content>
            <div class="flex justify-around ">
              <div class="self-center">
                  <Button label="Hent spejderinvitation" icon="pi pi-download" size="large" severity="warning" @click="download('/invitation/Patruljeinvitation_Nathejk2024.pdf')" />
              </div>
              <Image src="/invitation/Invitationer2024_collage.png" alt="" />
              <div class="self-center">
                  <Button label="Hent seniorinvitation" icon="pi pi-download" size="large" severity="warning" @click="download('/invitation/Seniorinvitation_Nathejk2024.pdf')" />
              </div>
            </div>
          </template>
        </Card>
      </div>
    </div>

    <div class="bg-slate-700">
      <div class="container mx-auto py-5">
        <h1 class="mb-5 font-nathejk font-bold text-5xl text-yellow-500">TILMELDINGSPROCEDURE</h1>
        <div class="text-2xl text-slate-300 leading-loose">
          <p>Tilmeldingen består af 4 trin</p>
        <Card class="bg-slate-500">
          <template #content>
          <ol class="list-decimal text-yellow-500 ml-10">
            <li><span class="text-slate-300 pl-5">Indtast e-mailadresse og klik på link i tilsendt e-mail</span></li>
            <li><span class="text-slate-300 pl-5">Indtast telefonnummer og modtga en PIN-kode</span></li>
            <li><span class="text-slate-300 pl-5">Indtast patruljenavn, spejdergruppe, kontaktperson og ønskede antal deltagere</span></li>
            <li><span class="text-slate-300 pl-5">Gennemfør betaling via MobilePay</span></li>
          </ol>
          </template>
        </Card>
        </div>
      </div>
    </div>

  </main>
  <footer class="bg-slate-900 text-slate-100 font-nathejk text-3xl text-center uppercase">
      <div class="p-52">Vi ses i mørket...</div>
  </footer>

</template>

<style>
.transition {
    transition: all 1s;
}

.transition.fading {
    opacity: 0;
}
</style>
