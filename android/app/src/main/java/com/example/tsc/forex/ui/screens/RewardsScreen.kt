package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.example.tsc.forex.data.GamBadge
import com.example.tsc.forex.data.GamProfile
import com.example.tsc.forex.data.Leaderboard
import com.example.tsc.forex.data.LeaderEntry
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.ScreenHeader
import com.example.tsc.forex.ui.StatTile
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

@Composable
fun RewardsScreen(auth: AuthStore) {
    val scope = rememberCoroutineScope()
    var profile by remember { mutableStateOf<GamProfile?>(null) }
    var board by remember { mutableStateOf<Leaderboard?>(null) }
    var loading by remember { mutableStateOf(true) }
    var checkingIn by remember { mutableStateOf(false) }
    var toast by remember { mutableStateOf<String?>(null) }

    suspend fun load() {
        try {
            val api = auth.newApiClient()
            profile = api.gamification()
            board = api.leaderboard(20)
        } catch (_: Exception) { /* keep last */ } finally { loading = false }
    }

    androidx.compose.runtime.LaunchedEffect(Unit) { load() }

    Column(Modifier.fillMaxSize().background(Th.bg)) {
        ScreenHeader("Ödüller", "Serini sürdür, XP kazan")
        Column(
            Modifier.verticalScroll(rememberScrollState()).padding(horizontal = 16.dp).padding(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            val p = profile
            if (loading && p == null) {
                repeat(3) { SkeletonCard() }
            } else if (p != null) {
                HeroCard(p)
                StatsRow(p)
                DailyReward(
                    checkedIn = p.checkedInToday,
                    busy = checkingIn,
                    onClaim = {
                        checkingIn = true
                        scope.launch {
                            try {
                                val res = auth.newApiClient().gamificationCheckIn()
                                profile = res.profile
                                toast = if (res.alreadyCheckedIn) "Bugün zaten giriş yaptın"
                                        else "+${res.xpGained} XP · ${res.profile.currentStreak} günlük seri!"
                                board = auth.newApiClient().leaderboard(20)
                            } catch (_: Exception) {
                                toast = "Bir hata oluştu"
                            } finally { checkingIn = false }
                        }
                    },
                )
                if (p.quests.isNotEmpty()) QuestsCard(p)
                if (p.badges.isNotEmpty()) BadgesCard(p.badges)
                board?.let { LeaderboardCard(it) }
            }
            toast?.let {
                Text(it, color = Th.accent, fontSize = 13.sp, fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.fillMaxWidth().padding(top = 4.dp))
            }
        }
    }
}

@Composable
private fun HeroCard(p: GamProfile) {
    Card(Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(16.dp)) {
            Box(contentAlignment = Alignment.Center) {
                CircularProgressIndicator(
                    progress = { p.progress.toFloat().coerceIn(0f, 1f) },
                    modifier = Modifier.size(72.dp),
                    color = Th.accent,
                    trackColor = Th.surfaceAlt,
                    strokeWidth = 6.dp,
                )
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("Sv", color = Th.textDim, fontSize = 9.sp)
                    Text("${p.level}", color = Th.text, fontSize = 20.sp, fontWeight = FontWeight.Bold)
                }
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text("Seviye ${p.level}", color = Th.text, fontSize = 16.sp, fontWeight = FontWeight.Bold)
                Text("${p.xpIntoLevel} / ${p.xpForNextLevel} XP", color = Th.textDim, fontSize = 12.sp)
                Box(Modifier.fillMaxWidth().height(8.dp).clip(RoundedCornerShape(4.dp)).background(Th.surfaceAlt)) {
                    Box(Modifier.fillMaxWidth(p.progress.toFloat().coerceIn(0f, 1f)).height(8.dp)
                        .clip(RoundedCornerShape(4.dp)).background(Th.accent))
                }
                Text("Toplam ${p.xp} XP", color = Th.textFaint, fontSize = 11.sp)
            }
        }
    }
}

@Composable
private fun StatsRow(p: GamProfile) {
    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        Card(Modifier.weight(1f)) { StatTile("Seri", "${p.currentStreak} gün", Th.accent) }
        Card(Modifier.weight(1f)) { StatTile("Bu hafta", "${p.weeklyXp} XP") }
        Card(Modifier.weight(1f)) { StatTile("Sıralama", if (p.rank > 0) "#${p.rank}" else "-") }
    }
}

