import { defineStore } from "pinia";
import { ref } from "vue";
import { darkTheme, lightTheme } from "naive-ui";

export const useThemeStore = defineStore("theme", () => {
  const isLight = ref(true);
  const theme = ref(lightTheme);

  const toggle = () => {
    if (isLight.value) {
      isLight.value = false;
      theme.value = darkTheme;
    } else {
      isLight.value = true;
      theme.value = lightTheme;
    }
  };

  return {
    isLight,
    theme,
    toggle,
  };
});
