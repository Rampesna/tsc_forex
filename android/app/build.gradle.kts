import java.util.Properties
import org.gradle.api.tasks.Sync

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

val generatedLocalizationAssets = layout.buildDirectory.dir("generated/assets/localizations").get().asFile
val syncSharedLocalizations = tasks.register<Sync>("syncSharedLocalizations") {
    from(rootProject.file("../ios/tscforex/tscforex")) {
        include("*.lproj/Localizable.strings")
    }
    into(generatedLocalizationAssets)
}

android {
    namespace = "com.example.tsc.forex"
    compileSdk {
        version = release(36) {
            minorApiLevel = 1
        }
    }

    defaultConfig {
        // Published package name on Google Play. Mirrors the iOS bundle
        // (talhacan.tscforex). NOTE: "com.example.*" is rejected by Play, so this
        // differs from the internal `namespace` (which only affects R/BuildConfig).
        applicationId = "com.talhacan.tscforex"
        minSdk = 24
        targetSdk = 36
        versionCode = 7
        versionName = "1.2.2"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    signingConfigs {
        // Play upload key (android/keystore.properties + app/upload-keystore.jks).
        // Falls back to the local debug keystore for adb installs when missing.
        create("release") {
            val propsFile = rootProject.file("keystore.properties")
            if (propsFile.exists()) {
                val props = Properties().apply {
                    propsFile.inputStream().use { stream -> load(stream) }
                }
                storeFile = file(props.getProperty("storeFile"))
                storePassword = props.getProperty("storePassword")
                keyAlias = props.getProperty("keyAlias")
                keyPassword = props.getProperty("keyPassword")
            } else {
                storeFile = file("${System.getProperty("user.home")}/.android/debug.keystore")
                storePassword = "android"
                keyAlias = "androiddebugkey"
                keyPassword = "android"
            }
        }
    }

    buildTypes {
        release {
            isDebuggable = false
            signingConfig = signingConfigs.getByName("release")
            optimization {
                enable = false
            }
        }
    }
    compileOptions {
        // java.time is used throughout charting and portfolio date handling.
        // Keep it safe on minSdk 24/25 as well as newer Android releases.
        isCoreLibraryDesugaringEnabled = true
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
    bundle {
        // The app has its own runtime language picker. Keep every supported
        // locale in the base install so switching also works while offline.
        language {
            enableSplit = false
        }
    }

    sourceSets {
        getByName("main").assets.directories.add(generatedLocalizationAssets.absolutePath)
    }
}

tasks.named("preBuild").configure {
    dependsOn(syncSharedLocalizations)
}

dependencies {
    coreLibraryDesugaring("com.android.tools:desugar_jdk_libs:2.0.3")
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.material.icons.extended)
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.graphics)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.navigation.compose)
    implementation(libs.androidx.datastore.preferences)
    implementation(libs.kotlinx.coroutines.android)
    implementation(libs.okhttp)
    implementation(libs.okhttp.logging)
    implementation(libs.coil.compose)
    implementation(libs.androidx.credentials)
    implementation(libs.androidx.credentials.play.services)
    implementation(libs.googleid)
    implementation(libs.billing.ktx)
    testImplementation(libs.junit)
    androidTestImplementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(libs.androidx.junit)
    debugImplementation(libs.androidx.compose.ui.test.manifest)
    debugImplementation(libs.androidx.compose.ui.tooling)
}
