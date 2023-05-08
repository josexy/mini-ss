<script setup lang="ts">
import { useConfigStore } from "@/stores/modules/config";
const configStore = useConfigStore();
</script>

<template>
  <div class="container">
    <n-alert type="info">
      当前匹配规则：<strong>{{ configStore.rule.mode }}</strong>
    </n-alert>
    <div class="table-wrapper">
      <n-table :bordered="true" :striped="true">
        <thead>
          <tr>
            <th>类型</th>
            <th>目标</th>
            <th>代理</th>
            <th>行为</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(item, index) in configStore.rule.rules" :key="index">
            <td>{{ item.type }}</td>
            <td>{{ item.target }}</td>
            <td>{{ item.proxy }}</td>
            <td>{{ item.action }}</td>
          </tr>
        </tbody>
      </n-table>
    </div>
  </div>
</template>

<style scoped>
.container {
  height: 90vh;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.table-wrapper {
  border-radius: 5px;
  border: 1px solid #c0c0c0;
  flex: 1;
  overflow-y: auto;
}
</style>
