package com.example.tsc.forex

/**
 * Central backend address resolution.
 *
 * Release builds always talk to the production API over TLS (https/wss) and
 * expose no server-configuration UI. Debug builds may override the host at
 * runtime so the app can be tested against a local backend on the LAN.
 */
object AppConfig {
    const val PRODUCTION_HOST = "tsc-api.talhacan.com"

    @Volatile
    var hostOverride: String? = null

    val allowsServerOverride: Boolean get() = BuildConfig.DEBUG

    val host: String
        get() = if (allowsServerOverride) (hostOverride ?: PRODUCTION_HOST) else PRODUCTION_HOST

    private val secure: Boolean get() = host.contains("talhacan.com")

    val apiBaseUrl: String
        get() = "${if (secure) "https" else "http"}://$host/api/v1"

    val wsUrl: String
        get() = "${if (secure) "wss" else "ws"}://$host/ws"
}
