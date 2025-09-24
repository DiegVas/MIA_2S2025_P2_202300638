import { ref } from "vue";
import { sendCommand } from "../services/send_commands.ts";

const user = ref(
 /* localStorage.getItem("user")
    ? JSON.parse(localStorage.getItem("user"))
    :*/ 
    null
);

console.log("Usuario cargado:", user.value);

export function useUserStore() {
  function setUser(u) {
    user.value = u;
    localStorage.setItem("user", JSON.stringify(u));
    console.log("Usuario guardado:", user.value);
  }
  function clearUser() {
    user.value = null;
    localStorage.removeItem("user");
  }

  async function validateSession() {
    const token = localStorage.getItem("token");
    if (!token) {
      clearUser();
      return false;
    }
    try {
          const commandObj = {
      command: "sesion",
      timestamp: new Date().toISOString(),
      adjunt_file: null,
    };

    const res = await sendCommand(commandObj);

      if (res.ok) {
        const data = await res.json();
        setUser(data.user);
        return true;
      } else {
        clearUser();
        return false;
      }
    } catch {
      clearUser();
      return false;
    }
  }

  return { user, setUser, clearUser, validateSession };
}