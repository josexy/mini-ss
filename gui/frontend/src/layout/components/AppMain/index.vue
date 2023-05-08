<script setup lang="ts"></script>

<template>
  <div class="main-contanier">
    <RouterView v-slot="{ Component }">
      <keep-alive>
        <component :is="Component" />
      </keep-alive>
    </RouterView>
  </div>
</template>

<style scoped>
.main-contanier {
  margin-left: 2%;
  margin-right: 2%;
  padding-top: 2rem;
}
</style>
