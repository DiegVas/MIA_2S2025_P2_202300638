import { ref } from "vue";
import { sendCommand } from "../services/send_commands.ts";
import { useFileStore } from "./files.js";

const disks = ref(null);
const selectedDisk = ref(null);
const selectedPartition = ref(null);

export function usedisksStore() {
  function setdisks(u) {
    disks.value = u;
    localStorage.setItem("disks", JSON.stringify(u));
    console.log("discos guardado:", disks.value);
  }
  function cleardisks() {
    disks.value = null;
    localStorage.removeItem("disks");
    console.log("discos limpiado:", disks.value);
  }

  function setSelectedDisk(disk) {
    localStorage.setItem("selectedDisk", JSON.stringify(disk));
    selectedDisk.value = disk;
    console.log("Disco seleccionado:", selectedDisk.value);
  }

  function clearSelectedDisk() {
    selectedDisk.value = null;
    localStorage.removeItem("selectedDisk");
    console.log("Disco seleccionado limpiado:", selectedDisk.value);
  }

  async function setSelectedPartition(partition) {
    localStorage.setItem("selectedPartition", JSON.stringify(partition));
    selectedPartition.value = partition;
    console.log("Partición seleccionada:", selectedPartition.value);

    const commandObj = {
      command:
        "ls -path=" +
        selectedDisk.value.path +
        " -name=" +
        selectedPartition.value.name +
        " -of=/",
      timestamp: new Date().toISOString(),
      adjunt_file: null,
    };
    const system_response = await sendCommand(commandObj);

    if (system_response.type == "system-error") {
      console.error("Error del sistema:", system_response.message);
      clearSelectedPartition();
      return null;
    }

    return system_response.data.entries;
  }

  function clearSelectedPartition() {
    selectedPartition.value = null;
    localStorage.removeItem("selectedPartition");
    console.log("Partición seleccionada limpiada:", selectedPartition.value);
  }

  return {
    disks,
    setdisks,
    cleardisks,
    selectedDisk,
    setSelectedDisk,
    clearSelectedDisk,
    selectedPartition,
    setSelectedPartition,
    clearSelectedPartition,
  };
}
