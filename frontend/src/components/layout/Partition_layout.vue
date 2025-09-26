<script setup>
import { usedisksStore } from "../../store/disks.js";
import { useFileStore } from "../../store/files.js";

const { selectedDisk, setSelectedPartition } = usedisksStore();

const { diskPath, partitionName, setbodyPath } = useFileStore();

console.log("Selected Disk in Partition_layout.vue:", selectedDisk);

const initDirectory = async (partition) => {
  diskPath.value = selectedDisk.value.path;
  partitionName.value = partition.name;
  setbodyPath(await setSelectedPartition(partition));
};
</script>

<template>
  <div class="disk-grid">
    <div
      v-for="partition in selectedDisk.partitions"
      :key="partition.name"
      class="disk-card"
      @click="initDirectory(partition)"
    >
      <span
        class="partition-status"
        :class="{ active: partition.status != 'Montada' }"
      >
        {{ partition.status }}
      </span>
      <div class="disk-header">
        <span class="material-symbols-outlined disk-icon"> storage </span>
        <div class="disk-info">
          <h3>{{ partition.name }}</h3>
          <small>{{ partition.size + " MB" }}</small>
        </div>
      </div>
      <div class="disk-details">
        <div class="detail-item">
          <div class="label">Algoritmo Fit</div>
          <div class="value">{{ partition.fit }}</div>
        </div>
        <div class="detail-item">
          <div class="label">Particiones</div>
          <div class="value">
            {{ partition.type }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.partition-status {
  display: inline-block;
  position: absolute;
  left: 40%;
  transform: translateX(-50%);
  top: -35px;
  background-color: var(--color-primary);
  color: var(--color-background);
  border-top-left-radius: 12px;
  border-top-right-radius: 12px;
  border-bottom-left-radius: 6px;
  border-bottom-right-radius: 6px;
  padding: 4px 24px;
  font-weight: bold;
  font-size: 1rem;
  letter-spacing: 1px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  z-index: 2;
}
.active {
  background-color: rgba(var(--color-primary-rgb), 0.5);
}
.disk-card {
  position: relative;
  margin-top: 24px;
}
</style>
