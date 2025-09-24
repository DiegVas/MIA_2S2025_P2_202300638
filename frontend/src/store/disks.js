import { ref } from "vue";

const disks = ref(null);
const selectedDisk = ref(null);

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

  return {
    disks,
    setdisks,
    cleardisks,
    selectedDisk,
    setSelectedDisk,
    clearSelectedDisk,
  };
}
