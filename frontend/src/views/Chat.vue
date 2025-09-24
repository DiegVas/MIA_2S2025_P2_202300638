<script setup>
import Header_Layout from "../components/layout/Header_Layout.vue";
import Body_Layout from "../components/layout/Body_Layout.vue";
</script>

<template>
  <div class="w-full h-full">
    <Header_Layout />
    <Body_Layout />
  </div>
</template>

<style scoped>
.header {
  height: 10%;
  padding: 2rem;
}

.body {
  padding: 2rem;
  height: 90%;
}
</style>
