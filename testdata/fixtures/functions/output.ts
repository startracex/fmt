export async function loadUser(id: string, opts?: { cache: boolean }) {
  const res = await fetch("/users/" + id);
  return res.json();
}

const greet = (name: string) => {
  return "Hello " + name;
};

class Counter {
  value = 0;
  inc() {
    this.value++;
    return this.value;
  }
}
