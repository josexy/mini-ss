<script setup lang="ts">
import { useMessage } from "naive-ui";
import { onMounted } from "vue";
import { ref, reactive, watch } from "vue";
import { useConfigStore } from "@/stores/modules/config";
import {
  ListOutboundInterfaces,
  StartServer,
  StopServer,
} from "@/../wailsjs/go/main/App";

const form = reactive({
  socks_addr: "",
  socks_auth: "",
  http_addr: "",
  http_auth: "",
  mixed_addr: "",
  enable_system_proxy: false,
  enable_tun_mode: false,
  outbound_interface: "",
  enhancer_tun_name: "",
  enhancer_tun_cidr: "",
  enhancer_tun_mtu: 0,
  enhancer_nameservers: Array<string>(),
  enhancer_dns: "",
});

const outbound_interfaces = ref<{ label: string; value: string }[]>([]);
const start_ok = ref(false);

const message = useMessage();
const configStore = useConfigStore();

onMounted(() => {
  listOutbound();
});

watch(
  () => configStore.config,
  () => {
    setupLocalConfig();
  }
);

const listOutbound = () => {
  ListOutboundInterfaces().then((res) => {
    outbound_interfaces.value.length = 0;
    outbound_interfaces.value.push({ value: "auto", label: "自动" });
    res.forEach((name) => {
      outbound_interfaces.value.push({
        value: name,
        label: name,
      });
    });
  });
};

const setupLocalConfig = () => {
  if (configStore.config?.auto_detect_iface) {
    form.outbound_interface = "auto";
  } else {
    form.outbound_interface = configStore.config?.iface || "";
  }

  const local = configStore.config?.local;
  form.socks_addr = local?.socks_addr || "";
  form.socks_auth = local?.socks_auth || "";
  form.http_addr = local?.http_addr || "";
  form.http_auth = local?.http_auth || "";
  form.mixed_addr = local?.mixed_addr || "";
  form.enable_system_proxy = local?.system_proxy || false;
  form.enable_tun_mode = local?.enable_tun || false;
  form.enhancer_tun_name = local?.tun?.name || "utun5";
  form.enhancer_tun_cidr = local?.tun?.cidr || "192.18.0.1/16";
  form.enhancer_tun_mtu = local?.tun?.mtu || 1500;
  form.enhancer_nameservers = local?.fake_dns?.nameservers || [
    "114.114.114.114",
    "8.8.8.8",
  ];
  form.enhancer_dns = local?.fake_dns?.listen || ":53";
};

const applyCurrentConfig = () => {
  if (configStore.config) {
    configStore.config.auto_detect_iface = form.outbound_interface === "auto";
    configStore.config.iface = form.outbound_interface;
  }
  const local = configStore.config?.local;
  if (local) {
    local.socks_addr = form.socks_addr;
    local.socks_auth = form.socks_auth;
    local.http_addr = form.http_addr;
    local.http_auth = form.http_auth;
    local.mixed_addr = form.mixed_addr;
    local.enable_tun = form.enable_tun_mode;
    local.system_proxy = form.enable_system_proxy;
    if (local.enable_tun) {
      local.tun = {
        name: form.enhancer_tun_name,
        cidr: form.enhancer_tun_cidr,
        mtu: form.enhancer_tun_mtu,
      };
      local.fake_dns = {
        listen: form.enhancer_dns,
        nameservers: form.enhancer_nameservers,
      };
    }
  }
};

const startServer = () => {
  console.log(form);
  if (configStore.config) {
    applyCurrentConfig();
    StartServer(configStore.config)
      .then(() => {
        start_ok.value = true;
        message.success("启动服务器！");
      })
      .catch((err) => {
        message.error(err);
      });
  } else {
    message.error("配置未加载！");
  }
};

const stopServer = () => {
  StopServer()
    .then(() => {
      start_ok.value = false;
      message.success("停止服务器！");
    })
    .catch((err) => {
      message.error(err);
    });
};
</script>

