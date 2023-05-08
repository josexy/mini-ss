<script setup lang="ts">
import {
  RuleFilled as RuleIcon,
  CloudDownloadFilled as ConnectionIcon,
  InfoFilled as AboutIcon,
  MessageOutlined as ServerIcon,
  HomeFilled as HomeIcon,
  SettingsFilled as SettingIcon,
} from "@vicons/material";
import { h } from "vue";
import type { MenuOption } from "naive-ui";
import { RouterLink } from "vue-router";
import { Component } from "vue";
import { NIcon } from "naive-ui";

const renderIcon = (icon: Component) => {
  return () => h(NIcon, null, { default: () => h(icon) });
};

const menuOptions: MenuOption[] = [
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: "home",
          },
        },
        { default: () => "主页" }
      ),
    key: "home",
    icon: renderIcon(HomeIcon),
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: "server",
          },
        },
        { default: () => "节点" }
      ),
    key: "server",
    icon: renderIcon(ServerIcon),
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: "connection",
          },
        },
        { default: () => "连接" }
      ),
    key: "connection",
    icon: renderIcon(ConnectionIcon),
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: "rule",
          },
        },
        { default: () => "规则" }
      ),
    key: "rule",
    icon: renderIcon(RuleIcon),
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: "setting",
          },
        },
        { default: () => "设置" }
      ),
    key: "setting",
    icon: renderIcon(SettingIcon),
  },
  {
    label: () =>
      h(
        RouterLink,
        {
          to: {
            name: "about",
          },
        },
        { default: () => "关于" }
      ),
    key: "about",
    icon: renderIcon(AboutIcon),
  },
];
</script>

<template>
  <n-menu :options="menuOptions" />
</template>
