package main

// The distroless runtime image has no timezone database, so embed Go's copy:
// APP_TIMEZONE and the client's X-Client-Timezone header must resolve in
// production exactly as they do on a developer machine.
import _ "time/tzdata"
