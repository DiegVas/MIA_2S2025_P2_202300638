<template>
  <div class="login-wrapper">
    <div class="login-container">
      <Back_button />
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        fill="currentColor"
        class="bi bi-dropbox logo"
        viewBox="0 0 16 16"
      >
        <path
          d="M8.01 4.555 4.005 7.11 8.01 9.665 4.005 12.22 0 9.651l4.005-2.555L0 4.555 4.005 2zm-4.026 8.487 4.006-2.555 4.005 2.555-4.005 2.555zm4.026-3.39 4.005-2.556L8.01 4.555 11.995 2 16 4.555 11.995 7.11 16 9.665l-4.005 2.555z"
        />
      </svg>

      <div class="welcome-text">
        <h1>¡Hola, bienvenido!</h1>
      </div>

      <Login_error :errorMsg="errorMsg" @cerrarError="cerrarError" />

      <login_form
        :formData="formData"
        :fieldErrors="fieldErrors"
        :isLoading="isLoading"
        :focusedField="focusedField"
        @update:formData="updateFormData"
        @update:focusedField="(value) => (focusedField = value)"
        @clearFieldError="clearFieldError"
        @submit="handleSubmit"
      />

      <div class="terms">
        Al continuar, reconoces que has leído y aceptas nuestros<br />
        <a href="#" @click="handleTerms">Términos de Servicio</a> y nuestra
        <a href="#" @click="handlePrivacy">Política de Privacidad</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from "vue";
import { sendCommand } from "../services/send_commands.ts";
import Login_error from "../components/common/Login_error.vue";
import Login_form from "../components/common/Login_form.vue";
import { useUserStore } from "../store/user.js";
import { useRouter } from "vue-router";
import Back_button from "../components/layout/Back_button.vue";

const router = useRouter();

const { setUser } = useUserStore();

// Estado reactivo
const formData = reactive({
  partitionId: "",
  user: "",
  password: "",
});

const focusedField = ref(null);
const isLoading = ref(false);

// Estado para notificación de error
const errorMsg = ref("");
const errorTimer = ref(null);

// Errores específicos por campo
const fieldErrors = reactive({
  partitionId: "",
  user: "",
  password: "",
});

// Emits
const emit = defineEmits(["login-success", "error", "go-back"]);

// Métodos
const handleSubmit = async () => {
  console.log("Intentando iniciar sesión con:", formData);

  // Limpiar errores previos
  clearAllErrors();

  // Validaciones individuales
  let hasErrors = false;

  if (!formData.partitionId.trim()) {
    fieldErrors.partitionId = "El ID de la partición es requerido";
    hasErrors = true;
  }

  if (!formData.user.trim()) {
    fieldErrors.user = "El usuario es requerido";
    hasErrors = true;
  }

  if (!formData.password) {
    fieldErrors.password = "La contraseña es requerida";
    hasErrors = true;
  }

  if (hasErrors) {
    mostrarError("Por favor, completa todos los campos correctamente");
    return;
  }

  isLoading.value = true;
  console.log("isLoading establecido a:", isLoading.value);

  try {
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

    const system_response = await sendCommand(commandObj);

    console.log("Login exitoso:", system_response);

    if (
      system_response.type === "system-error" ||
      system_response.type === "auth-error"
    ) {
      mostrarError(
        system_response.text || "Error desconocido al iniciar sesión"
      );
      return;
    }

    // Emitir evento de login exitoso
    emit("login-success", {
      user: formData.user,
      partitionId: formData.partitionId,
    });

    setUser({
      user: formData.user,
      partitionId: formData.partitionId,
    });
    router.replace("/home");
  } catch (error) {
    console.error("Error al iniciar sesión:", error);
    mostrarError("Error de conexión. Por favor, intenta nuevamente.");
    return;
  } finally {
    isLoading.value = false;
    console.log("isLoading establecido a:", isLoading.value);
  }
};

// Función para mostrar notificación de error
function mostrarError(msg) {
  // Limpiar timer anterior si existe
  if (errorTimer.value) {
    clearTimeout(errorTimer.value);
  }

  errorMsg.value = msg;
  errorTimer.value = setTimeout(() => {
    errorMsg.value = "";
    errorTimer.value = null;
  }, 5000);
}

// Función para cerrar error manualmente
function cerrarError() {
  if (errorTimer.value) {
    clearTimeout(errorTimer.value);
    errorTimer.value = null;
  }
  errorMsg.value = "";
}

// Limpiar error de campo específico
function clearFieldError(field) {
  fieldErrors[field] = "";
}

// Limpiar todos los errores
function clearAllErrors() {
  Object.keys(fieldErrors).forEach((key) => {
    fieldErrors[key] = "";
  });
}

const goBack = () => {
  emit("go-back");
};

function updateFormData(newData) {
  Object.assign(formData, newData);
}
</script>

<style scoped>
/* Estilos base mantenidos */
.login-wrapper {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen,
    Ubuntu, Cantarell, sans-serif;
  background: rgba(255, 255, 255, 0.03);
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  position: relative;
}

.login-container {
  padding: 48px;
  width: 100%;
  max-width: 420px;
}

.logo {
  width: 60px;
  height: 60px;
  display: flex;
  margin: 0 auto 16px auto;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.welcome-text {
  text-align: center;
  margin-bottom: 8px;
}

.welcome-text h1 {
  font-size: 32px;
  font-weight: 700;
  margin-bottom: 16px;
}

.terms {
  text-align: center;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  line-height: 1.5;
}

.terms a {
  color: rgba(255, 255, 255, 0.6);
  text-decoration: none;
}

.terms a:hover {
  color: rgba(255, 255, 255, 0.8);
}

@media (max-width: 480px) {
  .login-container {
    padding: 32px 24px;
    margin: 20px;
    border-radius: 16px;
  }

  .welcome-text h1 {
    font-size: 28px;
  }

  .notification-container {
    top: 10px;
    right: 10px;
    left: 10px;
    max-width: none;
  }
}
</style>
