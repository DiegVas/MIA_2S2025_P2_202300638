import { ref } from "vue";
import { sendCommand } from "../services/send_commands.ts";

const path = ref(["/"]);
const bodyPath = ref(null);

const diskPath = ref(null);
const partitionName = ref(null);

/*
const comman_parse =
  "login -id=" +
  formData.partitionId +
  " -user=" +
  formData.user +
  " -pass=" +
  formData.password;

const commandObj = {
  command: comman_parse,
  timestamp: new Date().toISOString(),
  adjunt_file: null,
};

const system_response = await sendCommand(commandObj);*/

export function useFileStore() {
  function setbodyPath(newBodyPath) {
    let revisedBodyPath = [];
    for (let index = 0; index < newBodyPath.length; index++) {
      if (newBodyPath[index].name == ".") {
        continue;
      }

      revisedBodyPath.push(newBodyPath[index]);
    }

    bodyPath.value = revisedBodyPath;
    localStorage.setItem("bodyPath", JSON.stringify(revisedBodyPath));
    console.log("bodyPath guardado:", bodyPath.value);
  }

  async function setPath(newPath) {
    let fullPath = "";
    let newPathValue = newPath;

    if (path.value.length == 1) {
      fullPath = path.value[0] + newPath;
    } else {
      fullPath = path.value.join("") + "/" + newPath;
      newPathValue = "/" + newPath;
    }

    await fetchDirectoryEntries(fullPath);

    path.value.push(newPathValue);
    localStorage.setItem("path", JSON.stringify(path.value));
    console.log("Path guardado:", path.value);
  }

  function clearPath() {
    path.value = ["/"];
    localStorage.setItem("path", JSON.stringify(path.value));
    console.log("Path limpiado:", path.value);
  }

  async function popPath() {
    let fullPath = "";
    if (path.value.length > 1) {
      path.value.pop();
      localStorage.setItem("path", JSON.stringify(path.value));
    }

    fullPath = path.value.join("");

    await fetchDirectoryEntries(fullPath);

    console.log("Path actualizado:", fullPath);
  }

  async function fetchDirectoryEntries(fullPath) {
    const commandObj = {
      command:
        "ls -path=" +
        diskPath.value +
        " -name=" +
        partitionName.value +
        " -of=" +
        fullPath,
      timestamp: new Date().toISOString(),
      adjunt_file: null,
    };

    const system_response = await sendCommand(commandObj);

    if (system_response.type === "system-error") {
      console.error("Error del sistema:", system_response.message);
      clearSelectedPartition();
      return false;
    }

    setbodyPath(system_response.data.entries);
    return true;
  }

  return {
    path,
    setPath,
    clearPath,
    popPath,

    setbodyPath,

    diskPath,
    partitionName,
    bodyPath,
  };
}
