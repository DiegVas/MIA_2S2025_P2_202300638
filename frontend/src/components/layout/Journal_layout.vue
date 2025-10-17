<template>
  <div class="journal-container">
    <div class="journal-header">
      <h2>Journal de Operaciones</h2>
      <div class="journal-info">
        <span class="partition-info">Partición ID: {{ partitionId }}</span>
        <span class="entries-count">{{ journalData.length }} entradas</span>
      </div>
      <div class="header-actions">
        <button
          class="refresh-btn"
          @click="loadJournalData"
          :disabled="isLoading"
        >
          <span
            class="material-symbols-outlined"
            :class="{ spinning: isLoading }"
            >refresh</span
          >
          Actualizar
        </button>
        <button class="close-btn" @click="closeJournal">
          <span class="material-symbols-outlined">close</span>
          Cerrar
        </button>
      </div>
    </div>

    <div class="journal-body" v-if="!isLoading && journalData.length > 0">
      <div class="journal-list">
        <TransitionGroup name="journal-fade" tag="div">
          <div
            v-for="(entry, index) in journalData"
            :key="index"
            class="journal-entry"
            :class="getStatusClass(entry.status)"
          >
            <div class="entry-header">
              <div class="entry-operation">
                <span class="operation-icon material-symbols-outlined">
                  {{ getOperationIcon(entry.operation) }}
                </span>
                <span class="operation-text">{{ entry.operation }}</span>
              </div>
              <div class="entry-status">
                <span class="status-badge" :class="entry.status.toLowerCase()">
                  {{ entry.status }}
                </span>
              </div>
            </div>

            <div class="entry-details">
              <div class="entry-path">
                <span class="material-symbols-outlined">folder_open</span>
                <span class="path-text">{{ entry.path }}</span>
              </div>

              <div class="entry-content" v-if="entry.content">
                <span class="material-symbols-outlined">description</span>
                <span class="content-text">{{ entry.content }}</span>
              </div>

              <div class="entry-meta">
                <div class="entry-partition">
                  <span class="material-symbols-outlined">storage</span>
                  <span>{{ entry.partition_id }}</span>
                </div>
                <div class="entry-timestamp">
                  <span class="material-symbols-outlined">schedule</span>
                  <span>{{ formatTimestamp(entry.timestamp) }}</span>
                </div>
              </div>
            </div>
          </div>
        </TransitionGroup>
      </div>
    </div>

    <div class="journal-loading" v-else-if="isLoading">
      <span class="material-symbols-outlined loading-icon">refresh</span>
      <h3>Cargando journal...</h3>
      <p>Obteniendo entradas del sistema</p>
    </div>

    <div class="journal-empty" v-else>
      <span class="material-symbols-outlined empty-icon">history</span>
      <h3>No hay entradas en el journal</h3>
      <p>Las operaciones realizadas aparecerán aquí</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import { useFileStore } from "../../store/files.js";

const { callJournal } = useFileStore();

const journalData = ref([]);
const partitionId = ref("");
const isLoading = ref(false);

// Definir el evento para cerrar el journal
const emit = defineEmits(["close-journal"]);

const closeJournal = () => {
  emit("close-journal");
};

const formatTimestamp = (timestamp) => {
  const date = new Date(timestamp);
  return date.toLocaleString("es-ES", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

const getOperationIcon = (operation) => {
  const icons = {
    CREATE_FILE: "add_circle",
    DELETE_FILE: "delete",
    MODIFY_FILE: "edit",
    CREATE_DIRECTORY: "create_new_folder",
    DELETE_DIRECTORY: "folder_delete",
    MOVE: "drive_file_move",
    COPY: "content_copy",
    RENAME: "drive_file_rename_outline",
    CHANGE_PERMISSIONS: "security",
    LOGIN: "login",
    LOGOUT: "logout",
    DEFAULT: "description",
  };
  return icons[operation] || icons["DEFAULT"];
};

const getStatusClass = (status) => {
  return {
    "status-committed": status === "COMMITTED",
    "status-pending": status === "PENDING",
    "status-aborted": status === "ABORTED",
  };
};

const loadJournalData = async () => {
  isLoading.value = true;
  try {
    const result = await callJournal();
    if (result && result.success) {
      // Los datos vienen en result.data que contiene el objeto JournalingInfo del backend
      const journalingInfo = result.data;
      journalData.value = journalingInfo.entries || [];
      partitionId.value =
        journalingInfo.partition_id || result.partitionId || "N/A";
    } else {
      console.error("Error loading journal:", result?.error);
      journalData.value = [];
      partitionId.value = "Error";
    }
  } catch (error) {
    console.error("Error loading journal data:", error);
    journalData.value = [];
    partitionId.value = "Error";
  } finally {
    isLoading.value = false;
  }
};

onMounted(() => {
  loadJournalData();
});
</script>

<style scoped>
.journal-container {
  background: var(--color-primary);
  color: var(--color-background);
  border-radius: 24px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.12);
  padding: 24px 32px;
  max-width: 1000px;
  margin: 40px auto;
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-height: 80vh;
}

.journal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.journal-header h2 {
  font-size: 1.8rem;
  font-weight: bold;
  margin: 0;
}

.journal-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.9rem;
  opacity: 0.8;
}

