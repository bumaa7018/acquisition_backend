'use client'
import { X } from 'lucide-react'
import { ScrollArea } from '@/components/ui/scroll-area'

interface FeaturePopupProps {
  features: Record<string, string>[]
  x: number
  y: number
  onClose: () => void
}

const FIELD_LABELS: Record<string, string> = {
  id: 'ID',
  plan_code: 'Төлөвлөгөөний дугаар',
  status: 'Статус',
  start_date: 'Эхлэх огноо',
  end_date: 'Дуусах огноо',
  area_m2: 'Талбай (м²)',
  plan_area_m2: 'Төлөвлөгөөний талбай',
  parcel_id: 'Нэгж талбарын дугаар',
  au1_code: 'Аймаг',
  au2_code: 'Сум',
  au3_code: 'Баг',
  right_type: 'Эрхийн төрөл',
  landuse: 'Зориулалт',
  acquisition_area_m2: 'Давхцах талбай',
  code: 'Код',
  name: 'Нэр',
  acquisition_id: 'Чөлөөлөлтийн ID',
}

export function FeaturePopup({ features, x, y, onClose }: FeaturePopupProps) {
  const style: React.CSSProperties = {
    position: 'absolute',
    left: x + 12,
    top: y - 8,
    transform: 'translateY(-50%)',
    zIndex: 20,
    maxWidth: 320,
  }

  // Adjust if too close to right edge
  if (x > 500) {
    style.left = undefined
    style.right = `calc(100% - ${x}px + 12px)`
  }

  return (
    <div style={style} className="bg-background/97 backdrop-blur border rounded-xl shadow-2xl overflow-hidden">
      <div className="flex items-center justify-between px-4 py-2.5 border-b bg-muted/50">
        <span className="text-sm font-semibold">{features.length > 1 ? `${features.length} давхарга` : 'Давхаргын мэдээлэл'}</span>
        <button onClick={onClose} className="rounded-md p-1 hover:bg-muted transition-colors">
          <X className="h-3.5 w-3.5" />
        </button>
      </div>
      <ScrollArea className="max-h-72">
        {features.map((feat, i) => (
          <div key={i} className={`px-4 py-3 ${i > 0 ? 'border-t' : ''}`}>
            {Object.entries(feat)
              .filter(([, v]) => v !== null && v !== '' && v !== 'null')
              .map(([k, v]) => (
                <div key={k} className="flex items-start gap-2 py-0.5 text-xs">
                  <span className="text-muted-foreground shrink-0 w-28">{FIELD_LABELS[k] ?? k}</span>
                  <span className="font-medium break-all">{String(v)}</span>
                </div>
              ))
            }
          </div>
        ))}
      </ScrollArea>
    </div>
  )
}
