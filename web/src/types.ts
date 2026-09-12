export type UnitView = { id: number; rank: string; location: string };
export type PlayerView = {
  username: string;
  units: UnitView[];
  lastSeen: string;
};
export type Occupant = { username: string; units: number; power: number };
export type LocationView = { name: string; occupants: Occupant[] };
export type LogEntry = { time: string; username: string; message: string };

export type Snapshot = {
  players: PlayerView[];
  locations: LocationView[];
  log: LogEntry[];
};
