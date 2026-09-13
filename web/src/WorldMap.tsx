import type { LocationView } from "./types";

type Region = {
  path: string;
  label: { x: number; y: number };
};

const REGIONS: Record<string, Region> = {
  americas: {
    path: "M 50 65 L 145 42 L 232 68 L 220 118 L 192 145 L 172 178 L 150 200 L 128 178 L 96 150 L 68 115 Z M 150 240 L 205 232 L 230 268 L 222 318 L 198 362 L 176 388 L 162 360 L 158 312 L 138 278 Z",
    label: { x: 140, y: 108 },
  },
  europe: {
    path: "M 352 66 L 398 56 L 442 72 L 452 100 L 430 122 L 400 132 L 372 148 L 356 126 L 344 96 Z",
    label: { x: 398, y: 98 },
  },
  africa: {
    path: "M 358 170 L 418 162 L 462 178 L 470 218 L 448 262 L 424 300 L 402 340 L 382 316 L 372 272 L 356 232 L 350 198 Z",
    label: { x: 408, y: 246 },
  },
  asia: {
    path: "M 470 58 L 560 42 L 660 48 L 740 68 L 762 104 L 742 140 L 700 158 L 646 172 L 600 196 L 556 210 L 516 196 L 486 168 L 468 130 L 462 92 Z",
    label: { x: 608, y: 112 },
  },
  australia: {
    path: "M 636 288 L 700 278 L 748 296 L 754 330 L 726 356 L 676 360 L 642 340 L 630 312 Z",
    label: { x: 692, y: 318 },
  },
  antarctica: {
    path: "M 70 410 L 200 400 L 380 396 L 560 400 L 720 410 L 748 426 L 700 444 L 500 450 L 300 448 L 120 438 L 62 426 Z",
    label: { x: 400, y: 422 },
  },
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
        const region = REGIONS[location.name];
        if (!region) return null;

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
        const firstLine = region.label.y - ((lines - 1) * LINE_HEIGHT) / 2 + 5;

        return (
          <g
            key={location.name}
            className={leader ? "region filled" : "region"}
          >
            <path
              d={region.path}
              fill={fill}
              stroke="#fcfcfb"
              strokeWidth={2}
              strokeLinejoin="round"
            />
            <text className="region-name" x={region.label.x} y={firstLine}>
              {location.name}
            </text>
            {location.occupants.map((occupant, index) => (
              <text
                key={occupant.username}
                className="region-occupant"
                x={region.label.x}
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
