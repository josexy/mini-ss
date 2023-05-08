<script setup lang="ts">
import { ref, reactive } from "vue";
import Card from "@/components/Card/index.vue";
import AceEditor from "@/components/AceEditor/index.vue";
import { useConfigStore } from "@/stores/modules/config";
import { watch } from "vue";
import { onMounted } from "vue";
import { computed } from "vue";
import { PlusFilled as AddIcon } from "@vicons/material";
import { useThemeStore } from "@/stores/modules/theme";
import { Option, convertToOptions } from "@/utils/utils";
import { toRaw } from "vue";
import {
  ChangeDirectTo,
  ChangeGlobalTo,
  ListKcpCrypts,
  ListKcpModes,
  ListMethods,
  ListSSRObfs,
  ListSSRProtocols,
  ListTransports,
} from "@/../wailsjs/go/main/App";
import { config } from "@/../wailsjs/go/models";

const configStore = useConfigStore();

const proxy = reactive({
  directTo: "",
  globalTo: "",
});

const initServerConfigState = {
  disable: false,
  name: "",
  type: "ss",
  addr: "",
  password: "",
  method: "aes-256-cfb",
  transport: "default",
  udp: false,
  ssr: {
    protocol: "",
    protocol_param: "",
    obfs: "",
    obfs_param: "",
  },
  ws: {
    host: "",
    path: "",
    compress: false,
    tls: false,
  },
  obfs: {
    host: "",
  },
  kcp: {
    key: "",
    conns: 0,
    crypt: "",
    mode: "",
    compress: false,
  },
  quic: {
    conns: 0,
  },
  grpc: {
    hostname: "",
    cert_path: "",
    key_path: "",
    ca_path: "",
    tls: false,
  },
};

const serverConfigModel = reactive({ ...initServerConfigState });

const isAddOrUpdate = ref(true);
const showModalForDetail = ref(false);
const showModalInput = ref(false);
// 当前选择的card索引
const cardIndex = ref(-1);
// 当前选择的input索引
const inputIndex = ref(0);
const selectedServerConfig = ref<config.ServerConfig>();

const transportOptions = ref<Option[]>([]);
const methodOptions = ref<Option[]>([]);
const kcpModeOptions = ref<Option[]>([]);
const kcpCryptOptions = ref<Option[]>([]);
const ssrObfsOptions = ref<Option[]>([]);
const ssrProtocolOptions = ref<Option[]>([]);
const typeOptions = ref<Option[]>([
  { label: "SS", value: "ss" },
  { label: "SSR", value: "ssr" },
]);

const isDefault = computed(() => serverConfigModel.transport === "default");
const isDefaultSSR = computed(
  () => isDefault && serverConfigModel.type === "ssr"
);
const isWs = computed(
  () => serverConfigModel.transport === "ws" && serverConfigModel.type === "ss"
);
const isKcp = computed(
  () => serverConfigModel.transport === "kcp" && serverConfigModel.type === "ss"
);
const isGrpc = computed(
  () =>
    serverConfigModel.transport === "grpc" && serverConfigModel.type === "ss"
);
const isQuic = computed(
  () =>
    serverConfigModel.transport === "quic" && serverConfigModel.type === "ss"
);
const isObfs = computed(
  () =>
    serverConfigModel.transport === "obfs" && serverConfigModel.type === "ss"
);

const jsonServerConfig = computed(() => {
  return JSON.stringify(selectedServerConfig.value, null, 2);
});

const directTo = computed({
  get: () => proxy.directTo,
  set: (value: string) => {
    proxy.directTo = value;
    ChangeDirectTo(value);
  },
});

const globalTo = computed({
  get: () => proxy.globalTo,
  set: (value: string) => {
    proxy.globalTo = value;
    ChangeGlobalTo(value);
  },
});

onMounted(() => {
  setup();
  setupOptions();
});

watch(
  () => configStore.config,
  () => setup()
);

