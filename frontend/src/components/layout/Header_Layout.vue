<script setup>
import { useUserStore } from "../../store/user.js";
import { sendCommand } from "../../services/send_commands.ts";
import { useRouter } from "vue-router";

const router = useRouter();
const { user, setUser } = useUserStore();

const handleLogout = async () => {
  try {
    const comman_parse = "logout";

    const commandObj = {
      command: comman_parse,
      timestamp: new Date().toISOString(),
      adjunt_file: null,
    };

    const system_response = await sendCommand(commandObj);

    console.log("logout exitoso:", system_response);

    if (
      system_response.type === "system-error" ||
      system_response.type === "auth-error"
    ) {
      console.error(
        system_response.text || "Error desconocido al cerrar sesión"
      );
      return;
    }

    setUser(null);
    router.replace("/home");
  } catch (error) {
    console.error("Error al cerrar sesión:", error);
    return;
  } finally {
  }
};
</script>

<template>
  <div class="header w-full justify-between items-center flex">
    <button
      v-if="user != null"
      @click="handleLogout"
      class="login_btn"
      title="Página principal"
    >
      Cerrar sesión
    </button>
    <h1 class="text_header font-bold">ExtreamFS</h1>

    <button
      v-if="user != null"
      @click="$router.push('/home')"
      class="login_btn"
      title="Visualizar archivos"
    >
      Visualizar archivos
    </button>
    <button
      v-if="user == null"
      @click="$router.push('/login')"
      class="login_btn"
      title="Iniciar sesión"
    >
      Iniciar sesión
    </button>
  </div>
</template>

<style scoped>
.text_header {
  font-size: 2rem;
}

.login_btn {
  background-color: var(--color-primary);
  color: var(--color-background);
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  transition: all var(--transition-fast);
}

.login_btn:hover {
  background: #b8d142;
  transform: translateY(-2px);
}
</style>
