import type { LocationView } from "./types";

const REGIONS: Record<string, { x: number; y: number; w: number; h: number }> =
  {
    americas: { x: 60, y: 70, w: 140, h: 310 },
    europe: { x: 330, y: 70, w: 110, h: 90 },
    africa: { x: 330, y: 180, w: 120, h: 180 },
    asia: { x: 470, y: 60, w: 250, h: 170 },
    australia: { x: 600, y: 270, w: 120, h: 90 },
    antarctica: { x: 100, y: 400, w: 620, h: 45 },
  };

const LINE_HEIGHT = 17;

type Props = {
  locations: LocationView[];
  colors: Map<string, string>;
};

export function WorldMap({ locations, colors }: Props) {
  return (
    <svg className="map" viewBox="0 0 800 460">
      {locations.map((location) => {
        const box = REGIONS[location.name];
        if (!box) return null;

        let leader = null;
        for (const occupant of location.occupants) {
          if (leader === null || occupant.power > leader.power) {
            leader = occupant;
          }
        }

        const fill = leader
          ? (colors.get(leader.username) ?? "#6b6a66")
          : "#e2e1dc";

        const lines = 1 + location.occupants.length;
        const firstLine =
          box.y + box.h / 2 - ((lines - 1) * LINE_HEIGHT) / 2 + 5;

        return (
          <g
            key={location.name}
            className={leader ? "region filled" : "region"}
          >
            <rect
              x={box.x}
              y={box.y}
              width={box.w}
              height={box.h}
              rx={8}
              fill={fill}
              stroke="#fcfcfb"
              strokeWidth={2}
            />
            <text className="region-name" x={box.x + box.w / 2} y={firstLine}>
              {location.name}
            </text>
            {location.occupants.map((occupant, index) => (
              <text
                key={occupant.username}
                className="region-occupant"
                x={box.x + box.w / 2}
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
