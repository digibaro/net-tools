/** Equirectangular world map, 2:1; the view stops at 60° S (Antarctica has no probes). */
export const MAP_W = 960
export const MAP_H = 480
export const MAP_VIEW_H = 400

/** Longitude / latitude to map coordinates (same projection as geoEquirectangular fitted to the sphere). */
export function project(lon: number, lat: number): [number, number] {
  return [(lon + 180) / 360 * MAP_W, (90 - lat) / 180 * MAP_H]
}

/** Probes without a position report 0,0 ("null island"). */
export const hasPosition = (p: { latitude: number, longitude: number }) => p.latitude !== 0 || p.longitude !== 0

let land: Promise<string> | undefined

/** SVG path of the land masses (Natural Earth 1:110m, bundled; loaded on first use). */
export function landPath(): Promise<string> {
  land ??= Promise.all([import('d3-geo'), import('topojson-client'), import('world-atlas/land-110m.json')])
    .then(([{ geoEquirectangular, geoPath }, { feature }, json]) => {
      type Topology = Parameters<typeof feature>[0]
      const topo = (json.default ?? json) as unknown as Topology
      const projection = geoEquirectangular().fitSize([MAP_W, MAP_H], { type: 'Sphere' })
      return geoPath(projection)(feature(topo, topo.objects.land!)) ?? ''
    })
  return land
}
