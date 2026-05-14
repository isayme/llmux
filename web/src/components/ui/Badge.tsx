import { cn } from '@/lib/utils'
import type { HTMLAttributes, ReactNode } from 'react'

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: 'default' | 'success' | 'warning' | 'destructive' | 'outline'
  children: ReactNode
}

export function Badge({ variant = 'default', className, children, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors',
        {
          'bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))]': variant === 'default',
          'bg-[hsl(var(--success))] text-[hsl(var(--success-foreground))]': variant === 'success',
          'bg-[hsl(var(--warning))] text-[hsl(var(--warning-foreground))]': variant === 'warning',
          'bg-[hsl(var(--destructive))] text-[hsl(var(--destructive-foreground))]':
            variant === 'destructive',
          'border border-[hsl(var(--border))] text-[hsl(var(--foreground))]': variant === 'outline',
        },
        className
      )}
      {...props}
    >
      {children}
    </span>
  )
}
