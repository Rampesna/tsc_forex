package com.example.tsc.forex.data

import java.util.UUID

/** Stable per-install device id used by billing/AI quota paths. */
object DeviceIdentity {
    fun id(prefs: Prefs): String {
        val existing = prefs.deviceId
        if (!existing.isNullOrBlank()) return existing
        val created = UUID.randomUUID().toString()
        prefs.deviceId = created
        return created
    }
}
