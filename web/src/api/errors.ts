/*
 * The backend's stable error envelope: { "error": { code, message, fields? } }.
 * ApiError normalizes it so UI can map `fields` to inline form errors and show
 * `message` at the form level.
 */
export type FieldErrors = Record<string, string>

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly fields?: FieldErrors

  constructor(status: number, code: string, message: string, fields?: FieldErrors) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }

  /**
   * Build an ApiError from an already-parsed envelope body (openapi-fetch reads
   * the response body into `error` before we see it, so we can't re-read it).
   */
  static fromBody(status: number, body: unknown): ApiError {
    const envelope = body as
      { error?: { code?: string; message?: string; fields?: FieldErrors } } | undefined
    const e = envelope?.error
    return new ApiError(status, e?.code ?? 'error', e?.message ?? 'Request failed', e?.fields)
  }

  /** Build an ApiError from a non-OK Response, tolerating a missing/odd body. */
  static async fromResponse(res: Response): Promise<ApiError> {
    let code = 'error'
    let message = res.statusText || 'Request failed'
    let fields: FieldErrors | undefined
    try {
      const body = (await res.json()) as {
        error?: { code?: string; message?: string; fields?: FieldErrors }
      }
      if (body?.error) {
        code = body.error.code ?? code
        message = body.error.message ?? message
        fields = body.error.fields
      }
    } catch {
      // Non-JSON or empty body — keep the status-derived defaults.
    }
    return new ApiError(res.status, code, message, fields)
  }
}

/** Narrow an unknown thrown value to ApiError. */
export function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError
}
