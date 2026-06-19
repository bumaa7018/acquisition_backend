'use client'
import { useEffect, useRef, useState } from 'react'
import Map from 'ol/Map'
import View from 'ol/View'
import TileLayer from 'ol/layer/Tile'
import ImageLayer from 'ol/layer/Image'
import OSM from 'ol/source/OSM'
import ImageWMS from 'ol/source/ImageWMS'
import { fromLonLat } from 'ol/proj'
import 'ol/ol.css'
import { LayerPanel, DEFAULT_LAYERS, type GeoLayer } from './layer-panel'
import { FeaturePopup } from './feature-popup'
// eslint-disable-next-line @typescript-eslint/no-explicit-any

const GS_URL = process.env.NEXT_PUBLIC_GEOSERVER_URL ?? 'http://localhost:8600/geoserver'
const WMS_URL = `${GS_URL}/land/wms`

export default function MapView() {
  const mapRef = useRef<HTMLDivElement>(null)
  const mapInstance = useRef<Map | null>(null)
  const wmsLayers = useRef<Record<string, ImageLayer<ImageWMS>>>({})
  const [layers, setLayers] = useState<GeoLayer[]>(DEFAULT_LAYERS)
  const [popup, setPopup] = useState<{ features: Record<string, string>[]; x: number; y: number } | null>(null)

  useEffect(() => {
    if (!mapRef.current || mapInstance.current) return

    const osmLayer = new TileLayer({ source: new OSM() })

    const wmsLayerInstances: Record<string, ImageLayer<ImageWMS>> = {}
    DEFAULT_LAYERS.forEach(layer => {
      const wmsLayer = new ImageLayer({
        source: new ImageWMS({
          url: WMS_URL,
          params: { LAYERS: layer.name, STYLES: '', FORMAT: 'image/png', TRANSPARENT: true },
          ratio: 1,
          serverType: 'geoserver',
        }),
        visible: layer.visible,
        opacity: 0.8,
      })
      wmsLayerInstances[layer.id] = wmsLayer
    })
    wmsLayers.current = wmsLayerInstances

    const map = new Map({
      target: mapRef.current,
      layers: [osmLayer, ...Object.values(wmsLayerInstances)],
      view: new View({
        center: fromLonLat([106.9057, 47.8864]),
        zoom: 7,
        minZoom: 5,
        maxZoom: 18,
      }),
    })

    const handleClick = async (evtRaw: unknown) => {
      const evt = evtRaw as { coordinate: number[] }
      const visibleLayers = DEFAULT_LAYERS.filter(l => {
        const inst = wmsLayerInstances[l.id]
        return inst?.getVisible()
      }).map(l => l.name).join(',')

      if (!visibleLayers) return

      const view = map.getView()
      const resolution = view.getResolution() ?? 1
      const projection = view.getProjection()

      const anySource = Object.values(wmsLayerInstances)[0]?.getSource()
      if (!anySource) return

      const url = anySource.getFeatureInfoUrl(
        evt.coordinate,
        resolution,
        projection,
        {
          INFO_FORMAT: 'application/json',
          QUERY_LAYERS: visibleLayers,
          LAYERS: visibleLayers,
          FEATURE_COUNT: 10,
        },
      )
      if (!url) return

      try {
        const res = await fetch(url)
        if (!res.ok) return
        const json = await res.json()
        const feats = (json.features ?? []).map((f: { properties: Record<string, string> }) => f.properties)
        if (feats.length > 0) {
          const pixel = map.getPixelFromCoordinate(evt.coordinate)
          setPopup({ features: feats, x: pixel[0], y: pixel[1] })
        } else {
          setPopup(null)
        }
      } catch { /* ignore fetch errors */ }
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(map as any).on('singleclick', handleClick)

    mapInstance.current = map

    return () => {
      map.setTarget(undefined)
      mapInstance.current = null
    }
  }, [])

  const handleToggle = (id: string) => {
    setLayers(prev => prev.map(l => {
      if (l.id !== id) return l
      const updated = { ...l, visible: !l.visible }
      const inst = wmsLayers.current[id]
      if (inst) inst.setVisible(updated.visible)
      return updated
    }))
    setPopup(null)
  }

  return (
    <div className="relative w-full h-full rounded-lg overflow-hidden border shadow-sm">
      <div ref={mapRef} className="w-full h-full" />
      <LayerPanel layers={layers} onToggle={handleToggle} />
      {popup && (
        <FeaturePopup
          features={popup.features}
          x={popup.x}
          y={popup.y}
          onClose={() => setPopup(null)}
        />
      )}
    </div>
  )
}
