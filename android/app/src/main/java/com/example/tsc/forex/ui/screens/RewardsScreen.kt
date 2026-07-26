package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.GamBadge
import com.example.tsc.forex.data.GamificationExperience
import com.example.tsc.forex.data.GamProfile
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.PrimaryActionButton
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.StatTile
import com.example.tsc.forex.ui.ToastOnChange
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

@Composable
fun RewardsScreen(auth: AuthStore, prices: PriceStore) {
    var profile by remember { mutableStateOf<GamProfile?>(null) }
    var experience by remember { mutableStateOf<GamificationExperience?>(null) }
    var loading by remember { mutableStateOf(true) }
    var toast by remember { mutableStateOf<String?>(null) }
    var selectedBadge by remember { mutableStateOf<GamBadge?>(null) }
    var celebrate by remember { mutableStateOf<List<GamBadge>>(emptyList()) }
    val scope = rememberCoroutineScope()

    suspend fun load() {
        try {
            val api = auth.newApiClient()
            profile = api.gamification()
            experience = runCatching { api.gamificationExperience() }.getOrNull()
        } catch (_: Exception) {
            /* keep last */
        } finally {
            loading = false
        }
    }

    androidx.compose.runtime.LaunchedEffect(Unit) { load() }

    Box(Modifier.fillMaxSize().background(Th.bg)) {
        Column(Modifier.fillMaxSize()) {
            ScreenHeader("Ödüller", "Serini koru, seviye atla, rozet topla")
            Column(
                Modifier
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 16.dp)
                    .padding(bottom = 96.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                val p = profile
                when {
                    loading && p == null -> repeat(3) { SkeletonCard() }
                    p != null -> {
                        HeroCard(p)
                        StatsRow(p)
                        experience?.let { exp ->
                            ProgressSummaryCard(exp)
                            BadgesCard(p.badges) { selectedBadge = it }
                            GamificationExperienceSection(
                                auth = auth,
                                prices = prices,
                                initial = exp,
                                onExperience = { experience = it },
                                onProfile = { profile = it },
                                onToast = { toast = it },
                            )
                        } ?: run {
                            BadgesCard(p.badges) { selectedBadge = it }
                        }
                        if (p.quests.isNotEmpty()) QuestsCard(p)
                        DailyReward(
                            checkedIn = p.checkedInToday,
                            busy = false,
                            onClaim = {
                                scope.launch {
                                    runCatching { auth.newApiClient().gamificationCheckIn() }
                                        .onSuccess { res ->
                                            profile = res.profile
                                            toast = if (res.alreadyCheckedIn) {
                                                "Bugün zaten giriş yaptın ✓"
                                            } else {
                                                "+${res.xpGained} XP  •  🔥 ${res.profile.currentStreak} gün"
                                            }
                                            if (res.newBadges.isNotEmpty()) celebrate = res.newBadges
                                        }
                                        .onFailure { toast = "Bağlantı hatası, tekrar dene" }
                                }
                            },
                        )
                    }
                    else -> {
                        Card {
                            Column(
                                Modifier.fillMaxWidth().padding(vertical = 16.dp),
                                horizontalAlignment = Alignment.CenterHorizontally,
                                verticalArrangement = Arrangement.spacedBy(8.dp),
                            ) {
                                Text("Ödüller yüklenemedi", color = Th.text, fontWeight = FontWeight.SemiBold)
                                PrimaryActionButton(
                                    text = "Tekrar dene",
                                    onClick = { scope.launch { loading = true; load() } },
                                )
                            }
                        }
                    }
                }
            }
        }

        ToastOnChange(toast) { toast = null }

        if (celebrate.isNotEmpty()) {
            Box(
                Modifier
                    .fillMaxSize()
                    .background(Th.bg.copy(alpha = 0.72f))
                    .clickable { celebrate = emptyList() },
                contentAlignment = Alignment.Center,
            ) {
                Card(Modifier.padding(40.dp)) {
                    Column(
                        Modifier.fillMaxWidth(),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        Text("🎉", fontSize = 40.sp)
                        Text(
                            if (celebrate.size > 1) "Yeni Rozetler!" else "Yeni Rozet!",
                            color = Th.text,
                            fontSize = 18.sp,
                            fontWeight = FontWeight.Bold,
                        )
                        celebrate.forEach { b ->
                            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                                Text(b.icon.ifEmpty { "🏅" }, fontSize = 28.sp)
                                Column {
                                    Text(b.name, color = Th.text, fontWeight = FontWeight.Bold)
                                    Text(b.description, color = Th.textDim, fontSize = 12.sp)
                                }
                            }
                        }
                        PrimaryActionButton(text = "Harika!", onClick = { celebrate = emptyList() })
                    }
                }
            }
        }
    }

    selectedBadge?.let { badge ->
        BadgeDetailSheet(badge = badge, onDismiss = { selectedBadge = null })
    }
}

