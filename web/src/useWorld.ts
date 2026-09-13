import { useEffect, useRef, useState } from "react";
import { assignSlots } from "./colors";
import type { Snapshot } from "./types";

type WorldState = {
  snapshot: Snapshot | null;
  slots: Map<string, number>;
};

export function useWorld() {
  const [world, setWorld] = useState<WorldState>({
    snapshot: null,
    slots: new Map(),
  });

  const [connected, setConnected] = useState(false);
  const retry = useRef(0);

  useEffect(() => {
    let disposed = false;
    let socket: WebSocket | null = null;
    let timer: number | undefined;

    const connect = () => {
      if (disposed) return;
      const scheme = location.protocol === "https:" ? "wss:" : "ws:";
      socket = new WebSocket(`${scheme}//${location.host}/ws`);

      socket.onopen = () => {
        retry.current = 0;
        setConnected(true);
      };
      socket.onmessage = (event) => {
        const snapshot: Snapshot = JSON.parse(event.data);
        setWorld((previous) => ({
          snapshot,
          slots: assignSlots(
            previous.slots,
            snapshot.players.map((player) => player.username),
          ),
        }));
      };

      socket.onerror = () => socket?.close();
      socket.onclose = () => {
        setConnected(false);
        if (disposed) return;
        const delay = Math.min(1000 * 2 ** retry.current, 15000);
        retry.current += 1;
        timer = window.setTimeout(connect, delay);
      };
    };

    connect();
    return () => {
      disposed = true;
      window.clearTimeout(timer);
      socket?.close();
    };
  }, []);

  return { snapshot: world.snapshot, slots: world.slots, connected };
}
