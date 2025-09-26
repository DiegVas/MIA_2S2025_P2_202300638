<script setup>
import Back_button from "../components/common/Back_button.vue";
import Disk_layout from "../components/layout/Disk_layout.vue";
import Partition_layout from "../components/layout/Partition_layout.vue";
import SystemFiles from "./SystemFiles.vue";
import { useRouter } from "vue-router";
import { startTypewriter } from "../services/startWriters.js";
import { usedisksStore } from "../store/disks.js";
import { useFileStore } from "../store/files.js";
import { ref } from "vue";
import { computed } from "vue";
import { watch } from "vue";

const titles = {
  disco: "disco",
  particion: "partición",
};

const typewriterText = ref("");
const typingIndex = ref(0);
const typingInterval = ref(null);

const {
  selectedDisk,
  clearSelectedDisk,
  selectedPartition,
  clearSelectedPartition,
} = usedisksStore();

const $router = useRouter();

const { clearPath } = useFileStore();

const goBack = () => {
  if (selectedDisk.value === null) {
    $router.push("/");
    return;
  } else if (selectedPartition.value === null) {
    clearSelectedDisk();
    return;
  } else {
    clearSelectedPartition();
    clearPath();
    return;
  }
};

// ? Detecta cambios en selectedDisk y actualiza el título
watch(
  () => selectedDisk.value,
  (newVal) => {
    if (newVal) {
      startTypewriter(titles.particion, typingInterval, typewriterText);
    } else {
      startTypewriter(titles.disco, typingInterval, typewriterText);
    }
  },
  { immediate: true }
);
</script>

<template>
  <div class="page-container flex self-center justify-center w-full relative">
    <Back_button :backFunction="goBack" />
    <Transition name="fade-disk" mode="out-in">
      <SystemFiles v-if="selectedPartition" class="content" />
      <div class="content" v-else>
        <div class="typewriter-title">
          Seleccionar {{ typewriterText
          }}<span class="typewriter-cursor">|</span>
        </div>
        <Transition name="fade-disk" mode="out-in">
          <div :key="selectedDisk ? 'partition' : 'disk'">
            <component :is="selectedDisk ? Partition_layout : Disk_layout" />
          </div>
        </Transition>
      </div>
    </Transition>
  </div>
</template>
<style>
.typewriter-title {
  font-size: 2rem;
  font-weight: bold;
  margin-bottom: 2rem;
  font-family: monospace;
  min-height: 2.5rem;
  letter-spacing: 1px;
}

.typewriter-cursor {
  display: inline-block;
  width: 1ch;
  animation: blink 1s steps(1) infinite;
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}

.page-container {
  padding: 20px;

  min-height: 100vh;
}

.content {
  padding: 3vw 2vw;
  width: 80%;
  margin: 0 auto;
  position: relative;
}
.fade-disk-enter-active,
.fade-disk-leave-active {
  transition: opacity 0.5s ease;
}
.fade-disk-enter-from,
.fade-disk-leave-to {
  opacity: 0;
}
</style>
