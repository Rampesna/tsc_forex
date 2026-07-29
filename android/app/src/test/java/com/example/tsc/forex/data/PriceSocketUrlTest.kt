package com.example.tsc.forex.data

import org.junit.Assert.assertEquals
import org.junit.Test

class PriceSocketUrlTest {
    @Test
    fun secureWebSocketUrlIsNormalizedForOkHttp() {
        val url = websocketHttpUrl("wss://tsc-api.talhacan.com/ws")
            .newBuilder()
            .addQueryParameter("platform", "android")
            .addQueryParameter("locale", "tr-TR")
            .build()

        assertEquals("https", url.scheme)
        assertEquals("tsc-api.talhacan.com", url.host)
        assertEquals("/ws", url.encodedPath)
        assertEquals("android", url.queryParameter("platform"))
        assertEquals("tr-TR", url.queryParameter("locale"))
    }

    @Test
    fun localWebSocketUrlIsNormalizedForOkHttp() {
        val url = websocketHttpUrl("ws://10.0.2.2:8080/ws")

        assertEquals("http", url.scheme)
        assertEquals("10.0.2.2", url.host)
        assertEquals(8080, url.port)
        assertEquals("/ws", url.encodedPath)
    }
}
