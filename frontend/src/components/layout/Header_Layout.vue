<script setup>
import { useUserStore } from "../../store/user.js";
const { user } = useUserStore();
</script>

<template>
  <div class="header w-full justify-between items-center flex">
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
