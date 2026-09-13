import "./App.css";
import { playerColors } from "./colors";
import { PlayerList } from "./PlayerList";
import { useWorld } from "./useWorld";
import { WarLog } from "./WarLog";
import { WorldMap } from "./WorldMap";

function App() {
  const { snapshot, slots, connected } = useWorld();
  const colors = playerColors(slots);

  return (
    <div className="app">
      <header>
        <h1>Peril</h1>
        <span className={connected ? "pill live" : "pill down"}>
          {connected ? "live" : "reconnecting…"}
        </span>
      </header>
      <main className={connected ? undefined : "stale"}>
        {snapshot === null ? (
          <p className="empty waiting">Connecting to the gateway…</p>
        ) : (
          <>
            <WorldMap locations={snapshot.locations} colors={colors} />
            <aside>
              <h2>Players</h2>
              <PlayerList players={snapshot.players} colors={colors} />
              <h2>War log</h2>
              <WarLog log={snapshot.log} />
            </aside>
          </>
        )}
      </main>
    </div>
  );
}

export default App;