@Composable
private fun DailyReward(checkedIn: Boolean, busy: Boolean, onClaim: () -> Unit) {
    Card(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("GÜNLÜK ÖDÜL", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            if (checkedIn) {
                Text("Bugünkü ödülünü aldın ✓ Yarın tekrar gel!", color = Th.up, fontSize = 13.sp)
            } else {
                Text("Serini sürdür ve XP kazan.", color = Th.textDim, fontSize = 13.sp)
                Button(
                    onClick = onClaim,
                    enabled = !busy,
                    shape = RoundedCornerShape(12.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (busy) CircularProgressIndicator(Modifier.size(18.dp), color = Th.bg, strokeWidth = 2.dp)
                    else Text("Günlük Ödülü Al", fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

@Composable
private fun QuestsCard(p: GamProfile) {
    Card(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Text("GÜNLÜK GÖREVLER", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            p.quests.forEach { q ->
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    Box(Modifier.size(22.dp).clip(CircleShape)
                        .background(if (q.completed) Th.up.copy(alpha = 0.18f) else Th.surfaceAlt),
                        contentAlignment = Alignment.Center) {
                        Text(if (q.completed) "✓" else "•", color = if (q.completed) Th.up else Th.textDim, fontSize = 12.sp)
                    }
                    Column(Modifier.weight(1f)) {
                        Text(q.title, color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.SemiBold)
                        Text(q.description, color = Th.textDim, fontSize = 11.sp)
                    }
                    Text("+${q.xp}", color = if (q.completed) Th.up else Th.textFaint, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

@Composable
private fun BadgesCard(badges: List<GamBadge>) {
    Card(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            val earned = badges.count { it.earned }
            Text("ROZETLER ($earned/${badges.size})", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            // Simple wrapping grid: 4 per row.
            badges.chunked(4).forEach { row ->
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                    row.forEach { b -> BadgeCell(b, Modifier.weight(1f)) }
                    repeat(4 - row.size) { Spacer(Modifier.weight(1f)) }
                }
            }
        }
    }
}

@Composable
private fun BadgeCell(b: GamBadge, modifier: Modifier) {
    Column(modifier, horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(4.dp)) {
        Box(Modifier.size(46.dp).clip(RoundedCornerShape(12.dp))
            .background(if (b.earned) Th.accent.copy(alpha = 0.18f) else Th.surfaceAlt),
            contentAlignment = Alignment.Center) {
            Text(b.icon.ifEmpty { "🏅" }, fontSize = 20.sp,
                modifier = if (b.earned) Modifier else Modifier.alpha(0.35f))
        }
        Text(b.name, color = if (b.earned) Th.text else Th.textFaint, fontSize = 9.sp,
            fontWeight = FontWeight.SemiBold, maxLines = 1)
    }
}

@Composable
private fun LeaderboardCard(board: Leaderboard) {
    Card(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("HAFTALIK LİDER TABLOSU", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            if (board.entries.isEmpty()) {
                Text("Bu hafta henüz sıralama yok. İlk sen ol!", color = Th.textDim, fontSize = 12.sp)
            } else {
                board.entries.forEach { e -> LeaderRow(e, highlight = e.userId == board.me?.userId) }
                board.me?.let { me ->
                    if (board.entries.none { it.userId == me.userId }) {
                        Box(Modifier.fillMaxWidth().height(1.dp).background(Th.stroke))
                        LeaderRow(me, highlight = true)
                    }
                }
            }
        }
    }
}

@Composable
private fun LeaderRow(e: LeaderEntry, highlight: Boolean) {
    Row(
        Modifier.fillMaxWidth()
            .clip(RoundedCornerShape(10.dp))
            .background(if (highlight) Th.accent.copy(alpha = 0.10f) else Color.Transparent)
            .padding(horizontal = 6.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Text("${e.rank}", color = Th.textDim, fontSize = 13.sp, fontWeight = FontWeight.Bold,
            modifier = Modifier.width(22.dp))
        Box(Modifier.size(28.dp).clip(CircleShape).background(Th.surfaceAlt), contentAlignment = Alignment.Center) {
            if (e.avatarUrl.isNotEmpty()) {
                AsyncImage(model = e.avatarUrl, contentDescription = null,
                    contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
            } else {
                Text(e.displayName.take(1).uppercase(), color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            }
        }
        Text(e.displayName.ifEmpty { "Kullanıcı" }, color = Th.text, fontSize = 13.sp,
            fontWeight = FontWeight.SemiBold, maxLines = 1, modifier = Modifier.weight(1f))
        Text("Sv ${e.level}", color = Th.textFaint, fontSize = 11.sp)
        Text("${e.xp} XP", color = Th.accent, fontSize = 12.sp, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun SkeletonCard() {
    Card(Modifier.fillMaxWidth()) {
        Box(Modifier.fillMaxWidth().height(64.dp).clip(RoundedCornerShape(10.dp)).background(Th.surfaceAlt))
    }
}
