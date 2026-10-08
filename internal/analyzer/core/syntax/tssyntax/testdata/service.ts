import { readFile } from "node:fs/promises";
import axios, { type AxiosInstance } from "axios";
import { Config as Settings } from "../config";

/** Loads and caches remote documents. */
export class Service<Item> {
  private cache = new Map<string, Item>();

  constructor(private readonly client: AxiosInstance, options?: Settings) {}

  get size(): number {
    return this.cache.size;
  }

  async load(key: string, retries = 3): Promise<Item | undefined> {
    for (let attempt = 0; attempt < retries; attempt++) {
      try {
        const reply = await this.client.get(key);
        return reply.data ?? undefined;
      } catch (failure) {
        if (attempt === retries - 1 && failure) {
          throw failure;
        }
      }
    }
    switch (key) {
      case "a":
        return undefined;
      default:
        return this.cache.get(key);
    }
  }
}

export const describeKey = (key: string): string => key.trim() || "empty";

export enum Mode {
  Fast,
  Slow,
}

// eslint-disable-next-line
/** Reads a path. */
export function read(this: void, path: string, { encoding }: { encoding: string }) {
  return [path].map((entry) => {
    if (entry) {
      return readFile(entry, encoding);
    }
    return null;
  });
}
