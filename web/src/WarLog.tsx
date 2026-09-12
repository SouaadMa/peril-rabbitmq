import type { LogEntry } from "./types";

type Props = {
  log: LogEntry[];
};

export function WarLog({ log }: Props) {
  if (log.length === 0) {
    return <p className="empty">No wars yet.</p>;
  }

  const newestFirst = [...log].reverse();

  return (
    <ul className="log">
      {newestFirst.map((entry, index) => (
        <li key={`${entry.time}-${index}`}>
          <time>{new Date(entry.time).toLocaleTimeString()}</time>
          <span>{entry.message}</span>
        </li>
      ))}
    </ul>
  );
}
