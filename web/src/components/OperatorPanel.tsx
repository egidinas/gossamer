import type { ReactNode } from "react";

export function OperatorPanel({ title, meta, children }: { title: string; meta?: string; children: ReactNode }) {
  return (
    <section className="panel">
      <div className="panel-head">
        <h2>{title}</h2>
        {meta ? <span>{meta}</span> : null}
      </div>
      {children}
    </section>
  );
}

