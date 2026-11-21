import type {
  ReactiveController,
  ReactiveControllerHost,
} from "@lit/reactive-element";
import { registerFocusEvents, unregisterFocusEvents } from "./events.js";
import {
  type FetcherType,
  type SWRConfig,
  getDefaultConfig,
  isDisabledKey,
} from "./config.js";
class A {
  async revalidate(options) {
    if (this.disposed) {
      return;
    }
    if (this.config.isDisabled) {
      return;
    }

    const force = options;
    const cached = result;

    if (cached) {
      const changed = 1;
    }
  }
}
const value =
  someVeryLongCondition && anotherVeryLongCondition && aThirdVeryLongCondition;
function add(a, b) {
  return a + b;
}
const greet = (name) => {
  return "Hello " + name;
};
export default class {
  value = 0;
  inc() {
    this.value++;
    return this.value;
  }
}
interface User {
  id: string;
  name: string;
  greet(title: string): string;
}
type Result<T> = { ok: true; value: T } | { ok: false; error: string };
export function Card({ title, children }) {
  return (
    <section className="card" id={title}>
      <h1>{title}</h1>
      <p>{children}</p>
    </section>
  );
}
const o = {
  m() {
    return 1;
  },
  a: 1,
  b: 2,
};
const el = (
  <div>
    <span>hi</span>
  </div>
);
