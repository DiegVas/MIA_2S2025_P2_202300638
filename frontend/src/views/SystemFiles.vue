<script setup>
import { watch } from "vue";
import { useFileStore } from "../store/files.js";

const { bodyPath, path, setPath, popPath } = useFileStore();

const updatePath = async (fileName) => {
  if (fileName === "..") {
    await popPath();
  } else {
    await setPath(fileName);
  }
};
</script>

<template>
  <div>
    <h1>Archivos del Sistema</h1>
    <div class="path">{{ path.join("") }}</div>
    <div class="files-grid">
      <div
        class="file-card"
        v-for="file in bodyPath"
        :key="file.name"
        @click="updatePath(file.name)"
      >
        <p>{{ file.name }}</p>
        <p>{{ file.size }} bytes</p>
        <p>{{ file.date }}</p>
        <p>{{ file.file_type }}</p>
      </div>
    </div>
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

.file-card {
  flex: 1 1 250px;
  max-width: 300px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 1vw;
  padding: 0.5vw;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 0.2vw solid transparent;
  box-shadow: 0 0.3vw 1vw rgba(0, 0, 0, 0.1);
}

.files-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-top: 20px;
}
</style>
