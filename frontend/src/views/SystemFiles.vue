<script setup>
import { usedisksStore } from "../store/disks.js";
</script>

<template>
  <div>
    <h1>Archivos del Sistema</h1>
    <div class="path">/main</div>
    <div class="files-grid"></div>
  </div>
</template>

<style scoped>
.path {
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
}
</style>
