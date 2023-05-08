<script setup lang="ts">
import { computed } from "vue";
import { VAceEditor } from "vue3-ace-editor";
import "ace-builds/src-noconflict/mode-yaml";
import "ace-builds/src-noconflict/mode-json";
import "ace-builds/src-noconflict/theme-chrome";
import "ace-builds/src-noconflict/theme-tomorrow_night";
import "ace-builds/src-noconflict/ext-language_tools";

const props = withDefaults(
  defineProps<{
    modelValue: string;
    height?: string;
    // support json and yaml format
    lang?: string;
    readOnly?: boolean;
    // light: true, dark: false
    lightOrDark?: boolean;
  }>(),
  {
    height: "200px",
    lang: "json",
    readOnly: true,
    lightOrDark: true,
  }
);

const emits = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

const value = computed({
  get: () => props.modelValue,
  set: (val: string) => {
    emits("update:modelValue", val);
  },
});

const borderColor = computed(() => {
  return props.lightOrDark ? "rgb(213, 218, 225)" : "rgb(69, 67, 67)";
});
</script>

<template>
  <VAceEditor
    class="editor"
    wrap
    v-model:value="value"
    :lang="lang"
    :readonly="readOnly"
    :theme="lightOrDark ? 'chrome' : 'tomorrow_night'"
    :style="`height: ${height}`"
    :options="{
      showPrintMargin: false,
      showGutter: false,
      enableBasicAutocompletion: false,
      enableLiveAutocompletion: false,
      enableSnippets: false,
      showLineNumbers: false,
    }"
  />
</template>

<style scoped lang="scss">
.editor {
  border-radius: 2px;
  border: 1px solid v-bind(borderColor);
}
</style>
