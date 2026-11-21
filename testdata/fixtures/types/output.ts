export type Result<T> = { ok: true; value: T } | { ok: false; error: string };

interface User {
  id: string;
  name: string;
  greet(title: string): string;
}

type Rec<T> = { readonly [K in keyof T]?: T[K] };