const setupOptions = () => {
  ListTransports().then((res) => {
    transportOptions.value.push(...convertToOptions(res));
  });
  ListMethods().then((res) => {
    methodOptions.value.push(...convertToOptions(res));
  });
  ListSSRObfs().then((res) => {
    ssrObfsOptions.value.push(...convertToOptions(res));
  });
  ListSSRProtocols().then((res) => {
    ssrProtocolOptions.value.push(...convertToOptions(res));
  });
  ListKcpCrypts().then((res) => {
    kcpCryptOptions.value.push(...convertToOptions(res));
  });
  ListKcpModes().then((res) => {
    kcpModeOptions.value.push(...convertToOptions(res));
  });
};

const setup = () => {
  proxy.directTo = configStore.config?.rules?.direct_to || "";
  proxy.globalTo = configStore.config?.rules?.global_to || "";
  configStore.config?.server?.forEach((val, index) => {
    if (configStore.config?.rules?.global_to === val.name) {
      cardIndex.value = index;
    }
  });
};

const changeCard = (index: number, value: config.ServerConfig) => {
  // 选择对应的card
  cardIndex.value = index;
  if (inputIndex.value === 0) {
    directTo.value = value.name;
  } else {
    globalTo.value = value.name;
  }
};

const closeCard = (index: number) => {
  // 0/1
  if (inputIndex.value === 0) {
    directTo.value = "";
  } else {
    globalTo.value = "";
  }
  // 取消当前选择的card
  if (cardIndex.value === index) {
    cardIndex.value = -1;
  }
  configStore.deleteServerConfigByIndex(index);
};

const editCard = (cfg: config.ServerConfig) => {
  showModalInput.value = true;
  isAddOrUpdate.value = false;

  serverConfigModel.name = cfg.name;
  serverConfigModel.addr = cfg.addr;
  serverConfigModel.disable = cfg.disable || false;
  serverConfigModel.type = cfg.type || "ss";
  serverConfigModel.udp = cfg.udp || false;
  serverConfigModel.password = cfg.password || "";
  serverConfigModel.method = cfg.method || "";
  serverConfigModel.transport = cfg.transport || "default";
  if (cfg.ssr) {
    serverConfigModel.ssr.protocol = cfg.ssr.protocol || "";
    serverConfigModel.ssr.protocol_param = cfg.ssr.protocol_param || "";
    serverConfigModel.ssr.obfs = cfg.ssr.obfs || "";
    serverConfigModel.ssr.obfs_param = cfg.ssr.obfs_param || "";
  }
  if (cfg.ws) {
    serverConfigModel.ws.host = cfg.ws.host || "";
    serverConfigModel.ws.path = cfg.ws.path || "";
    serverConfigModel.ws.tls = cfg.ws.tls || false;
    serverConfigModel.ws.compress = cfg.ws.compress || false;
  }
  if (cfg.quic) {
    serverConfigModel.quic.conns = cfg.quic.conns || 0;
  }
  if (cfg.grpc) {
    serverConfigModel.grpc.hostname = cfg.grpc.hostname || "";
    serverConfigModel.grpc.ca_path = cfg.grpc.ca_path || "";
    serverConfigModel.grpc.cert_path = cfg.grpc.cert_path || "";
    serverConfigModel.grpc.key_path = cfg.grpc.key_path || "";
    serverConfigModel.grpc.tls = cfg.grpc.tls || false;
  }
  if (cfg.kcp) {
    serverConfigModel.kcp.key = cfg.kcp.key || "";
    serverConfigModel.kcp.mode = cfg.kcp.mode || "";
    serverConfigModel.kcp.conns = cfg.kcp.conns || 0;
    serverConfigModel.kcp.crypt = cfg.kcp.crypt || "";
    serverConfigModel.kcp.compress = cfg.kcp.compress || false;
  }
  if (cfg.obfs) {
    serverConfigModel.obfs.host = cfg.obfs.host || "";
  }
};

const showDetail = (cfg: config.ServerConfig) => {
  showModalForDetail.value = true;
  selectedServerConfig.value = cfg;
};

const showAddModel = () => {
  // reset serverConfigModel reactive
  Object.assign(serverConfigModel, initServerConfigState);

  showModalInput.value = true;
  isAddOrUpdate.value = true;
};

