'use client'
import { useRouter } from 'next/navigation'
import { LogOut } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { authStorage } from '@/lib/auth'

export function Header() {
  const router = useRouter()

  const handleLogout = () => {
    authStorage.clear()
    router.replace('/login')
  }

  return (
    <header className="flex h-16 items-center justify-between border-b bg-card px-6">
      <div />
      <Button variant="ghost" size="sm" onClick={handleLogout} className="text-muted-foreground">
        <LogOut className="mr-2 h-4 w-4" />
        Гарах
      </Button>
    </header>
  )
}
