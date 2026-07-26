package com.example.tsc.forex.ui

import android.content.Context
import android.content.res.Configuration
import java.util.Locale
import java.util.concurrent.ConcurrentHashMap

enum class AppLanguage(val tag: String, val nativeName: String) {
    TURKISH("tr", "Türkçe"),
    ENGLISH("en", "English"),
    ARABIC("ar", "العربية"),
    GERMAN("de", "Deutsch"),
    ITALIAN("it", "Italiano"),
    SPANISH("es", "Español");

    companion object {
        fun fromTag(tag: String?): AppLanguage =
            entries.firstOrNull { it.tag == tag?.substringBefore("-") } ?: TURKISH
    }
}

object AppLocale {
    private data class TemplateTranslation(
        val regex: Regex,
        val localizedTemplate: String,
    )

    private const val PREFS = "tsc_forex"
    private const val KEY = "app_language_v1"
    private val supported = AppLanguage.entries.map { it.tag }.toSet()
    private val cache = ConcurrentHashMap<String, Map<String, String>>()
    private val templateCache = ConcurrentHashMap<String, List<TemplateTranslation>>()
    private val resolvedCache = ConcurrentHashMap<String, String>()

    fun selected(context: Context): AppLanguage {
        val stored = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getString(KEY, null)
        if (!stored.isNullOrBlank()) return AppLanguage.fromTag(stored)
        val configured = context.resources.configuration.locales
            .takeIf { !it.isEmpty }
            ?.get(0)
            ?.language
        val system = configured
            ?.takeIf(supported::contains)
            ?: Locale.getDefault().language.takeIf(supported::contains)
            ?: "tr"
        return AppLanguage.fromTag(system)
    }

    fun select(context: Context, language: AppLanguage) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putString(KEY, language.tag)
            .apply()
    }

    fun wrap(base: Context): Context {
        val language = selected(base)
        val locale = Locale.forLanguageTag(language.tag)
        Locale.setDefault(locale)
        val configuration = Configuration(base.resources.configuration)
        configuration.setLocale(locale)
        configuration.setLayoutDirection(locale)
        return base.createConfigurationContext(configuration)
    }

    fun text(context: Context, raw: String): String {
        if (raw.isBlank()) return raw
        val language = selected(context)
        if (language == AppLanguage.TURKISH) return raw
        val resolvedKey = "${language.tag}\u0000$raw"
        resolvedCache[resolvedKey]?.let { return it }
        val translations = cache.getOrPut(language.tag) { load(context, language.tag) }
        translations[raw]?.let {
            resolvedCache[resolvedKey] = it
            return it
        }

        val signInSuffix = " için giriş yapın"
        if (raw.endsWith(signInSuffix)) {
            val subject = raw.removeSuffix(signInSuffix)
            val translatedSubject = translations[subject] ?: subject
            val format = translations["%@ için giriş yapın"] ?: "Sign in for %@"
            return format.replace("%@", translatedSubject).also { resolvedCache[resolvedKey] = it }
        }

        Regex("""^(\d+)\s+hak$""").matchEntire(raw)?.let {
            val unit = when (language) {
                AppLanguage.ENGLISH -> "credits"
                AppLanguage.ARABIC -> "رصيد"
                AppLanguage.GERMAN -> "Nutzungen"
                AppLanguage.ITALIAN -> "utilizzi"
                AppLanguage.SPANISH -> "usos"
                AppLanguage.TURKISH -> "hak"
            }
            return "${it.groupValues[1]} $unit".also { resolved -> resolvedCache[resolvedKey] = resolved }
        }

        val templates = templateCache.getOrPut(language.tag) {
            translations.mapNotNull { (source, localized) ->
                compileTemplate(source)?.let { TemplateTranslation(it, localized) }
            }
        }
        templates.firstNotNullOfOrNull { template ->
            template.regex.matchEntire(raw)?.let { match ->
                renderTemplate(template.localizedTemplate, match.groupValues.drop(1))
            }
        }?.let {
            resolvedCache[resolvedKey] = it
            return it
        }
        return raw
    }

    private fun compileTemplate(source: String): Regex? {
        val pattern = StringBuilder("^")
        var index = 0
        var tokenCount = 0
        while (index < source.length) {
            if (source[index] == '%') {
                if (source.startsWith("%%", index)) {
                    pattern.append(Regex.escape("%"))
                    index += 2
                    continue
                }
                val token = formatTokenAt(source, index)
                if (token != null) {
                    pattern.append("(.+?)")
                    tokenCount += 1
                    index += token.length
                    continue
                }
            }
            pattern.append(Regex.escape(source[index].toString()))
            index += 1
        }
        if (tokenCount == 0) return null
        pattern.append("\$")
        return Regex(pattern.toString())
    }

    private fun renderTemplate(template: String, values: List<String>): String {
        val output = StringBuilder()
        var index = 0
        var valueIndex = 0
        while (index < template.length) {
            if (template[index] == '%') {
                if (template.startsWith("%%", index)) {
                    output.append('%')
                    index += 2
                    continue
                }
                val token = formatTokenAt(template, index)
                if (token != null) {
                    output.append(values.getOrElse(valueIndex) { "" })
                    valueIndex += 1
                    index += token.length
                    continue
                }
            }
            output.append(template[index])
            index += 1
        }
        return output.toString()
    }

    private fun formatTokenAt(value: String, index: Int): String? {
        val suffix = value.substring(index)
        return listOf("%lld", "%ld", "%.2f", "%.1f", "%@", "%d", "%f")
            .firstOrNull(suffix::startsWith)
    }

    private fun load(context: Context, tag: String): Map<String, String> {
        val path = "$tag.lproj/Localizable.strings"
        return runCatching {
            context.assets.open(path).bufferedReader(Charsets.UTF_8).useLines { lines ->
                lines.mapNotNull(::parseLine).toMap()
            }
        }.getOrDefault(emptyMap())
    }

    private fun parseLine(line: String): Pair<String, String>? {
        val trimmed = line.trim()
        if (!trimmed.startsWith("\"") || !trimmed.endsWith(";")) return null
        val separator = trimmed.indexOf("\" = \"")
        if (separator <= 0) return null
        val key = trimmed.substring(1, separator)
        val valueStart = separator + 5
        val valueEnd = trimmed.lastIndexOf("\";")
        if (valueEnd < valueStart) return null
        return unescape(key) to unescape(trimmed.substring(valueStart, valueEnd))
    }

    private fun unescape(value: String): String =
        value.replace("\\\"", "\"").replace("\\n", "\n").replace("\\\\", "\\")
}
