<template>
  <form @submit.prevent="onSubmit">
    <div
      class="form-group"
      :class="{ 'has-error': fieldErrors.partitionId }"
      :style="{
        transform:
          focusedField === 'partitionId' ? 'translateY(-2px)' : 'translateY(0)',
      }"
    >
      <label for="partition-id">ID de la partición</label>
      <input
        type="text"
        id="partition-id"
        :value="localForm.partitionId"
        @input="updateField('partitionId', $event.target.value)"
        placeholder="Ingresa el ID de la partición"
        @focus="onFocus('partitionId')"
        @blur="onBlur()"
        :disabled="isLoading"
      />
      <div v-if="fieldErrors.partitionId" class="field-error">
        {{ fieldErrors.partitionId }}
      </div>
    </div>

    <div class="form-group" :class="{ 'has-error': fieldErrors.user }">
      <label for="user">Usuario</label>
      <input
        type="user"
        id="user"
        :value="localForm.user"
        @input="updateField('user', $event.target.value)"
        placeholder="Ingresa tu usuario"
        @focus="onFocus('user')"
        @blur="onBlur()"
        :disabled="isLoading"
      />
      <div v-if="fieldErrors.user" class="field-error">
        {{ fieldErrors.user }}
      </div>
    </div>

    <div
      class="form-group"
      :class="{ 'has-error': fieldErrors.password }"
      :style="{
        transform:
          focusedField === 'password' ? 'translateY(-2px)' : 'translateY(0)',
      }"
    >
      <label for="password">Contraseña</label>
      <input
        type="password"
        id="password"
        :value="localForm.password"
        @input="updateField('password', $event.target.value)"
        placeholder="••••••••"
        @focus="onFocus('password')"
        @blur="onBlur()"
        :disabled="isLoading"
      />
      <div v-if="fieldErrors.password" class="field-error">
        {{ fieldErrors.password }}
      </div>
    </div>

    <button type="submit" class="sign-in-button" :disabled="isLoading">
      <span v-if="isLoading" class="loading-spinner"></span>
      {{ isLoading ? "Iniciando sesión..." : "Iniciar Sesión" }}
    </button>
  </form>
</template>

<script setup>
import { ref, watch } from "vue";
const props = defineProps({
  formData: Object,
  fieldErrors: Object,
  isLoading: Boolean,
  focusedField: String,
});
const emit = defineEmits([
  "update:formData",
  "update:focusedField",
  "clearFieldError",
  "submit",
]);

// Copia local para edición
const localForm = ref({ ...props.formData });

watch(
  () => props.formData,
  (val) => {
    localForm.value = { ...val };
  }
);

function updateField(field, value) {
  localForm.value[field] = value;
  emit("update:formData", { ...localForm.value });
}

function onFocus(field) {
  emit("update:focusedField", field);
  emit("clearFieldError", field);
}

function onBlur() {
  emit("update:focusedField", null);
}

function onSubmit() {
  emit("submit");
}
</script>

<style scoped>
.field-error {
  color: #ff4757;
  font-size: 12px;
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.form-group.has-error input {
  border-color: #ff4757;
  background: rgba(255, 71, 87, 0.05);
}

.form-group.has-error input:focus {
  border-color: #ff4757;
  box-shadow: 0 0 0 3px rgba(255, 71, 87, 0.2);
}

/* Spinner de carga */
.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid transparent;
  border-top: 2px solid currentColor;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-right: 8px;
  display: inline-block;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
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
  display: flex;
  align-items: center;
  justify-content: center;
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
</style>