@Composable
private fun HeroCard(p: GamProfile) {
    Card(Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(16.dp)) {
            Box(contentAlignment = Alignment.Center, modifier = Modifier.size(78.dp)) {
                CircularProgressIndicator(
                    progress = { p.progress.toFloat().coerceIn(0f, 1f) },
                    modifier = Modifier.size(78.dp),
                    color = Th.accent,
                    trackColor = Th.surfaceAlt,
                    strokeWidth = 7.dp,
                )
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("SEVİYE", color = Th.textDim, fontSize = 8.sp, fontWeight = FontWeight.Bold)
                    Text("${p.level}", color = Th.text, fontSize = 24.sp, fontWeight = FontWeight.Bold)
                }
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Seviye ${p.level}", color = Th.text, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                Box(Modifier.fillMaxWidth().height(10.dp).clip(RoundedCornerShape(5.dp)).background(Th.surfaceAlt)) {
                    Box(
                        Modifier
                            .fillMaxWidth(p.progress.toFloat().coerceIn(0f, 1f))
                            .height(10.dp)
                            .clip(RoundedCornerShape(5.dp))
                            .background(Th.accent),
                    )
                }
                Text(
                    "${p.xpIntoLevel} / ${p.xpForNextLevel} XP  •  Sonraki seviyeye ${maxOf(0, p.xpForNextLevel - p.xpIntoLevel)} XP",
                    color = Th.textDim,
                    fontSize = 12.sp,
                )
            }
        }
    }
}

@Composable
private fun StatsRow(p: GamProfile) {
    Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        Card(Modifier.weight(1f)) {
            StatTile("🔥 Seri", "${p.currentStreak} gün", if (p.currentStreak > 0) Th.down else Th.text)
        }
        Card(Modifier.weight(1f)) { StatTile("Haftalık XP", "${p.weeklyXp}", Th.accent) }
        Card(Modifier.weight(1f)) { StatTile("Toplam XP", "${p.xp}") }
    }
}

@Composable
private fun ProgressSummaryCard(exp: GamificationExperience) {
    Card(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text("Gelişim Özeti", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                Text("v${exp.ruleVersion}", color = Th.textFaint, fontSize = 10.sp, fontWeight = FontWeight.Bold)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                SummaryTile("Ustalık", "${exp.weeklySummary.masteredLessons}/${exp.weeklySummary.totalLessons}", Modifier.weight(1f))
                SummaryTile("Tekrar", "${exp.weeklySummary.dueReviews}", Modifier.weight(1f))
                SummaryTile("Karar", "${exp.paperTrades.size}", Modifier.weight(1f))
            }
            Text(
                "Puanlama finansal getiriden bağımsızdır; öğrenme ve karar sürecini ölçer.",
                color = Th.textFaint,
                fontSize = 10.sp,
            )
        }
    }
}

