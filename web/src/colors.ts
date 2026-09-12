const SERIES_LIGHT = ["#2a78d6", "#eb6834", "#1baf7a"];
const SERIES_DARK = ["#3987e5", "#d95926", "#199e70"];
const OVERFLOW_LIGHT = "#6b6a66";
const OVERFLOW_DARK = "#8a8984";

export function playerColors(players: string[], dark: boolean) {
  const series = dark ? SERIES_DARK : SERIES_LIGHT;
  const overflow = dark ? OVERFLOW_DARK : OVERFLOW_LIGHT;
  const sorted = [...players].sort();
  const map = new Map<string, string>();
  sorted.forEach((name, i) => map.set(name, series[i] ?? overflow));
  return map;
}
