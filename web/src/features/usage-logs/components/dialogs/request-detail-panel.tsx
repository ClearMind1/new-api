/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { Alert02Icon, Copy01Icon, Tick02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { cn } from '@/lib/utils'

import { getLogDetail } from '../../api'

type RequestDetailPanelProps = {
  enabled: boolean
  requestId: string
}

type RequestPayloadBlockProps = {
  content: string
  copied: boolean
  onCopy: (content: string) => void
}

function formatPayload(content: string): string {
  if (!content) return ''

  try {
    return JSON.stringify(JSON.parse(content), null, 2)
  } catch {
    return content
  }
}

function RequestPayloadBlock({
  content,
  copied,
  onCopy,
}: RequestPayloadBlockProps) {
  const { t } = useTranslation()

  if (!content) {
    return (
      <Empty className='min-h-48 border'>
        <EmptyHeader>
          <EmptyTitle>{t('No data available')}</EmptyTitle>
        </EmptyHeader>
      </Empty>
    )
  }

  return (
    <div className='bg-muted/30 relative min-w-0 overflow-hidden rounded-lg border'>
      <Button
        type='button'
        variant='ghost'
        size='sm'
        className='absolute top-2 right-2'
        onClick={() => onCopy(content)}
        aria-label={copied ? t('Copied') : t('Copy to clipboard')}
        title={copied ? t('Copied') : t('Copy to clipboard')}
      >
        <HugeiconsIcon
          icon={copied ? Tick02Icon : Copy01Icon}
          data-icon='inline-start'
          aria-hidden='true'
        />
        {copied ? t('Copied') : t('Copy')}
      </Button>
      <pre className='max-h-[26rem] min-h-48 overflow-auto p-4 pt-12 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap sm:break-words'>
        {content}
      </pre>
    </div>
  )
}

function RequestMetadataItem({
  label,
  children,
  mono = false,
}: {
  label: string
  children: React.ReactNode
  mono?: boolean
}) {
  return (
    <div className='flex min-w-0 flex-col gap-1 rounded-lg border p-3'>
      <span className='text-muted-foreground text-xs'>{label}</span>
      <div
        className={cn(
          'min-w-0',
          mono ? 'font-mono text-xs break-all' : 'text-sm'
        )}
      >
        {children}
      </div>
    </div>
  )
}

export function RequestDetailPanel({
  enabled,
  requestId,
}: RequestDetailPanelProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const detailQuery = useQuery({
    queryKey: ['usage-logs', 'request-detail', requestId],
    queryFn: async () => {
      const response = await getLogDetail(requestId)
      if (!response.success || !response.data) {
        throw new Error(response.message || '')
      }
      return response.data
    },
    enabled: enabled && requestId.length > 0,
    retry: false,
    staleTime: 60_000,
  })

  if (detailQuery.isPending) {
    return (
      <div
        className='text-muted-foreground flex min-h-64 items-center justify-center gap-2 text-sm'
        role='status'
      >
        <Spinner />
        <span>{t('Loading...')}</span>
      </div>
    )
  }

  if (detailQuery.isError) {
    const message =
      detailQuery.error instanceof Error && detailQuery.error.message
        ? detailQuery.error.message
        : t('Loading failed')

    return (
      <Empty className='min-h-64 border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <HugeiconsIcon icon={Alert02Icon} aria-hidden='true' />
          </EmptyMedia>
          <EmptyTitle>{t('Loading failed')}</EmptyTitle>
          <EmptyDescription>{message}</EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button
            type='button'
            variant='outline'
            onClick={() => void detailQuery.refetch()}
          >
            {t('Retry')}
          </Button>
        </EmptyContent>
      </Empty>
    )
  }

  const detail = detailQuery.data
  const requestBody = formatPayload(detail.request_body)
  const requestHeaders = formatPayload(detail.request_headers)
  const responseBody = formatPayload(detail.response_body)

  const handleCopy = (content: string) => {
    void copyToClipboard(content)
  }

  return (
    <div className='flex min-w-0 flex-col gap-4'>
      <div className='grid gap-3 sm:grid-cols-2'>
        <RequestMetadataItem label={t('Request ID')} mono>
          {detail.request_id || requestId}
        </RequestMetadataItem>
        <RequestMetadataItem label={t('Model')} mono>
          {detail.model_name || '—'}
        </RequestMetadataItem>
        <RequestMetadataItem label={t('Path')} mono>
          <div className='flex min-w-0 items-start gap-2'>
            {detail.request_method && (
              <Badge variant='outline' className='shrink-0 font-mono'>
                {detail.request_method}
              </Badge>
            )}
            <span className='min-w-0 break-all'>
              {detail.request_path || '—'}
            </span>
          </div>
        </RequestMetadataItem>
        <RequestMetadataItem label={t('Status Code')} mono>
          {detail.status_code > 0 ? detail.status_code : '—'}
        </RequestMetadataItem>
      </div>

      <Tabs defaultValue='request' className='min-w-0 gap-3'>
        <TabsList className='grid w-full grid-cols-3'>
          <TabsTrigger value='request'>{t('Request Body')}</TabsTrigger>
          <TabsTrigger value='headers'>{t('Request Headers')}</TabsTrigger>
          <TabsTrigger value='response'>{t('Response Body')}</TabsTrigger>
        </TabsList>
        <TabsContent value='request'>
          <RequestPayloadBlock
            content={requestBody}
            copied={copiedText === requestBody}
            onCopy={handleCopy}
          />
        </TabsContent>
        <TabsContent value='headers'>
          <RequestPayloadBlock
            content={requestHeaders}
            copied={copiedText === requestHeaders}
            onCopy={handleCopy}
          />
        </TabsContent>
        <TabsContent value='response'>
          <RequestPayloadBlock
            content={responseBody}
            copied={copiedText === responseBody}
            onCopy={handleCopy}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}
