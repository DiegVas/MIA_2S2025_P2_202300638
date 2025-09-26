<script setup>
import { useFileStore } from "../store/files.js";
import { ref, watch } from "vue";
import { startPathTypewriter } from "../services/startWriters.js";
import File_layout from "../components/layout/File_layout.vue";
import Directory_layout from "../components/layout/Directory_layout.vue";

const { path, fileContent } = useFileStore();

const typewriterPath = ref("");
const typingPathInterval = ref(null);

// Ejemplo de uso con watch sobre path
watch(
  () => [...path.value],
  (newVal, oldVal) => {
    startPathTypewriter(newVal, oldVal, typingPathInterval, typewriterPath);
  },
  { immediate: true }
);
</script>

<template>
  <div>
    <h1>Archivos del Sistema</h1>
    <div class="typewriter-path">{{ typewriterPath }}</div>

    <Transition name="fade" mode="out-in">
      <div :key="fileContent ? 'fileContent' : 'directory'">
        <component :is="fileContent ? File_layout : Directory_layout" />
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.type-icon {
  font-size: 80px;
  color: var(--color-secundary);
}

.file-name {
  margin: 8px 0 4px 0;
  font-size: 1.1rem;
  font-weight: bold;
  color: var(--color-text);
}
.typewriter-path {
  font-family: monospace;
  font-size: 1.2rem;
  color: var(--color-background);
  background-color: var(--color-primary);
  width: 80%;
  height: 40px;
  border-radius: 30px;
  padding: 8px 16px;
  display: flex;
  align-items: center;
  justify-content: start;
  position: relative;
}

.typewriter-path::after {
  content: "|";
  animation: blink-cursor 1s steps(1) infinite;
  margin-left: 2px;
  color: var(--color-text);
}

@keyframes blink-cursor {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}
</style>
