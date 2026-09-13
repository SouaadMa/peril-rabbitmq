import type { LocationView } from "./types";

import { CONTINENTS, project, toPath, type Point } from "./geography";

const SHAPES: Record<string, { path: string; label: Point }> = {};
for (const [name, continent] of Object.entries(CONTINENTS)) {
  SHAPES[name] = {
    path: toPath(continent.rings),
    label: project(continent.label),
  };
}

const MERIDIANS = Array.from({ length: 11 }, (_, i) => -150 + i * 30);
const PARALLELS = [-60, -30, 0, 30, 60];

const LINE_HEIGHT = 17;

type Props = {
  locations: LocationView[];
  colors: Map<string, string>;
};

export function WorldMap({ locations, colors }: Props) {
  return (
    <svg className="map" viewBox="0 0 800 460">
      <g className="graticule">
        {MERIDIANS.map((lon) => {
          const [x] = project([lon, 0]);
          return <line key={`lon${lon}`} x1={x} y1={0} x2={x} y2={460} />;
        })}
        {PARALLELS.map((lat) => {
          const [, y] = project([0, lat]);
          return <line key={`lat${lat}`} x1={0} y1={y} x2={800} y2={y} />;
        })}
      </g>

      {locations.map((location) => {
        const shape = SHAPES[location.name];
        if (!shape) return null;

        const [labelX, labelY] = shape.label;

        let leader = null;
        for (const occupant of location.occupants) {
          if (leader === null || occupant.power > leader.power) {
            leader = occupant;
          }
        }

        const fill = leader ? colors.get(leader.username) : undefined;

        const lines = 1 + location.occupants.length;
        const firstLine = labelY - ((lines - 1) * LINE_HEIGHT) / 2 + 5;

        return (
          <g
            key={location.name}
            className={leader ? "region filled" : "region"}
          >
            <path
              d={shape.path}
              style={leader ? { fill } : undefined}
              strokeWidth={1}
              strokeLinejoin="round"
            />

            <text className="region-name" x={labelX} y={firstLine}>
              {location.name}
            </text>
            {location.occupants.map((occupant, index) => (
              <text
                key={occupant.username}
                className="region-occupant"
                x={labelX}
                y={firstLine + (index + 1) * LINE_HEIGHT}
              >
                {occupant.username} ×{occupant.units} ({occupant.power})
              </text>
            ))}
          </g>
        );
      })}
    </svg>
  );
}
