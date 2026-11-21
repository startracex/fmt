// keep me
const a = 1; // end of line

/* between */
const b = 2;
export type Result<T> =
  | { ok: true; value: T }
  | { ok: false; error: string };
type Long = VeryLongName | AnotherVeryLongName | YetAnotherLongName | StillGoing;
function f(a: number, b = 1, ...rest: string[]) {
  return a;
}
for (let i = 0; i < 3; i++) {
  total += i;
}
for (const x of xs) {
  use(x);
}
try {
  work();
} catch (e) {
  fail(e);
} finally {
  done();
}
switch (kind) {
  case "a":
    run();
    break;
  default:
    halt();
}
const { a: aa, b: c, ...rest } = obj;
const [x, y = 1, ...zs] = arr;
obj.foo().bar().baz();
const s = `hello ${name}!`;
enum Color { Red = 1, Blue }
async function load(url: string) {
  const res = await fetch(url);
  return res.json();
}
a = b
;(function () {})()
