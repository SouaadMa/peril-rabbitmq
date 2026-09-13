const SERIES = ["var(--series-0)", "var(--series-1)", "var(--series-2)"];
const OVERFLOW = "var(--overflow)";

export function assignSlots(
  previous: Map<string, number>,
  usernames: string[],
): Map<string, number> {
  const next = new Map<string, number>();

  for (const name of usernames) {
    const slot = previous.get(name);
    if (slot !== undefined) next.set(name, slot);
  }

  const taken = new Set(next.values());
  for (const name of [...usernames].sort()) {
    if (next.has(name)) continue;
    let slot = 0;
    while (taken.has(slot)) slot++;
    next.set(name, slot);
    taken.add(slot);
  }

  return next;
}

export function playerColors(slots: Map<string, number>) {
  const colors = new Map<string, string>();
  for (const [name, slot] of slots) {
    colors.set(name, SERIES[slot] ?? OVERFLOW);
  }
  return colors;
}