@Composable
private fun SummaryTile(label: String, value: String, modifier: Modifier) {
    Column(
        modifier
            .clip(RoundedCornerShape(10.dp))
            .background(Th.surfaceAlt)
            .padding(9.dp),
    ) {
        Text(label, color = Th.textFaint, fontSize = 9.sp)
        Text(value, color = Th.text, fontSize = 16.sp, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun DailyReward(checkedIn: Boolean, busy: Boolean, onClaim: () -> Unit) {
    if (checkedIn) {
        Card(Modifier.fillMaxWidth()) {
            Text("Günlük giriş tamamlandı — yarın tekrar gel, serini büyüt!", color = Th.up, fontSize = 13.sp)
        }
    } else {
        PrimaryActionButton(text = "Günlük ödülü al", onClick = onClaim, enabled = !busy)
    }
}

@Composable
private fun QuestsCard(p: GamProfile) {
    Card(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Row(Modifier.fillMaxWidth()) {
                Text("Günlük Görevler", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                Spacer(Modifier.weight(1f))
                val done = p.quests.count { it.completed }
                Text("$done/${p.quests.size}", color = Th.textDim, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
            p.quests.forEach { q ->
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Text(if (q.completed) "✓" else "○", color = if (q.completed) Th.up else Th.textFaint, fontSize = 18.sp)
                    Column(Modifier.weight(1f)) {
                        Text(q.title, color = if (q.completed) Th.textDim else Th.text, fontSize = 14.sp, fontWeight = FontWeight.SemiBold)
                        Text(q.description, color = Th.textFaint, fontSize = 11.sp)
                    }
                    Text("+${q.xp}", color = Th.accent, fontSize = 13.sp, fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

@Composable
private fun BadgesCard(badges: List<GamBadge>, onSelect: (GamBadge) -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Row(Modifier.fillMaxWidth()) {
                Text("Rozetler", color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                Spacer(Modifier.weight(1f))
                val earned = badges.count { it.earned }
                Text("$earned/${badges.size}", color = Th.textDim, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
            BoxWithConstraints(Modifier.fillMaxWidth()) {
                val fontScale = LocalDensity.current.fontScale
                val columnCount = when {
                    fontScale >= 1.2f && maxWidth < 400.dp -> 2
                    maxWidth < 360.dp -> 3
                    else -> 4
                }
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    badges.chunked(columnCount).forEach { row ->
                        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), modifier = Modifier.fillMaxWidth()) {
                            row.forEach { b ->
                                BadgeCell(b, Modifier.weight(1f).clickable { onSelect(b) })
                            }
                            repeat(columnCount - row.size) { Spacer(Modifier.weight(1f)) }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun BadgeCell(b: GamBadge, modifier: Modifier) {
    Column(
        modifier,
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Box(contentAlignment = Alignment.Center, modifier = Modifier.size(54.dp)) {
            val ring = Th.stroke
            val progressColor = if (b.earned) Th.up else Th.accent
            Canvas(Modifier.size(54.dp)) {
                drawCircle(color = ring, style = Stroke(2.5.dp.toPx()))
                drawArc(
                    color = progressColor,
                    startAngle = -90f,
                    sweepAngle = (b.normalizedProgress * 360).toFloat(),
                    useCenter = false,
                    style = Stroke(width = 2.5.dp.toPx(), cap = StrokeCap.Round),
                )
            }
            Box(
                Modifier
                    .size(48.dp)
                    .clip(CircleShape)
                    .background(if (b.earned) Th.accent.copy(alpha = 0.18f) else Th.surfaceAlt),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    b.icon.ifEmpty { "🏅" },
                    fontSize = 22.sp,
                    modifier = if (b.earned) Modifier else Modifier.alpha(0.35f),
                )
            }
            if (!b.earned) {
                Text("🔒", fontSize = 10.sp, modifier = Modifier.align(Alignment.BottomEnd))
            }
        }
        Text(
            b.name,
            color = if (b.earned) Th.text else Th.textFaint,
            fontSize = 9.5.sp,
            fontWeight = FontWeight.Medium,
            textAlign = TextAlign.Center,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
            lineHeight = 11.sp,
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = 28.dp),
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun BadgeDetailSheet(badge: GamBadge, onDismiss: () -> Unit) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState, containerColor = Th.bg) {
        Column(
            Modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp)
                .padding(bottom = 36.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            Text(badge.icon.ifEmpty { "🏅" }, fontSize = 44.sp, modifier = if (badge.earned) Modifier else Modifier.alpha(0.62f))
            Text(badge.name, color = Th.text, fontSize = 21.sp, fontWeight = FontWeight.Bold)
            Column(Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text("Kazanma koşulu", color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                Text(badge.description, color = Th.textDim, fontSize = 13.sp)
            }
            if (badge.target > 0) {
                Column(
                    Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(12.dp))
                        .background(Th.surfaceAlt)
                        .padding(13.dp),
                    verticalArrangement = Arrangement.spacedBy(9.dp),
                ) {
                    Row(Modifier.fillMaxWidth()) {
                        Text(if (badge.earned) "Tamamlandı" else "İlerleme", color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                        Spacer(Modifier.weight(1f))
                        Text("%${badge.progressPercent}", color = if (badge.earned) Th.up else Th.accent, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                    }
                    Box(
                        Modifier
                            .fillMaxWidth()
                            .height(6.dp)
                            .clip(RoundedCornerShape(3.dp))
                            .background(Th.surface),
                    ) {
                        val progress = badge.normalizedProgress.toFloat().coerceIn(0f, 1f)
                        if (progress > 0f) {
                            Box(
                                Modifier
                                    .fillMaxWidth(progress)
                                    .height(6.dp)
                                    .background(if (badge.earned) Th.up else Th.accent),
                            )
                        }
                    }
                    Row(Modifier.fillMaxWidth()) {
                        Text(
                            "${minOf(badge.current, badge.target)} / ${badge.target}",
                            color = Th.textDim,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                        )
                        Spacer(Modifier.weight(1f))
                        Text(
                            when {
                                badge.earned -> "Rozet kazanıldı"
                                else -> "${maxOf(0, badge.target - badge.current)} kaldı"
                            },
                            color = Th.textFaint,
                            fontSize = 10.sp,
                        )
                    }
                }
            }
            Text(
                when {
                    badge.earned && !badge.earnedAt.isNullOrBlank() -> "Kazanıldı: ${badge.earnedAt.take(10)}"
                    badge.earned -> "Rozet kazanıldı"
                    else -> "Henüz kazanılmadı"
                },
                color = if (badge.earned) Th.up else Th.textDim,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
            )
        }
    }
}

@Composable
private fun SkeletonCard() {
    Card(Modifier.fillMaxWidth()) {
        Box(Modifier.fillMaxWidth().height(64.dp).clip(RoundedCornerShape(10.dp)).background(Th.surfaceAlt))
    }
}
