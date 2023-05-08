<script setup lang="ts">
import { ref, reactive } from "vue";
import { SpeedTest } from "@/../wailsjs/go/main/App";
import { config } from "@/../wailsjs/go/models";
import { computed } from "vue";
import {
  CloseFilled as CloseIcon,
  TextSnippetOutlined as MoreIcon,
  SpeedFilled as SpeedTestIcon,
  EditNoteFilled as EditIcon,
} from "@vicons/material";
import { useThemeStore } from "@/stores/modules/theme";
import { useMessage } from "naive-ui";

const props = defineProps<{
  serverConfig: config.ServerConfig;
  index: number;
  clickedIndex: number;
}>();

const emits = defineEmits<{
  (event: "click:card", index: number, cfg: config.ServerConfig): void;
  (event: "click:close", index: number): void;
  (event: "click:detail", cfg: config.ServerConfig): void;
  (event: "click:edit", cfg: config.ServerConfig): void;
}>();

const message = useMessage();

const isClosed = ref(false);
const latency = ref("N/A");

const serverProtocol = computed(() => {
  if (props.serverConfig.type) {
    switch (props.serverConfig.type) {
      case "ss":
        return "SS";
      case "ssr":
        return "SSR";
      default:
        return "SS";
    }
  } else {
    return "SS";
  }
});

const latencyColor = computed(() => {
  if (latency.value === "timeout") {
    return "red";
  }
  const parts = latency.value.split(/([a-zA-Zµ]+)/);
  if (parts.length >= 2) {
    switch (parts[1]) {
      case "s":
        return "#f2e13b";
      case "ms":
        return parseFloat(parts[0]) <= 60 ? "#30cc18" : "#f2e13b";
      case "µs":
        return "#30cc18";
      default:
        break;
    }
  }
  if (!useThemeStore().isLight) {
    return "rgb(255, 255, 255, 0.82)";
  } else {
    return "rgb(51, 54, 57)";
  }
});

const speedTest = () => {
  if (props.serverConfig.disable) {
    return;
  }
  SpeedTest(props.serverConfig.addr)
    .then((res) => {
      latency.value = res;
    })
    .catch((err) => {
      message.error(err);
    });
};

const close = () => {
  isClosed.value = true;
  emits("click:close", props.index);
};

const click = () => {
  emits("click:card", props.index, props.serverConfig);
};
</script>

<template>
  <div
    class="card"
    :class="clickedIndex === index ? 'card-wrapper-click' : 'card-wrapper'"
    v-if="!isClosed"
    @click="click"
  >
    <div class="card-header">
      <n-tooltip trigger="hover">
        <template #trigger>
          <n-tag
            size="small"
            :bordered="false"
            :type="serverConfig.disable ? 'error' : 'success'"
          >
            {{ serverProtocol }}
          </n-tag>
        </template>
        {{ serverConfig.disable ? "禁止" : "开启" }}
      </n-tooltip>

      <div class="right-button">
        <n-button
          quaternary
          circle
          size="small"
          @click.prevent.stop
          @click="speedTest()"
        >
          <template #icon>
            <n-icon>
              <SpeedTestIcon />
            </n-icon>
          </template>
        </n-button>
      </div>
      <div>
        <n-button
          quaternary
          circle
          size="small"
          @click.prevent.stop
          @click="$emit('click:detail', serverConfig)"
        >
          <template #icon>
            <n-icon>
              <MoreIcon />
            </n-icon>
          </template>
        </n-button>
      </div>
      <div>
        <n-button
          quaternary
          circle
          size="small"
          @click.prevent.stop
          @click="$emit('click:edit', serverConfig)"
        >
          <template #icon>
            <n-icon>
              <EditIcon />
            </n-icon>
          </template>
        </n-button>
      </div>
      <div>
        <n-button
          quaternary
          circle
          size="small"
          @click.prevent.stop
          @click="close()"
        >
          <template #icon>
            <n-icon color="red">
              <CloseIcon />
            </n-icon>
          </template>
        </n-button>
      </div>
    </div>
    <div class="card-body">
      {{ serverConfig.name }}
    </div>
    <div class="card-bottom">
      <div class="card-left">{{ latency }}</div>
      <div class="card-right">
        <n-tag
          :bordered="false"
          size="small"
          type="warning"
          v-if="serverConfig.udp"
        >
          UDP
        </n-tag>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.card-wrapper {
  border: 1px solid rgb(234, 228, 228);
}

.card-wrapper-click {
  border: 1px solid rgb(234, 228, 228);
  box-shadow: 0 0 1px 3px rgb(233, 232, 232);
}

.card {
  margin: 10px;
  width: 20%;
  height: 80px;
  min-width: 135px;
  padding: 10px;
  border-radius: 2px;
  transition: all 0.5s ease;
  cursor: pointer;

  .card-header {
    display: flex;
    align-items: center;

    .right-button {
      margin-left: auto;
    }
  }

  .card-body {
    text-align: center;
    margin-bottom: 30px;
  }

  .card-bottom {
    display: flex;
    justify-content: space-between;
    position: relative;

    .card-left {
      font-size: smaller;
      position: absolute;
      bottom: 0;
      left: 0;
      color: v-bind(latencyColor);
    }

    .card-right {
      position: absolute;
      right: 0;
      bottom: 0;
    }
  }
}
</style>
