const SERIES_LIGHT = ["#2a78d6", "#eb6834", "#1baf7a"];
const SERIES_DARK = ["#3987e5", "#d95926", "#199e70"];
const OVERFLOW_LIGHT = "#6b6a66";
const OVERFLOW_DARK = "#8a8984";

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

export function playerColors(slots: Map<string, number>, dark: boolean) {
  const series = dark ? SERIES_DARK : SERIES_LIGHT;
  const overflow = dark ? OVERFLOW_DARK : OVERFLOW_LIGHT;
  const colors = new Map<string, string>();
  for (const [name, slot] of slots) {
    colors.set(name, series[slot] ?? overflow);
  }
  return colors;
}
