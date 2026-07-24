package com.example.tsc.forex.auth

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper

/** Walks ContextWrappers to find a hosting [Activity] (needed by Credential Manager). */
fun Context.findActivity(): Activity? {
    var current: Context? = this
    while (current is ContextWrapper) {
        if (current is Activity) return current
        current = current.baseContext
    }
    return null
}

fun Context.requireActivity(): Activity =
    findActivity() ?: error("Credential Manager için Activity context gerekli")
