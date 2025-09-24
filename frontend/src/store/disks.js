import { ref } from "vue";

const disks = ref(
  /*localStorage.getItem("disks")
    ? JSON.parse(localStorage.getItem("disks"))
    :*/
  null
);

export function usedisksStore() {
  function setdisks(u) {
    disks.value = u;
    localStorage.setItem("disks", JSON.stringify(u));
    console.log("discos guardado:", disks.value);
  }
  function cleardisks() {
    disks.value = null;
    localStorage.removeItem("disks");
  }

  return { disks, setdisks, cleardisks };
}
