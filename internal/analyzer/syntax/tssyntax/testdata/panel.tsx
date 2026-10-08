import React from "react";
import { useItems } from "./hooks";

interface PanelProps {
  title: string;
  onPick(index: number): void;
}

export default function Panel({ title, onPick }: PanelProps) {
  const items = useItems();
  const handlers = items.map((item, index) => () => onPick(index));
  return (
    <section>
      <h1>{title}</h1>
      {items.length > 0 ? handlers.map((go) => <a onClick={go}>{title}</a>) : null}
    </section>
  );
}
