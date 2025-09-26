<script setup>
import { useFileStore } from "../../store/files.js";
const { fileContent, clearFileContent } = useFileStore();

// Solo visual, no funcional
function goBack() {
  // Aquí podrías emitir un evento si luego quieres funcionalidad
}
</script>

<template>
  <div class="file-content-container">
    <div class="file-content-header">
      <h2>Contenido del archivo</h2>
      <button class="close-btn" @click="clearFileContent">
        <span class="material-symbols-outlined">close</span>
        Cerrar
      </button>
    </div>
    <div class="file-content-body">
      <pre>{{ fileContent }}</pre>
    </div>
  </div>
</template>

<style scoped>
.file-content-container {
  background: var(--color-primary);
  color: var(--color-background);
  border-radius: 24px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.12);
  padding: 24px 32px;
  max-width: 800px;
  margin: 40px auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.file-content-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.file-content-header h2 {
  flex: 1;
  font-size: 1.5rem;
  font-weight: bold;
  margin: 0;
}

.back-btn,
.close-btn {
  background: var(--color-secundary);
  color: var(--color-background);
  border: none;
  border-radius: 16px;
  padding: 8px 18px;
  font-size: 1rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: background 0.2s;
}

.back-btn:hover,
.close-btn:hover {
  background: var(--color-secundary-dark, #2c3e50);
}

.file-content-body {
  background: var(--color-background);
  color: var(--color-text, #222);
  border-radius: 12px;
  padding: 18px;
  font-family: "Fira Mono", "Consolas", monospace;
  font-size: 1.1rem;
  overflow-x: auto;
  min-height: 200px;
  max-height: 400px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}
</style>
