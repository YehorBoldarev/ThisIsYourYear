/*
 * Date helpers bridging three representations:
 *  - API form:   "DD-MM-YYYY" (the contract's calendar-date shape)
 *  - input form: "YYYY-MM-DD" (native <input type="date"> value)
 *  - display:    "Oct 4"       (short due-date label in the checklist)
 * Empty strings pass through as empty ("unset") in every direction.
 */

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

const API_RE = /^(\d{2})-(\d{2})-(\d{4})$/
const INPUT_RE = /^(\d{4})-(\d{2})-(\d{2})$/

/** "YYYY-MM-DD" (date input) → "DD-MM-YYYY" (API). Empty/invalid → "". */
export function toApiDate(input: string): string {
  const m = INPUT_RE.exec(input.trim())
  if (!m) return ''
  const [, y, mo, d] = m
  return `${d}-${mo}-${y}`
}

/** "DD-MM-YYYY" (API) → "YYYY-MM-DD" (date input). Empty/invalid → "". */
export function toInputDate(api: string): string {
  const m = API_RE.exec(api.trim())
  if (!m) return ''
  const [, d, mo, y] = m
  return `${y}-${mo}-${d}`
}

/** "DD-MM-YYYY" → "Oct 4". Empty/invalid → "". Month names avoid any tz math. */
export function formatDueShort(api: string): string {
  const m = API_RE.exec(api.trim())
  if (!m) return ''
  const [, d, mo] = m
  const month = MONTHS[Number(mo) - 1]
  if (!month) return ''
  return `${month} ${Number(d)}`
}

const ISO_DATE_RE = /^(\d{4})-(\d{2})-(\d{2})/

/** ISO date-time (e.g. `completedAt`) → "Oct 4, 2026". Empty/invalid → "".
 *  Reads the calendar-date portion literally, so no timezone drift. */
export function formatIsoDate(iso: string | undefined): string {
  if (!iso) return ''
  const m = ISO_DATE_RE.exec(iso)
  if (!m) return ''
  const [, y, mo, d] = m
  const month = MONTHS[Number(mo) - 1]
  if (!month) return ''
  return `${month} ${Number(d)}, ${y}`
}
