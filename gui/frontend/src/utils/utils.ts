export type Option = {
  label: string;
  value: string;
};

export const convertToOptions = (arr: string[]): Option[] => {
  return arr.map<Option>((val) => ({ label: val, value: val }));
};

export const formatBytes = (bytes?: number): string => {
  if (!bytes || bytes === 0) {
    return "0 B";
  }
  const units: string[] = ["B", "KB", "MB", "GB", "TB"];
  let index: number = 0;
  while (bytes >= 1024 && index < units.length - 1) {
    bytes /= 1024;
    index++;
  }
  return `${bytes.toFixed(2)} ${units[index]}`;
};