<template>
  <div>
    <n-grid :cols="1">
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>SOCKS</n-input-group-label>
          <n-input
            placeholder="SOCKS代理地址"
            v-model:value="form.socks_addr"
            :readonly="form.mixed_addr.length != 0"
          />
          <n-input
            placeholder="认证 (user:password)"
            v-model:value="form.socks_auth"
            :readonly="form.mixed_addr.length != 0"
          />
        </n-input-group>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="1" class="grid-margin">
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>HTTP</n-input-group-label>
          <n-input
            placeholder="HTTP代理地址"
            v-model:value="form.http_addr"
            :readonly="form.mixed_addr.length != 0"
          />
          <n-input
            placeholder="认证 (user:password)"
            v-model:value="form.http_auth"
            :readonly="form.mixed_addr.length != 0"
          />
        </n-input-group>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="1" class="grid-margin">
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>混合</n-input-group-label>
          <n-input placeholder="混合代理地址" v-model:value="form.mixed_addr" />
        </n-input-group>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="1" class="grid-margin">
      <n-grid-item :span="1">
        <n-checkbox v-model:checked="form.enable_system_proxy">
          系统代理
        </n-checkbox>
        <n-checkbox v-model:checked="form.enable_tun_mode">
          增强模式
        </n-checkbox>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="1" class="grid-margin">
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>出站接口</n-input-group-label>
          <n-select
            v-model:value="form.outbound_interface"
            :options="outbound_interfaces"
            @update:value="listOutbound()"
          />
        </n-input-group>
      </n-grid-item>
    </n-grid>

    <n-divider v-if="form.enable_tun_mode"> 增强模式 </n-divider>

    <n-grid
      :cols="3"
      class="grid-margin"
      :x-gap="5"
      v-if="form.enable_tun_mode"
    >
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>设备名称</n-input-group-label>
          <n-input
            v-model:value="form.enhancer_tun_name"
            placeholder="如 tun1"
            clearable
          />
        </n-input-group>
      </n-grid-item>
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>CIRD</n-input-group-label>
          <n-input
            v-model:value="form.enhancer_tun_cidr"
            placeholder="如 192.18.0.1/16"
            clearable
          />
        </n-input-group>
      </n-grid-item>
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>MTU</n-input-group-label>
          <n-input-number
            v-model:value="form.enhancer_tun_mtu"
            placeholder=""
          />
        </n-input-group>
      </n-grid-item>
    </n-grid>

    <n-grid
      :cols="2"
      class="grid-margin"
      :x-gap="5"
      v-if="form.enable_tun_mode"
    >
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>FakeDNS</n-input-group-label>
          <n-input
            v-model:value="form.enhancer_dns"
            placeholder="如 :53"
            clearable
          />
        </n-input-group>
      </n-grid-item>
      <n-grid-item :span="1">
        <n-input-group>
          <n-input-group-label>Nameservers</n-input-group-label>
          <n-select
            v-model:value="form.enhancer_nameservers"
            filterable
            multiple
            tag
            placeholder=""
          />
        </n-input-group>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="2" class="grid-margin" :x-gap="5">
      <n-grid-item :span="1">
        <n-button
          style="width: 100%"
          type="success"
          secondary
          strong
          :disabled="start_ok"
          @click="startServer()"
        >
          启动
        </n-button>
      </n-grid-item>
      <n-grid-item :span="1">
        <n-button
          style="width: 100%"
          type="warning"
          secondary
          strong
          :disabled="!start_ok"
          @click="stopServer()"
        >
          停止
        </n-button>
      </n-grid-item>
    </n-grid>

    <n-grid :cols="1" class="grid-margin" :x-gap="5" v-if="start_ok">
      <n-grid-item :span="1">
        <n-alert type="success"> 本地客户端启动成功！ </n-alert>
      </n-grid-item>
    </n-grid>
  </div>
</template>
