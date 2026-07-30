package com.example.tsc.forex.ui

import android.app.Activity
import android.content.Context
import com.google.android.play.core.review.ReviewManagerFactory
import java.util.concurrent.TimeUnit

/**
 * Requests Google Play's native review flow after several meaningful actions.
 * It never asks whether the user is happy, filters users, or grants a reward.
 */
object InAppReviewCoordinator {
    private const val PREFS = "tsc_review_prompt"
    private const val FIRST_ACTION_AT = "first_meaningful_action_at"
    private const val ACTION_COUNT = "meaningful_action_count"
    private const val LAST_ATTEMPT_AT = "last_attempt_at"
    private const val LAST_ATTEMPT_VERSION = "last_attempt_version"

    private const val MINIMUM_ACTIONS = 3
    private val MINIMUM_EXPERIENCE_MS = TimeUnit.DAYS.toMillis(3)
    private val RETRY_INTERVAL_MS = TimeUnit.DAYS.toMillis(120)

    fun recordMeaningfulAction(activity: Activity) {
        val context = activity.applicationContext
        val prefs = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
        val now = System.currentTimeMillis()
        val firstAction = prefs.getLong(FIRST_ACTION_AT, 0L).let {
            if (it > 0L) it else now.also { created ->
                prefs.edit().putLong(FIRST_ACTION_AT, created).apply()
            }
        }
        val actions = prefs.getInt(ACTION_COUNT, 0) + 1
        prefs.edit().putInt(ACTION_COUNT, actions).apply()

        if (actions < MINIMUM_ACTIONS || now - firstAction < MINIMUM_EXPERIENCE_MS) return

        val version = runCatching {
            context.packageManager.getPackageInfo(context.packageName, 0).versionName
        }.getOrNull().orEmpty().ifBlank { "unknown" }
        if (prefs.getString(LAST_ATTEMPT_VERSION, null) == version) return

        val lastAttempt = prefs.getLong(LAST_ATTEMPT_AT, 0L)
        if (lastAttempt > 0L && now - lastAttempt < RETRY_INTERVAL_MS) return

        val manager = ReviewManagerFactory.create(activity)
        manager.requestReviewFlow().addOnCompleteListener { request ->
            if (!request.isSuccessful) return@addOnCompleteListener

            prefs.edit()
                .putLong(LAST_ATTEMPT_AT, now)
                .putString(LAST_ATTEMPT_VERSION, version)
                .putInt(ACTION_COUNT, 0)
                .apply()

            manager.launchReviewFlow(activity, request.result)
        }
    }
}
