import { defineStore } from "pinia";
import { reactive, ref } from "vue";
import { useMessage } from "naive-ui";
import {
  AddServerConfig,
  UpdateServerConfig,
  DeleteServerConfig,
  LoadConfig,
  SaveConfig,
} from "@/../wailsjs/go/main/App";
import { config as configNs, main } from "@/../wailsjs/go/models";
import { RuleConfig, Rule } from "./type";

export const useConfigStore = defineStore("config", () => {
  const message = useMessage();
  const path = ref("");
  const config = ref<configNs.Config>();
  const rule = ref<RuleConfig>({ mode: "" });

  const load = () => {
    LoadConfig()
      .then((res) => {
        const cfg = res as main.Config;
        path.value = cfg.path;
        config.value = cfg.value;
        convertRules();
      })
      .catch((err) => {
        message.error(err);
      });
  };

  const save = () => {
    if (config.value) {
      SaveConfig(config.value)
        .then(() => {})
        .catch((err) => {
          message.error(err);
        });
    }
  };

  const addServerConfig = (cfg: configNs.ServerConfig) => {
    if (!cfg || cfg.name.trim() === "") {
      message.error("名称不能为空！");
      return;
    }
    if (!config.value) {
      config.value = configNs.Config.createFrom({
        server: new Array<configNs.ServerConfig>(),
      });
    }

    const index = config.value.server?.findIndex(
      (value) => value.name === cfg.name
    );
    if (index !== -1) {
      message.error('已经存在"' + cfg.name + '"');
      return;
    }
    config.value.server?.push(cfg);
    AddServerConfig(cfg)
      .then(() => {
        message.success("添加成功！");
      })
      .catch((err) => {
        message.error(err);
      });
  };

  const deleteServerConfigByIndex = (index: number) => {
    if (
      config.value &&
      config.value.server &&
      index >= 0 &&
      index < config.value.server.length
    ) {
      const name = config.value.server[index].name;
      config.value.server.splice(index, 1);

      DeleteServerConfig(name)
        .then(() => {
          message.success("删除成功！");
        })
        .catch((err) => {
          message.error(err);
        });
    }
  };

  const deleteServerConfig = (name: string) => {
    if (config.value && config.value.server) {
      config.value.server.splice(
        config.value.server.findIndex((val) => val.name === name),
        1
      );
      DeleteServerConfig(name)
        .then(() => {
          message.success("删除成功！");
        })
        .catch((err) => {
          message.error(err);
        });
    }
  };

  const updateServerConfig = (cfg: configNs.ServerConfig) => {
    if (!cfg || !config.value || !config.value.server) {
      return;
    }
    const index = config.value.server.findIndex(
      (item) => item.name === cfg.name
    );
    if (index === -1) {
      return;
    }
    if (config.value.server) {
      config.value.server[index].addr = cfg.addr;
      config.value.server[index].method = cfg.method;
      config.value.server[index].transport = cfg.transport;
      config.value.server[index].password = cfg.password;
      config.value.server[index].disable = cfg.disable;
      config.value.server[index].type = cfg.type;
      config.value.server[index].udp = cfg.udp;
      config.value.server[index].ws = cfg.ws;
      config.value.server[index].grpc = cfg.grpc;
      config.value.server[index].obfs = cfg.obfs;
      config.value.server[index].ssr = cfg.ssr;
      config.value.server[index].quic = cfg.quic;
      config.value.server[index].kcp = cfg.kcp;
      UpdateServerConfig(cfg)
        .then(() => {
          message.success("更新成功！");
        })
        .catch((err) => {
          message.error(err);
        });
    }
  };

  const convert = (
    type: string,
    proxy: string,
    action: string,
    target: string[]
  ): Rule[] => {
    const res = new Array<Rule>();
    target.forEach((item) => {
      res.push({
        type: type,
        target: item,
        proxy: proxy,
        action: action,
      });
    });
    return res;
  };

  const convertRules = () => {
    if (!config.value || !config.value.rules) {
      return;
    }
    rule.value.mode = config.value.rules.mode;
    rule.value.rules = new Array<Rule>();
    if (config.value.rules.mode === "match") {
      config.value.rules.match?.domain?.forEach((val) => {
        rule.value.rules?.push(
          ...convert("Domain", val.proxy, val.action, val.value)
        );
      });
      config.value.rules.match?.domain_suffix?.forEach((val) => {
        rule.value.rules?.push(
          ...convert("Domain-Suffix", val.proxy, val.action, val.value)
        );
      });
      config.value.rules.match?.domain_keyword?.forEach((val) => {
        rule.value.rules?.push(
          ...convert("Domain-Keyword", val.proxy, val.action, val.value)
        );
      });
      config.value.rules.match?.geoip?.forEach((val) => {
        rule.value.rules?.push(
          ...convert("GeoIP", val.proxy, val.action, val.value)
        );
      });
      config.value.rules.match?.ipcidr?.forEach((val) => {
        rule.value.rules?.push(
          ...convert("IP-CIDR", val.proxy, val.action, val.value)
        );
      });

      let proxy = "";
      let action = "accept";
      switch (config.value.rules.match?.others) {
        case "global":
          proxy = config.value.rules.global_to || "";
          break;
        case "direct":
          proxy = config.value.rules.direct_to || "";
          break;
        default:
          proxy = config.value.rules.match?.others || "";
          break;
      }
      if (proxy === "") action = "drop";
      rule.value.rules?.push({
        type: "Others",
        target: "*",
        action: action,
        proxy: proxy,
      });
    }
  };

  return {
    path,
    config,
    rule,
    load,
    save,
    addServerConfig,
    deleteServerConfig,
    deleteServerConfigByIndex,
    updateServerConfig,
  };
});
