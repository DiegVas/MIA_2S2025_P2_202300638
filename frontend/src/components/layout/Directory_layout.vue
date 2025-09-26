<script setup>
import { useFileStore } from "../../store/files.js";
import { ref, watch } from "vue";
import { formatearFecha } from "../../services/format_date.js";

const {
  bodyPath,
  path,
  setPath,
  setPathFile,
  popPath,
  fetchFileContent,
  fileContent,
} = useFileStore();

const updatePath = async (fileName, fileType) => {
  if (fileName === "..") {
    await popPath();
  } else if (fileType === "file") {
    await fetchFileContent(fileName);
    await setPathFile(fileName);
  } else {
    await setPath(fileName);
  }
};

const bodyContent = ref([]);

const updateContent = (newContent) => {
  if (bodyContent.value.length === 0) {
    // Animación de inserción elemento por elemento
    let i = 0;
    bodyContent.value = [];
    const insertInterval = setInterval(() => {
      if (i < newContent.length) {
        bodyContent.value.push(newContent[i]);
        i++;
      } else {
        clearInterval(insertInterval);
      }
    }, 80); // velocidad de inserción
  } else {
    // Animación de borrado elemento por elemento

    const interval = setInterval(() => {
      if (bodyContent.value.length > 0) {
        bodyContent.value.pop();
      } else {
        clearInterval(interval);
        // Después de borrar, animar la inserción
        let i = 0;
        const insertInterval = setInterval(() => {
          if (i < newContent.length) {
            bodyContent.value.push(newContent[i]);
            i++;
          } else {
            clearInterval(insertInterval);
          }
        }, 100); // velocidad de inserción
      }
    }, 100); // velocidad de borrado
  }
};

watch(
  () => bodyPath.value,
  (newVal) => {
    updateContent(newVal);
  },
  { immediate: true }
);
</script>

<template>
  <TransitionGroup
    name="file-list"
    tag="div"
    class="files-grid"
    v-if="!fileContent"
    :key="bodyPath"
  >
    <div
      class="file-card"
      v-for="file in bodyContent"
      :key="file.name"
      @click="updatePath(file.name, file.file_type)"
    >
      <div class="icon-container">
        <span class="material-symbols-outlined icon normal-icon">
          {{ file.file_type == "directory" ? "folder" : "news" }}
        </span>
        <span class="material-symbols-outlined icon hover-icon">
          {{ file.file_type == "directory" ? "folder_open" : "visibility" }}
        </span>
      </div>

      <p class="file-name">{{ file.name }}</p>
      <div class="file-info">
        <p>{{ file.size }} bytes</p>
        <p>{{ formatearFecha(file.date) }}</p>
      </div>
    </div>
  </TransitionGroup>
</template>

<style scoped>
.file-info {
  font-size: 0.8rem;
  color: gray;
  margin-top: 8px;
  display: flex;
  justify-content: space-between;
  width: 100%;
}

.file-card {
  flex: 1 1 250px;
  max-width: 300px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 1vw;
  padding: 1vw;
  display: flex;
  flex-direction: column;
  align-items: center;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 0.2vw solid transparent;
  box-shadow: 0 0.3vw 1vw rgba(0, 0, 0, 0.1);
}

.file-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
}

.files-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 250px));
  gap: 16px;
  margin-top: 20px;
}

/* Contenedor para superponer los íconos */
.icon-container {
  position: relative;
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Estilos base para los íconos */
.icon {
  font-size: 80px;
  color: var(--color-secundary);
  position: absolute;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  top: 50%;
  left: 50%;
  transform-origin: center;
}

.normal-icon {
  opacity: 1;
  transform: translate(-50%, -50%) scale(1) rotate(0deg);
}

.hover-icon {
  opacity: 0;
  transform: translate(-50%, -50%) scale(0.8) rotate(-10deg);
}

/* Transiciones en hover */
.file-card:hover .normal-icon {
  opacity: 0;
  transform: translate(-50%, -50%) scale(0.8) rotate(10deg);
}

.file-card:hover .hover-icon {
  opacity: 1;
  transform: translate(-50%, -50%) scale(1.1) rotate(0deg);
}
/* Animaciones de entrada y salida */
.file-list-enter-from {
  opacity: 0;
  transform: translateY(30px) scale(0.8);
  filter: blur(80px);
}
.file-list-leave-to {
  opacity: 0;
  transform: translateY(-20px) scale(0.9);
  filter: blur(2px);
}
.file-list-move {
  transition: transform 0.5s cubic-bezier(0.25, 0.46, 0.45, 0.94);
}
</style>
