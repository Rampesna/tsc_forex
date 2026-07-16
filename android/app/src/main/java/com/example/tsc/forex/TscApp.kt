package com.example.tsc.forex

import android.app.Application
import com.example.tsc.forex.data.Prefs
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PortfolioStore
import com.example.tsc.forex.store.PriceStore

/** App-wide dependency container (simple manual DI). */
class TscApp : Application() {
    lateinit var prefs: Prefs
        private set
    lateinit var auth: AuthStore
        private set
    lateinit var prices: PriceStore
        private set
    lateinit var portfolio: PortfolioStore
        private set

    override fun onCreate() {
        super.onCreate()
        prefs = Prefs(this)
        if (AppConfig.allowsServerOverride) AppConfig.hostOverride = prefs.hostOverride
        auth = AuthStore(prefs)
        prices = PriceStore(prefs, auth)
        portfolio = PortfolioStore(prefs, auth)
    }
}
