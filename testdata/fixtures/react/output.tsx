export function Card({ title, children }: {
  title: string;
  children: string;
}) {
  return <section className="card" id={title}><h1>{title}</h1><p>
    {children}
  </p></section>;
}
