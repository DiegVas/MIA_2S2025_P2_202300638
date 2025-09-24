<template>
  <!-- Notificación de error mejorada -->
  <Transition name="notification">
    <div v-if="errorMsg" class="notification-container">
      <div class="notification-error">
        <div class="notification-icon">
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        </div>
        <div class="notification-content">
          <div class="notification-title">Error</div>
          <div class="notification-message">{{ errorMsg }}</div>
        </div>
        <button class="notification-close" @click="cerrarError">
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { watch, toRef } from "vue";
const props = defineProps({
  errorMsg: String,
});
const emit = defineEmits(["cerrarError"]);

function cerrarError() {
  emit("cerrarError");
}
</script>

<style scoped>
.notification-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1000;
  max-width: 400px;
}

.notification-error {
  background: linear-gradient(135deg, #ff4757 0%, #ff3838 100%);
  color: white;
  padding: 16px;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(255, 71, 87, 0.3);
  display: flex;
  align-items: flex-start;
  gap: 12px;
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.notification-icon {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  margin-top: 2px;
}

.notification-content {
  flex: 1;
}

.notification-title {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 4px;
}

.notification-message {
  font-size: 13px;
  opacity: 0.95;
  line-height: 1.4;
}

.notification-close {
  background: none;
  border: none;
  color: white;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: background-color 0.2s ease;
  flex-shrink: 0;
}

.notification-close:hover {
  background: rgba(255, 255, 255, 0.2);
}

/* Animaciones para la notificación */
.notification-enter-active {
  transition: all 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.notification-leave-active {
  transition: all 0.3s ease-in;
}

.notification-enter-from {
  transform: translateX(100%) scale(0.8);
  opacity: 0;
}

.notification-leave-to {
  transform: translateX(100%) scale(0.8);
  opacity: 0;
}
</style>
