<script setup>
import { usedisksStore } from "../../store/disks.js";
import { sendCommand } from "../../services/send_commands.js";

const { disks, setSelectedDisk, selectedDisk, clearSelectedDisk } =
  usedisksStore();

const disks_list = disks.value.data ? disks.value.data : [];

async function setDisk(disk) {
  const command = `partdisk -path=${disk.path}`;

  const commandObj = {
    command: command,
    timestamp: new Date().toISOString(),
    adjunt_file: null,
  };

  const system_response = await sendCommand(commandObj);
  disk.partitions = system_response.data ? system_response.data.partitions : [];

  setSelectedDisk(disk);
}
</script>

<template>
  <div class="disk-grid">
    <div
      v-for="disk in disks_list"
      :key="disk.id"
      class="disk-card"
      @click="setDisk(disk)"
    >
      <div class="disk-header">
        <span class="material-symbols-outlined disk-icon"> hard_drive_2 </span>
        <div class="disk-info">
          <h3>{{ disk.name }}</h3>
          <small>{{ disk.size + " MB" }}</small>
        </div>
      </div>

      <div class="disk-details">
        <div class="detail-item">
          <div class="label">Algoritmo Fit</div>
          <div class="value">{{ disk.fit }}</div>
        </div>
        <div class="detail-item">
          <div class="label">Particiones</div>
          <div class="value">
            {{
              disk.partitions == null
                ? "No hay particiones"
                : disk.partitions.length
            }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style>
.disk-icon {
  font-size: 50px;
  color: var(--color-secundary);
  margin-right: 10px;
}

.disk-grid {
  display: flex;
  flex-wrap: wrap;
  margin-top: 2vw;
  gap: 2vw;
  justify-content: start;
}

.disk-card {
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

.disk-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
}

.disk-header {
  display: flex;
  align-items: center;
  margin-bottom: 15px;
}

.disk-info h3 {
  margin: 0px;
  font-size: 25px;
}

.disk-info small {
  font-size: 14px;
  color: #666;
}

.disk-details {
  display: grid;
  grid-template-columns: 48% 48%;
  gap: 2%;
  margin-top: 1vw;
}

.detail-item {
  background: rgba(255, 255, 255, 0.8);
  padding: 10px;
  border-radius: 8px;
  text-align: center;
}

.detail-item .label {
  font-size: 0.8rem;
  color: #666;
  margin-bottom: 5px;
}

.detail-item .value {
  font-weight: bold;
  color: #333;
}
</style>
