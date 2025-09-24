<script setup>
const props = defineProps({
  route: {
    type: String,
    default: "/",
  },
});
</script>

<template>
  <button class="back-button" @click="$router.push(props.route)">
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
</template>

<style scoped>
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
</style>
