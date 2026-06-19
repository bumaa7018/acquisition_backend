'use client'
import dynamic from 'next/dynamic'
import { Skeleton } from '@/components/ui/skeleton'

const MapView = dynamic(() => import('@/components/map/map-view'), {
  ssr: false,
  loading: () => (
    <div className="w-full h-full flex items-center justify-center">
      <div className="text-center space-y-3">
        <Skeleton className="h-8 w-8 mx-auto rounded-full" />
        <p className="text-sm text-muted-foreground">Зураг ачааллаж байна...</p>
      </div>
    </div>
  ),
})

export default function MapPage() {
  return (
    <div className="h-[calc(100vh-7rem)] -m-6">
      <MapView />
    </div>
  )
}