const addOrUpdateServerConfig = () => {
  const cfg = config.ServerConfig.createFrom(toRaw(serverConfigModel));
  // Add or Update
  if (isAddOrUpdate.value) {
    configStore.addServerConfig(cfg);
  } else {
    configStore.updateServerConfig(cfg);
  }
};
</script>

<template>
  <div>
    <n-grid :cols="24" :x-gap="5">
      <n-grid-item :span="12">
        <n-input-group>
          <n-input-group-label>直连</n-input-group-label>
          <n-input
            clearable
            readonly
            placeholder=""
            :status="inputIndex === 0 ? 'error' : 'success'"
            @click="inputIndex = 0"
            v-model:value="directTo"
          />
        </n-input-group>
      </n-grid-item>
      <n-grid-item :span="11">
        <n-input-group>
          <n-input-group-label>全局</n-input-group-label>
          <n-input
            clearable
            readonly
            placeholder=""
            :status="inputIndex === 1 ? 'error' : 'success'"
            @click="inputIndex = 1"
            v-model:value="globalTo"
          />
        </n-input-group>
      </n-grid-item>
      <n-grid-item :span="1">
        <n-button quaternary circle @click="showAddModel">
          <template #icon>
            <n-icon>
              <AddIcon />
            </n-icon>
          </template>
        </n-button>
      </n-grid-item>
    </n-grid>
    <n-divider>节点列表</n-divider>
    <div class="card-container">
      <Card
        v-for="(item, index) in configStore.config?.server"
        :key="item.name"
        :index="index"
        :clicked-index="cardIndex"
        :server-config="item"
        @click:card="changeCard"
        @click:detail="showDetail"
        @click:close="closeCard"
        @click:edit="editCard"
      />
    </div>
    <div>
      <n-modal v-model:show="showModalForDetail">
        <n-card
          style="width: 60%"
          :bordered="false"
          size="huge"
          role="dialog"
          aria-modal="true"
        >
          <div>
            <AceEditor
              v-model="jsonServerConfig"
              height="280px"
              :light-or-dark="useThemeStore().isLight"
            />
          </div>
          <template #footer>
            <div style="display: flex">
              <n-button
                size="small"
                style="margin-left: auto"
                @click="showModalForDetail = false"
              >
                取消
              </n-button>
            </div>
          </template>
        </n-card>
      </n-modal>
    </div>
    <div>
      <n-modal v-model:show="showModalInput">
        <n-card
          style="width: 60%"
          :bordered="false"
          size="huge"
          role="dialog"
          aria-modal="true"
          :title="isAddOrUpdate ? '添加节点配置' : '更新节点配置'"
        >
          <n-form
            size="small"
            :model="serverConfigModel"
            label-placement="left"
            label-width="auto"
            require-mark-placement="right-hanging"
          >
            <n-form-item label="禁用">
              <n-switch
                placeholder=""
                clearable
                v-model:value="serverConfigModel.disable"
              />
            </n-form-item>
            <n-form-item label="名称">
              <n-input
                placeholder=""
                :readonly="!isAddOrUpdate"
                clearable
                v-model:value="serverConfigModel.name"
              />
            </n-form-item>
            <n-form-item label="地址">
              <n-input
                v-model:value="serverConfigModel.addr"
                placeholder=""
                clearable
              />
            </n-form-item>
            <n-form-item label="密码">
              <n-input
                type="password"
                show-password-on="click"
                placeholder=""
                clearable
                v-model:value="serverConfigModel.password"
              />
            </n-form-item>
            <n-form-item label="协议类型">
              <n-select
                placeholder=""
                :options="typeOptions"
                v-model:value="serverConfigModel.type"
              />
            </n-form-item>
            <n-form-item label="加密方法">
              <n-select
                placeholder=""
                :options="methodOptions"
                v-model:value="serverConfigModel.method"
              />
            </n-form-item>
            <n-form-item label="传输类型">
              <n-select
                placeholder=""
                :options="transportOptions"
                v-model:value="serverConfigModel.transport"
              />
            </n-form-item>
            <n-form-item label="UDP" v-show="isDefault">
              <n-switch v-model:value="serverConfigModel.udp" />
            </n-form-item>

            <!-- default ssr -->
            <n-form-item label="协议" v-show="isDefaultSSR">
              <n-select
                placeholder=""
                :options="ssrProtocolOptions"
                v-model:value="serverConfigModel.ssr.protocol"
              />
            </n-form-item>
            <n-form-item label="协议参数" v-show="isDefaultSSR">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.ssr.protocol_param"
              />
            </n-form-item>
            <n-form-item label="混淆" v-show="isDefaultSSR">
              <n-select
                placeholder=""
                :options="ssrObfsOptions"
                v-model:value="serverConfigModel.ssr.obfs"
              />
            </n-form-item>
            <n-form-item label="混淆参数" v-show="isDefaultSSR">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.ssr.obfs_param"
              />
            </n-form-item>

            <!-- ws -->
            <n-form-item label="主机名" v-show="isWs">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.ws.host"
              />
            </n-form-item>
            <n-form-item label="路径" v-show="isWs">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.ws.path"
              />
            </n-form-item>
            <n-form-item label="压缩" v-show="isWs">
              <n-switch
                placeholder=""
                v-model:value="serverConfigModel.ws.compress"
              />
            </n-form-item>
            <n-form-item label="TLS" v-show="isWs">
              <n-switch
                placeholder=""
                v-model:value="serverConfigModel.ws.tls"
              />
            </n-form-item>

            <!-- obfs -->
            <n-form-item label="主机名" v-show="isObfs">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.obfs.host"
              />
            </n-form-item>

            <!-- grpc -->
            <n-form-item label="主机名" v-show="isGrpc">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.grpc.hostname"
              />
            </n-form-item>
            <n-form-item label="证书路径" v-show="isGrpc">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.grpc.cert_path"
              />
            </n-form-item>
            <n-form-item label="私钥路径" v-show="isGrpc">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.grpc.key_path"
              />
            </n-form-item>
            <n-form-item label="CA路径" v-show="isGrpc">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.grpc.ca_path"
              />
            </n-form-item>
            <n-form-item label="TLS" v-show="isGrpc">
              <n-switch v-model:value="serverConfigModel.grpc.tls" />
            </n-form-item>

            <!-- quic -->
            <n-form-item label="Conns" v-show="isQuic">
              <n-input-number
                placeholder=""
                clearable
                :min="0"
                :max="255"
                v-model:value="serverConfigModel.quic.conns"
              />
            </n-form-item>

            <!-- kcp -->
            <n-form-item label="Key" v-show="isKcp">
              <n-input
                placeholder=""
                clearable
                v-model:value="serverConfigModel.kcp.key"
              />
            </n-form-item>
            <n-form-item label="模式" v-show="isKcp">
              <n-select
                placeholder=""
                :options="kcpModeOptions"
                v-model:value="serverConfigModel.kcp.mode"
              />
            </n-form-item>
            <n-form-item label="加密方法" v-show="isKcp">
              <n-select
                placeholder=""
                :options="kcpCryptOptions"
                v-model:value="serverConfigModel.kcp.crypt"
              />
            </n-form-item>
            <n-form-item label="Conns" v-show="isKcp">
              <n-input-number
                placeholder=""
                clearable
                :min="0"
                :max="255"
                v-model:value="serverConfigModel.kcp.conns"
              />
            </n-form-item>
            <n-form-item label="压缩" v-show="isKcp">
              <n-switch
                placeholder=""
                v-model:value="serverConfigModel.kcp.compress"
              />
            </n-form-item>
          </n-form>

          <template #footer>
            <div class="input-modal-footer">
              <n-button round type="primary" @click="addOrUpdateServerConfig">
                {{ isAddOrUpdate ? "添加" : "更新" }}
              </n-button>
              <n-button round @click="showModalInput = false"> 取消 </n-button>
            </div>
          </template>
        </n-card>
      </n-modal>
    </div>
  </div>
</template>

<style scoped>
.card-container {
  display: flex;
  flex-wrap: wrap;
  align-content: flex-start;
}

.input-modal-footer {
  display: flex;
  flex-direction: row;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 10px;
}

::v-deep(.n-form-item) {
  height: 35px;
}
</style>