.partition-info {
  font-weight: 600;
}

.entries-count {
  font-style: italic;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.refresh-btn,
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

.refresh-btn:hover,
.close-btn:hover {
  background: var(--color-secundary-dark, #2c3e50);
}

.refresh-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.refresh-btn .material-symbols-outlined.spinning {
  animation: spin 1s linear infinite;
}

.journal-body {
  overflow-y: auto;
  flex: 1;
}

.journal-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.journal-entry {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  padding: 20px;
  border-left: 4px solid var(--color-secundary);
  transition: all 0.3s ease;
}

.journal-entry:hover {
  background: rgba(255, 255, 255, 0.15);
  transform: translateX(4px);
}

.journal-entry.status-committed {
  border-left-color: #27ae60;
}

.journal-entry.status-pending {
  border-left-color: #f39c12;
}

.journal-entry.status-aborted {
  border-left-color: #e74c3c;
}

.entry-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.entry-operation {
  display: flex;
  align-items: center;
  gap: 8px;
}

.operation-icon {
  font-size: 24px;
  color: var(--color-secundary);
}

.operation-text {
  font-size: 1.2rem;
  font-weight: 600;
}

.entry-status .status-badge {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
}

.status-badge.committed {
  background: #27ae60;
  color: white;
}

.status-badge.pending {
  background: #f39c12;
  color: white;
}

.status-badge.aborted {
  background: #e74c3c;
  color: white;
}

.entry-details {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.entry-path,
.entry-content {
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: "Fira Mono", "Consolas", monospace;
}

.entry-path .material-symbols-outlined,
.entry-content .material-symbols-outlined {
  font-size: 18px;
  opacity: 0.7;
}

.path-text {
  background: rgba(0, 0, 0, 0.2);
  padding: 4px 8px;
  border-radius: 8px;
  font-size: 0.9rem;
}

.content-text {
  font-size: 0.9rem;
  opacity: 0.8;
  font-style: italic;
}

.entry-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
  font-size: 0.8rem;
  opacity: 0.7;
}

.entry-partition,
.entry-timestamp {
  display: flex;
  align-items: center;
  gap: 4px;
}

.entry-partition .material-symbols-outlined,
.entry-timestamp .material-symbols-outlined {
  font-size: 16px;
}

.journal-empty,
.journal-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 60px 20px;
  text-align: center;
  opacity: 0.7;
}

.empty-icon,
.loading-icon {
  font-size: 64px;
  opacity: 0.5;
}

.loading-icon {
  animation: spin 2s linear infinite;
}

.journal-empty h3,
.journal-loading h3 {
  margin: 0;
  font-size: 1.5rem;
}

.journal-empty p,
.journal-loading p {
  margin: 0;
  font-size: 1rem;
  opacity: 0.8;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Animaciones */
.journal-fade-enter-active,
.journal-fade-leave-active {
  transition: all 0.3s ease;
}

.journal-fade-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.journal-fade-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

/* Responsive */
@media (max-width: 768px) {
  .journal-container {
    margin: 20px auto;
    padding: 16px 20px;
    max-height: 90vh;
  }

  .journal-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .header-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .journal-header h2 {
    font-size: 1.5rem;
  }

  .journal-entry {
    padding: 16px;
  }

  .entry-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .entry-meta {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>
