'use client'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Layers, ChevronDown, ChevronUp } from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/lib/utils'

export interface GeoLayer {
  id: string
  name: string
  label: string
  group: string
  color: string
  visible: boolean
}

export const DEFAULT_LAYERS: GeoLayer[] = [
  { id: 'au1', name: 'land:au1', label: 'Аймаг', group: 'Захиргааны нэгж', color: '#6366f1', visible: true },
  { id: 'au2', name: 'land:au2', label: 'Сум', group: 'Захиргааны нэгж', color: '#8b5cf6', visible: true },
  { id: 'au3', name: 'land:au3', label: 'Баг', group: 'Захиргааны нэгж', color: '#a78bfa', visible: false },
  { id: 'v_acquisition_boundary', name: 'land:v_acquisition_boundary', label: 'Чөлөөлөлтийн хил', group: 'Газар чөлөөлөлт', color: '#ef4444', visible: true },
  { id: 'v_acquisition_plan', name: 'land:v_acquisition_plan', label: 'Төлөвлөгөөний хил', group: 'Газар чөлөөлөлт', color: '#f97316', visible: false },
  { id: 'parcel', name: 'land:parcel', label: 'Нэгж талбарын хил', group: 'Нэгж талбар', color: '#22c55e', visible: true },
  { id: 'v_parcel_acquisition', name: 'land:v_parcel_acquisition', label: 'Давхцах хэсэг', group: 'Нэгж талбар', color: '#14b8a6', visible: false },
]

interface LayerPanelProps {
  layers: GeoLayer[]
  onToggle: (id: string) => void
}

export function LayerPanel({ layers, onToggle }: LayerPanelProps) {
  const [collapsed, setCollapsed] = useState(false)

  const groups = layers.reduce((acc, l) => {
    if (!acc[l.group]) acc[l.group] = []
    acc[l.group].push(l)
    return acc
  }, {} as Record<string, GeoLayer[]>)

  return (
    <div className={cn(
      'absolute top-4 right-4 z-10 bg-background/95 backdrop-blur rounded-xl shadow-2xl border transition-all duration-200',
      collapsed ? 'w-12' : 'w-72'
    )}>
      <button
        className="flex items-center justify-between w-full px-4 py-3 font-semibold text-sm"
        onClick={() => setCollapsed(v => !v)}
      >
        {!collapsed && <span className="flex items-center gap-2"><Layers className="h-4 w-4 text-primary" />Давхаргын тохиргоо</span>}
        {collapsed && <Layers className="h-4 w-4 text-primary mx-auto" />}
        {!collapsed && (collapsed ? <ChevronDown className="h-4 w-4" /> : <ChevronUp className="h-4 w-4" />)}
      </button>

      {!collapsed && (
        <ScrollArea className="max-h-[60vh]">
          <div className="px-3 pb-3 space-y-4">
            {Object.entries(groups).map(([group, groupLayers]) => (
              <div key={group}>
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2 px-1">{group}</p>
                <div className="space-y-1">
                  {groupLayers.map(layer => (
                    <div
                      key={layer.id}
                      className="flex items-center justify-between px-3 py-2.5 rounded-lg hover:bg-muted/50 transition-colors"
                    >
                      <div className="flex items-center gap-2.5">
                        <span
                          className="block h-3 w-3 rounded-full border-2 border-white shadow-sm shrink-0"
                          style={{ backgroundColor: layer.color }}
                        />
                        <span className={cn('text-sm', !layer.visible && 'text-muted-foreground')}>{layer.label}</span>
                      </div>
                      <Switch
                        checked={layer.visible}
                        onCheckedChange={() => onToggle(layer.id)}
                        className="scale-90"
                      />
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  )
}
