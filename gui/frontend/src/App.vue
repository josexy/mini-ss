<script lang="ts" setup>
import { useThemeStore } from "@/stores/modules/theme";
</script>

<template>
  <div style="--wails-draggable: drag">
    <n-config-provider :theme="useThemeStore().theme">
      <RouterView />
    </n-config-provider>
  </div>
</template>
