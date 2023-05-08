<script setup lang="ts">
import Sidebar from "@/layout/components/Sidebar/index.vue";
import AppMain from "@/layout/components/AppMain/index.vue";
</script>

<template>
  <n-space vertical size="large">
    <n-layout has-sider class="container">
      <n-layout-sider class="sidebar">
        <Sidebar />
      </n-layout-sider>
      <n-layout-content class="app-main">
        <n-message-provider>
          <AppMain />
        </n-message-provider>
      </n-layout-content>
    </n-layout>
  </n-space>
</template>

<style scoped lang="scss">
.container {
  display: flex;
  height: 100vh;
}

.sidebar {
  width: 200px;
  min-height: 400px;
  align-items: center;
  align-content: center;
  padding-top: 2rem;
}

.app-main {
  flex: 1;
  overflow-y: auto;
}
</style>
