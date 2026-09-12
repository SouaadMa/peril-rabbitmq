import type { PlayerView } from "./types";

type Props = {
  players: PlayerView[];
  colors: Map<string, string>;
};

export function PlayerList({ players, colors }: Props) {
  if (players.length === 0) {
    return <p className="empty">Nobody has joined yet.</p>;
  }

  return (
    <ul className="players">
      {players.map((player) => (
        <li key={player.username}>
          <span
            className="swatch"
            style={{ background: colors.get(player.username) }}
          />
          <span>{player.username}</span>
          <span className="count">{player.units.length} units</span>
        </li>
      ))}
    </ul>
  );
}
