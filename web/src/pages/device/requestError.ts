type ErrorPayload = {
    error?: unknown
    message?: unknown
    response?: {
        data?: ErrorPayload | string
    }
}

const readableText = (value: unknown): string => {
    return typeof value === 'string' && value.trim() ? value.trim() : ''
}

export const requestErrorMessage = (error: unknown, fallback: string): string => {
    if (typeof error === 'string') return readableText(error) || fallback
    if (!error || typeof error !== 'object') return fallback

    const payload = error as ErrorPayload
    const responseData = payload.response?.data
    if (typeof responseData === 'string') return readableText(responseData) || fallback

    return readableText(responseData?.error)
        || readableText(responseData?.message)
        || readableText(payload.error)
        || readableText(payload.message)
        || fallback
}
