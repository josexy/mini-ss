<script setup lang="ts">
import { ref } from "vue";
import { useConfigStore } from "@/stores/modules/config";
import { useThemeStore } from "@/stores/modules/theme";
import { Brightness4Sharp } from "@vicons/material";
import AceEditor from "@/components/AceEditor/index.vue";

const configStore = useConfigStore();
const showModel = ref(false);

const configJson = ref("");
const show = () => {
  if (configStore.config) {
    configJson.value = JSON.stringify(configStore.config, null, 2);
  }
  showModel.value = true;
};
</script>

<template>
  <div>
    <n-grid :cols="1">
      <n-grid-item>
        <n-button strong secondary circle @click="useThemeStore().toggle()">
          <template #icon>
            <n-icon>
              <Brightness4Sharp />
            </n-icon>
          </template>
        </n-button>
      </n-grid-item>
    </n-grid>
    <n-grid class="el-row-margin" :cols="1">
      <n-grid-item>
        <n-input v-model:value="configStore.path" placeholder="配置文件路径" />
      </n-grid-item>
    </n-grid>
    <n-grid class="el-row-margin" :cols="1">
      <n-grid-item>
        <n-button
          secondary
          strong
          type="info"
          style="width: 100%"
          @click="configStore.load()"
        >
          加载
        </n-button>
      </n-grid-item>
    </n-grid>
    <n-grid class="el-row-margin" :cols="1">
      <n-grid-item>
        <n-button
          secondary
          strong
          type="warning"
          style="width: 100%"
          @click="show"
        >
          查看
        </n-button>
      </n-grid-item>
    </n-grid>
    <n-grid class="el-row-margin" :cols="1">
      <n-grid-item>
        <n-button
          secondary
          strong
          type="success"
          style="width: 100%"
          @click="configStore.save()"
        >
          导出
        </n-button>
      </n-grid-item>
    </n-grid>
    <div>
      <n-modal v-model:show="showModel">
        <n-card
          style="width: 60%"
          :bordered="false"
          size="huge"
          role="dialog"
          aria-modal="true"
        >
          <div>
            <AceEditor
              v-model="configJson"
              height="400px"
              :light-or-dark="useThemeStore().isLight"
            />
          </div>
        </n-card>
      </n-modal>
    </div>
  </div>
</template>

<style scoped lang="scss">
.el-row-margin {
  margin-top: 10px;
}

.light-green {
  height: 108px;
  background-color: rgba(0, 128, 0, 0.12);
}

.green {
  height: 108px;
  background-color: rgba(0, 128, 0, 0.24);
}
</style>
