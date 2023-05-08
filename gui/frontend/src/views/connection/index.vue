<script setup lang="ts">
import { h, ref, reactive, onMounted } from "vue";
import { EventsOff, EventsOn } from "@/../wailsjs/runtime/runtime";
import { formatBytes } from "@/utils/utils";
import {
  DownloadRound as DownloadIcon,
  UploadRound as UploadIcon,
  CloudUploadRound as UploadBytesIcon,
  CloudDownloadRound as DownloadBytesIcon,
} from "@vicons/material";
import type { DataTableColumns } from "naive-ui";

interface ConnectionSnapshot {
  src: string;
  dst: string;
  network: string;
  type: string;
  rule: string;
  proxy: string;
  download: number;
  upload: number;
}

interface Snapshot {
  download_total: number;
  upload_total: number;
  connections: ConnectionSnapshot[];
}

const traffic = reactive({
  download: "0.00 B/s",
  upload: "0.00 B/s",
  snapshot: {} as Snapshot,
});

onMounted(() => {
  EventsOff("event-traffic-speed");
  EventsOff("event-traffic-snapshot");

  EventsOn("event-traffic-speed", (download: string, upload: string) => {
    traffic.download = download;
    traffic.upload = upload;
  });
  EventsOn("event-traffic-snapshot", (data: any) => {
    if (data && data.download_total && data.upload_total) {
      traffic.snapshot.download_total = data.download_total;
      traffic.snapshot.upload_total = data.upload_total;
    }
    if (data.connections) {
      traffic.snapshot.connections = data.connections as ConnectionSnapshot[];
    } else {
      traffic.snapshot.connections = new Array<ConnectionSnapshot>();
    }
  });
});

const columns: DataTableColumns<ConnectionSnapshot> = [
  {
    title: "源IP",
    key: "src",
  },
  {
    title: "域名",
    key: "dst",
  },
  {
    title: "网络",
    key: "network",
    sorter: "default",
  },
  {
    title: "类型",
    key: "type",
    sorter: "default",
  },
  {
    title: "规则",
    key: "rule",
    sorter: "default",
  },
  {
    title: "代理",
    key: "proxy",
    render(row) {
      return h(
        "span",
        {},
        { default: () => (row.proxy === "" ? "直连" : row.proxy) }
      );
    },
    sorter: "default",
  },
  {
    title: "下载",
    key: "download",
    render(row) {
      return h("span", {}, { default: () => formatBytes(row.download) });
    },
    sorter: (row1, row2) => row1.download - row2.download,
  },
  {
    title: "上传",
    key: "upload",
    render(row) {
      return h("span", {}, { default: () => formatBytes(row.upload) });
    },
    sorter: (row1, row2) => row1.upload - row2.upload,
  },
];
</script>

<template>
  <div class="container">
    <n-card>
      <div class="card-container">
        <div class="icon-text">
          <n-icon size="20px" color="#339900">
            <DownloadBytesIcon />
          </n-icon>
          <span
            >总下载字节数: {{ formatBytes(traffic.snapshot.download_total) }}
          </span>
        </div>

        <div class="icon-text">
          <n-icon size="20px" color="#ffcc00">
            <UploadBytesIcon />
          </n-icon>
          <span
            >总上传字节数: {{ formatBytes(traffic.snapshot.upload_total) }}
          </span>
        </div>
      </div>

      <div class="card-container">
        <div class="icon-text">
          <n-icon size="20px" color="#339900">
            <DownloadIcon />
          </n-icon>
          <span>下载速率: {{ traffic.download }} </span>
        </div>

        <div class="icon-text">
          <n-icon size="20px" color="#ffcc00">
            <UploadIcon />
          </n-icon>
          <span>上传速率: {{ traffic.upload }} </span>
        </div>
      </div>
    </n-card>
    <div class="table-wrapper">
      <n-data-table
        :columns="columns"
        :data="traffic.snapshot.connections"
        :striped="true"
        :bordered="true"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
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
  overflow-x: auto;
}

.card-container {
  display: flex;
  gap: 5%;
  justify-content: center;

  .icon-text {
    display: flex;
    gap: 5px;
  }
}
</style>
