'use client'
import { useQuery } from '@tanstack/react-query'
import { landApi, usersApi } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { FileText, Users, MapPin, TrendingUp } from 'lucide-react'
import { STATUS_LABELS, STATUS_COLORS } from '@/types'
import { formatArea } from '@/lib/utils'
import Link from 'next/link'

export default function DashboardPage() {
  const { data: landData, isLoading: landLoading } = useQuery({
    queryKey: ['land-dashboard'],
    queryFn: () => landApi.list({ page: 1, page_size: 5 }),
  })
  const { data: usersData, isLoading: usersLoading } = useQuery({
    queryKey: ['users-dashboard'],
    queryFn: () => usersApi.list({ page: 1, page_size: 1 }),
  })

  const totalLand = landData?.total ?? 0
  const totalUsers = usersData?.total ?? 0

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Хяналтын самбар</h2>
        <p className="text-muted-foreground">Газрын мэдээллийн системийн үзүүлэлтүүд</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[
          { title: 'Нийт чөлөөлөлт', value: landLoading ? null : totalLand, icon: FileText, color: 'text-blue-600', bg: 'bg-blue-50' },
          { title: 'Хэрэглэгч', value: usersLoading ? null : totalUsers, icon: Users, color: 'text-green-600', bg: 'bg-green-50' },
          { title: 'Идэвхтэй', value: landLoading ? null : landData?.data.filter(l => l.status === 2).length ?? 0, icon: TrendingUp, color: 'text-emerald-600', bg: 'bg-emerald-50' },
          { title: 'Нэгж талбар', value: landLoading ? null : landData?.data.reduce((s, l) => s + (l.parcels?.length ?? 0), 0) ?? 0, icon: MapPin, color: 'text-purple-600', bg: 'bg-purple-50' },
        ].map(({ title, value, icon: Icon, color, bg }) => (
          <Card key={title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
              <div className={`p-2 rounded-lg ${bg}`}>
                <Icon className={`h-4 w-4 ${color}`} />
              </div>
            </CardHeader>
            <CardContent>
              {value === null ? <Skeleton className="h-8 w-16" /> : (
                <div className="text-3xl font-bold">{value}</div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Сүүлийн чөлөөлөлтүүд</CardTitle>
          <Link href="/land" className="text-sm text-primary hover:underline">Бүгдийг харах →</Link>
        </CardHeader>
        <CardContent>
          {landLoading ? (
            <div className="space-y-3">{[...Array(3)].map((_, i) => <Skeleton key={i} className="h-14 w-full" />)}</div>
          ) : (
            <div className="space-y-2">
              {landData?.data.map(land => (
                <Link key={land.id} href={`/land/${land.id}`} className="flex items-center justify-between rounded-lg border p-4 hover:bg-muted/50 transition-colors">
                  <div>
                    <p className="font-medium">{land.plan_code}</p>
                    <p className="text-sm text-muted-foreground">{formatArea(land.area_m2)}</p>
                  </div>
                  <Badge className={STATUS_COLORS[land.status]}>{STATUS_LABELS[land.status]}</Badge>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
