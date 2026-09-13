import { useState } from "react";
import { CONTINENTS, project, toPath, type Point } from "./geography";
import type { LocationView } from "./types";

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
  const [hovered, setHovered] = useState<string | null>(null);
  const [pointer, setPointer] = useState({ x: 0, y: 0, width: 0 });
  const flip = pointer.x > pointer.width - 220;
  const hoveredLocation = locations.find(
    (location) => location.name === hovered,
  );

  return (
    <div
      className="map-wrap"
      onPointerMove={(event) => {
        const box = event.currentTarget.getBoundingClientRect();
        setPointer({
          x: event.clientX - box.left,
          y: event.clientY - box.top,
          width: box.width,
        });
      }}
    >
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
              className={`region${leader ? " filled" : ""}${hovered === location.name ? " hovered" : ""}`}
              onPointerEnter={() => setHovered(location.name)}
              onPointerLeave={() => setHovered(null)}
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
                  {occupant.username}
                </text>
              ))}
            </g>
          );
        })}
      </svg>
      {hoveredLocation && (
        <div
          className="tooltip"
          style={{
            left: pointer.x,
            top: pointer.y,
            transform: flip ? "translate(calc(-100% - 12px), 12px)" : undefined,
          }}
        >
          <strong className="tooltip-title">{hoveredLocation.name}</strong>
          {hoveredLocation.occupants.length === 0 ? (
            <p className="empty">Unclaimed</p>
          ) : (
            <ul className="players">
              {[...hoveredLocation.occupants]
                .sort((a, b) => b.power - a.power)
                .map((occupant) => (
                  <li key={occupant.username}>
                    <span
                      className="swatch"
                      style={{ background: colors.get(occupant.username) }}
                    />
                    <span>{occupant.username}</span>
                    <span className="count">
                      {occupant.units} units · {occupant.power} power
                    </span>
                  </li>
                ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
