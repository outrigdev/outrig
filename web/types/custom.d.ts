declare global {
  type LogLine = {
    linenum: number;
    ts: number;
    msg: string;
    source: string;
  };
}

export {};
