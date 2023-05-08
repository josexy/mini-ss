export interface Rule {
  type: string;
  target: string;
  proxy: string;
  action: string;
}

export interface RuleConfig {
  mode: string;
  rules?: Rule[];
}
