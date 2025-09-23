<template>
  <div class="login-wrapper">
    <button class="back-button" @click="$router.push('/')">
      <svg
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <path d="M19 12H5M12 19l-7-7 7-7" />
      </svg>
      Atrás
    </button>

    <div class="login-container">
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

      <form @submit.prevent="handleSubmit">
        <div
          class="form-group"
          :style="{
            transform:
              focusedField === 'userId' ? 'translateY(-2px)' : 'translateY(0)',
          }"
        >
          <label for="user-id">ID de Usuario</label>
          <input
            type="text"
            id="user-id"
            v-model="formData.userId"
            placeholder="Ingresa tu ID"
            @focus="focusedField = 'userId'"
            @blur="focusedField = null"
            :disabled="isLoading"
          />
        </div>

        <div
          class="form-group"
          :style="{
            transform:
              focusedField === 'email' ? 'translateY(-2px)' : 'translateY(0)',
          }"
        >
          <label for="email">Tu correo electrónico</label>
          <input
            type="email"
            id="email"
            v-model="formData.email"
            placeholder="nombre@ejemplo.com"
            @focus="focusedField = 'email'"
            @blur="focusedField = null"
            :disabled="isLoading"
          />
        </div>

        <div
          class="form-group"
          :style="{
            transform:
              focusedField === 'password'
                ? 'translateY(-2px)'
                : 'translateY(0)',
          }"
        >
          <label for="password">Contraseña</label>
          <input
            type="password"
            id="password"
            v-model="formData.password"
            placeholder="••••••••"
            @focus="focusedField = 'password'"
            @blur="focusedField = null"
            :disabled="isLoading"
          />
        </div>

        <button type="submit" class="sign-in-button" :disabled="isLoading">
          {{ isLoading ? "Iniciando sesión..." : "Iniciar Sesión" }}
        </button>
      </form>

      <div class="terms">
        Al continuar, reconoces que has leído y aceptas nuestros<br />
        <a href="#" @click="handleTerms">Términos de Servicio</a> y nuestra
        <a href="#" @click="handlePrivacy">Política de Privacidad</a>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "LoginInterface",
  data() {
    return {
      formData: {
        userId: "",
        email: "",
        password: "",
      },
      focusedField: null,
      isLoading: false,
    };
  },
  methods: {
    async handleSubmit() {
      if (
        !this.formData.userId ||
        !this.formData.email ||
        !this.formData.password
      ) {
        this.$emit("error", "Por favor, completa todos los campos");
        return;
      }

      this.isLoading = true;

      try {
        // Simulación de proceso de login
        await new Promise((resolve) => setTimeout(resolve, 2000));

        // Emitir evento de login exitoso
        this.$emit("login-success", {
          userId: this.formData.userId,
          email: this.formData.email,
        });
      } catch (error) {
        this.$emit("error", "Error al iniciar sesión");
      } finally {
        this.isLoading = false;
      }
    },

    goBack() {
      this.$emit("go-back");
    },

    handleSignup() {
      this.$emit("go-to-signup");
    },

    handleTerms() {
      this.$emit("show-terms");
    },

    handlePrivacy() {
      this.$emit("show-privacy");
    },
  },
  emits: [
    "login-success",
    "error",
    "go-back",
    "go-to-signup",
    "show-terms",
    "show-privacy",
  ],
};
</script>

<style scoped>
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

.back-button {
  position: absolute;
  top: 24px;
  left: 24px;
  background: none;
  border: none;
  color: rgba(255, 255, 255, 0.7);
  font-size: 16px;
  cursor: pointer;
  padding: 8px;
  border-radius: 8px;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 8px;
}

.back-button:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
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

.form-group {
  margin-bottom: 24px;
  transition: transform 0.3s ease;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: rgba(255, 255, 255, 0.8);
  font-size: 14px;
  font-weight: 500;
}

.form-group input {
  width: 100%;
  padding: 16px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  color: white;
  font-size: 16px;
  transition: all 0.3s ease;
  outline: none;
}

.form-group input:focus {
  border-color: rgba(195, 221, 73, 1);
  background: rgba(255, 255, 255, 0.08);
  box-shadow: 0 0 0 3px rgba(195, 221, 73, 0.4);
}

.form-group input::placeholder {
  color: rgba(255, 255, 255, 0.4);
}

.form-group input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.sign-in-button {
  width: 100%;
  padding: 18px;
  background: var(--color-primary);
  border: none;
  border-radius: 12px;
  color: var(--color-background);
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  margin-bottom: 24px;
  box-shadow: 0 8px 16px rgba(195, 221, 73, 0.2);
}

.sign-in-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 12px 24px rgba(195, 221, 73, 0.3);
}

.sign-in-button:active:not(:disabled) {
  transform: translateY(0);
}

.sign-in-button:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none;
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
}
</style>
