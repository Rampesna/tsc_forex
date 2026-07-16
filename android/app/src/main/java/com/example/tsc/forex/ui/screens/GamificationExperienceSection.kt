package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.GamificationExperience
import com.example.tsc.forex.data.GamificationPrivacy
import com.example.tsc.forex.data.GamProfile
import com.example.tsc.forex.data.LessonStatus
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

@Composable
fun GamificationExperienceSection(
    auth: AuthStore,
    initial: GamificationExperience,
    onExperience: (GamificationExperience) -> Unit,
    onProfile: (GamProfile) -> Unit,
    onToast: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    var exp by remember { mutableStateOf(initial) }
    var tab by remember { mutableIntStateOf(0) }
    var busy by remember { mutableStateOf(false) }
    var risk by remember { mutableStateOf("moderate") }
    var codes by remember { mutableStateOf("") }
    var goal by remember { mutableStateOf(initial.lessons.firstOrNull()?.code.orEmpty()) }
    var reflection by remember { mutableStateOf("") }
    var selectedLesson by remember { mutableStateOf(initial.lessons.firstOrNull()?.code.orEmpty()) }
    var selectedAnswer by remember { mutableStateOf<Int?>(null) }
    var disciplineCode by remember { mutableStateOf(initial.disciplineQuests.firstOrNull()?.code.orEmpty()) }
    var disciplineNote by remember { mutableStateOf("") }
    var paperCode by remember { mutableStateOf("GRAM_GOLD") }
    var paperSide by remember { mutableStateOf("buy") }
    var paperQuantity by remember { mutableStateOf("1") }
    var paperPrice by remember { mutableStateOf("") }
    var paperRisk by remember { mutableStateOf("1") }
    var paperThesis by remember { mutableStateOf("") }
    var paperInvalidation by remember { mutableStateOf("") }
    var reviewTradeId by remember { mutableStateOf("") }
    var retrospective by remember { mutableStateOf("") }
    var followedPlan by remember { mutableStateOf(true) }
    var alias by remember { mutableStateOf(initial.privacy.alias) }
    var leagueOptIn by remember { mutableStateOf(initial.privacy.leagueOptIn) }
    var personalized by remember { mutableStateOf(initial.privacy.personalizedLearning) }
    var socialOptIn by remember { mutableStateOf(initial.privacy.socialOptIn) }
    var notifications by remember { mutableStateOf(initial.privacy.learningNotifications) }
    var groupName by remember { mutableStateOf("") }
    var inviteCode by remember { mutableStateOf("") }

    suspend fun reload() {
        runCatching { auth.newApiClient().gamificationExperience() }.onSuccess {
            exp = it
            onExperience(it)
        }
    }

    LaunchedEffect(initial) {
        exp = initial
        initial.dailyPlan?.let {
            risk = it.riskAppetite
            codes = it.watchCodes.joinToString(", ")
            goal = it.learningGoal
            reflection = it.reflection
        }
        alias = initial.privacy.alias
        leagueOptIn = initial.privacy.leagueOptIn
        personalized = initial.privacy.personalizedLearning
        socialOptIn = initial.privacy.socialOptIn
        notifications = initial.privacy.learningNotifications
    }

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Card(Modifier.fillMaxWidth()) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text("GELİŞİM ÖZETİ", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                    Text("v${exp.ruleVersion}", color = Th.textFaint, fontSize = 10.sp)
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    SummaryTile("Ustalık", "${exp.weeklySummary.masteredLessons}/${exp.weeklySummary.totalLessons}", Modifier.weight(1f))
                    SummaryTile("Tekrar", "${exp.weeklySummary.dueReviews}", Modifier.weight(1f))
                    SummaryTile("Sezon", "${exp.weeklySummary.seasonPoints}", Modifier.weight(1f))
                }
                Text("Puanlama finansal getiriden bağımsızdır; öğrenme ve karar sürecini ölçer.",
                    color = Th.textFaint, fontSize = 10.sp)
            }
        }

        Row(Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(4.dp)) {
            listOf("Bugün", "Öğren", "Sezon", "Paper", "Gizlilik").forEachIndexed { index, title ->
                TextButton(onClick = { tab = index }, colors = ButtonDefaults.textButtonColors(
                    contentColor = if (tab == index) Th.accent else Th.textDim,
                )) { Text(title, fontWeight = FontWeight.Bold) }
            }
        }

        when (tab) {
            0 -> {
                Card(Modifier.fillMaxWidth()) {
                    Column(verticalArrangement = Arrangement.spacedBy(9.dp)) {
                        Text(if (exp.dailyPlan == null) "BUGÜNÜ PLANLA" else "BUGÜNÜN PLANI",
                            color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        ChoiceRow(listOf("low" to "Düşük", "moderate" to "Orta", "high" to "Yüksek"), risk) { risk = it }
                        GamField(codes, { codes = it }, "İzlenecek kodlar: GRAM_GOLD, USDTRY, AAPL")
                        Text("Öğrenme hedefi", color = Th.textDim, fontSize = 11.sp)
                        ChoiceRow(exp.lessons.map { it.code to it.title }, goal) { goal = it }
                        GamField(reflection, { reflection = it }, "Dünkü plandan kısa not", minLines = 2)
                        ActionButton("Planı Kaydet ve Seriyi Sürdür", busy || goal.isEmpty()) {
                            scope.launch {
                                busy = true
                                runCatching {
                                    auth.newApiClient().saveDailyPlan(
                                        risk, codes.split(",").map { it.trim().uppercase() }.filter { it.isNotEmpty() }.take(3),
                                        goal, reflection,
                                    )
                                }.onSuccess {
                                    onProfile(it.profile)
                                    onToast("Plan kaydedildi · +${it.xpGained} XP · ${it.profile.currentStreak} günlük seri")
                                    reload()
                                }.onFailure { onToast(it.message ?: "Plan kaydedilemedi") }
                                busy = false
                            }
                        }
                    }
                }
                Card(Modifier.fillMaxWidth()) {
                    Column(verticalArrangement = Arrangement.spacedBy(9.dp)) {
                        Text("PORTFÖY DİSİPLİNİ", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        Text("Sonuca değil karar sürecine puan verilir.", color = Th.textDim, fontSize = 12.sp)
                        ChoiceRow(exp.disciplineQuests.map { it.code to it.title }, disciplineCode) { disciplineCode = it }
                        exp.disciplineQuests.firstOrNull { it.code == disciplineCode }?.let {
                            Text(it.description, color = Th.textFaint, fontSize = 11.sp)
                        }
                        GamField(disciplineNote, { disciplineNote = it },
                            "Risk, tez veya plan sapmasını en az 20 karakterle değerlendir", minLines = 3)
                        ActionButton("Süreç Değerlendirmesini Tamamla", busy || disciplineNote.trim().length < 20) {
                            scope.launch {
                                busy = true
                                runCatching { auth.newApiClient().completeDiscipline(disciplineCode, disciplineNote) }
                                    .onSuccess {
                                        onToast(if (it.first) "Süreç tamam · +${it.second} XP" else "Bugün zaten tamamlandı")
                                        disciplineNote = ""; reload()
                                    }.onFailure { onToast(it.message ?: "Kaydedilemedi") }
                                busy = false
                            }
                        }
                    }
                }
            }
            1 -> LearningCard(exp, selectedLesson, selectedAnswer,
                onSelectLesson = { selectedLesson = it; selectedAnswer = null },
                onSelectAnswer = { selectedAnswer = it },
                busy = busy,
                onSubmit = { lesson ->
                    val answer = selectedAnswer ?: return@LearningCard
                    scope.launch {
                        busy = true
                        runCatching { auth.newApiClient().submitQuiz(lesson.code, lesson.question.code, answer) }
                            .onSuccess {
                                onToast(if (it.correct) "Doğru · +${it.xpGained} XP" else "Kartları tekrar gözden geçir")
                                selectedAnswer = null; reload()
                            }.onFailure { onToast(it.message ?: "Yanıt gönderilemedi") }
                        busy = false
                    }
                })
            2 -> Card(Modifier.fillMaxWidth()) {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(exp.season.name, color = Th.text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                    Text("${exp.season.code} · ${exp.season.league} Ligi · #${exp.season.rank} · ${exp.season.points} puan",
                        color = Th.accent, fontSize = 12.sp, fontWeight = FontWeight.Bold)
                    Text(exp.season.scoring, color = Th.textFaint, fontSize = 10.sp)
                    exp.season.standings.take(8).forEach {
                        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                            Text("#${it.rank}  ${it.displayName.ifEmpty { "Yatırımcı" }}",
                                color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.SemiBold)
                            Text("${it.xp} puan", color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        }
                    }
                }
            }
            3 -> {
                Card(Modifier.fillMaxWidth()) {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("PAPER KARAR LABORATUVARI", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        Text("Getiri değil tez, risk, geçersizleşme ve retrospektif puanlanır.", color = Th.textDim, fontSize = 12.sp)
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            GamField(paperCode, { paperCode = it.uppercase() }, "Kod", Modifier.weight(1f))
                            ChoiceRow(listOf("buy" to "Alış", "sell" to "Satış"), paperSide, Modifier.weight(2f)) { paperSide = it }
                        }
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            GamField(paperQuantity, { paperQuantity = it }, "Miktar", Modifier.weight(1f), keyboard = KeyboardType.Decimal)
                            GamField(paperPrice, { paperPrice = it }, "Fiyat", Modifier.weight(1f), keyboard = KeyboardType.Decimal)
                            GamField(paperRisk, { paperRisk = it }, "Risk %", Modifier.weight(1f), keyboard = KeyboardType.Decimal)
                        }
                        GamField(paperThesis, { paperThesis = it }, "Tez (en az 20 karakter)", minLines = 2)
                        GamField(paperInvalidation, { paperInvalidation = it }, "Geçersizleşme koşulu", minLines = 2)
                        ActionButton("Kararı Kilitle", busy || paperThesis.length < 20 || paperInvalidation.length < 10 ||
                            paperPrice.replace(",", ".").toDoubleOrNull() == null) {
                            scope.launch {
                                busy = true
                                runCatching {
                                    auth.newApiClient().createPaperTrade(
                                        paperCode, paperSide,
                                        paperQuantity.replace(",", ".").toDoubleOrNull() ?: 0.0,
                                        paperPrice.replace(",", ".").toDoubleOrNull() ?: 0.0,
                                        paperThesis, paperRisk.replace(",", ".").toDoubleOrNull() ?: 0.0,
                                        paperInvalidation,
                                    )
                                }.onSuccess {
                                    onToast("Paper karar günlüğüne eklendi")
                                    paperThesis = ""; paperInvalidation = ""; reload()
                                }.onFailure { onToast(it.message ?: "Eklenemedi") }
                                busy = false
                            }
                        }
                    }
                }
                exp.paperTrades.take(8).forEach { trade ->
                    Card(Modifier.fillMaxWidth()) {
                        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                Text("${trade.code} · ${if (trade.side == "buy") "Alış" else "Satış"}",
                                    color = Th.text, fontSize = 13.sp, fontWeight = FontWeight.Bold)
                                Text(if (trade.reviewedAt == null) "Retrospektif bekliyor" else "Süreç ${trade.processScore}/100",
                                    color = if (trade.reviewedAt == null) Th.textDim else Th.up, fontSize = 10.sp)
                            }
                            Text(trade.thesis, color = Th.textDim, fontSize = 11.sp)
                            if (trade.reviewedAt == null) {
                                TextButton(onClick = { reviewTradeId = if (reviewTradeId == trade.id) "" else trade.id }) {
                                    Text("Kararı değerlendir")
                                }
                            }
                            if (reviewTradeId == trade.id) {
                                SettingToggle("Planıma uydum", followedPlan) { followedPlan = it }
                                GamField(retrospective, { retrospective = it }, "Sonuçtan bağımsız süreç değerlendirmesi", minLines = 2)
                                ActionButton("Retrospektifi Tamamla", busy || retrospective.length < 20) {
                                    scope.launch {
                                        busy = true
                                        runCatching { auth.newApiClient().reviewPaperTrade(trade.id, followedPlan, retrospective) }
                                            .onSuccess {
                                                onToast("Retrospektif tamam · +${it.second} XP")
                                                reviewTradeId = ""; retrospective = ""; reload()
                                            }.onFailure { onToast(it.message ?: "Kaydedilemedi") }
                                        busy = false
                                    }
                                }
                            }
                        }
                    }
                }
            }
            else -> {
                Card(Modifier.fillMaxWidth()) {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("GİZLİLİK KONTROLLERİ", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        Text("Para, pozisyon ve getiri sosyal yüzeylerde paylaşılmaz.", color = Th.textDim, fontSize = 12.sp)
                        GamField(alias, { alias = it }, "Takma ad")
                        SettingToggle("Sezon sıralamasında görün", leagueOptIn) { leagueOptIn = it }
                        SettingToggle("Kişiselleştirilmiş öğrenme", personalized) { personalized = it }
                        SettingToggle("Öğrenme bildirimleri", notifications) { notifications = it }
                        SettingToggle("Gizlilik güvenli çalışma grupları", socialOptIn) { socialOptIn = it }
                        ActionButton("Tercihleri Kaydet", busy) {
                            scope.launch {
                                busy = true
                                val privacy = GamificationPrivacy(
                                    exp.privacy.userId, alias, leagueOptIn, personalized, socialOptIn, notifications,
                                )
                                runCatching { auth.newApiClient().saveGamificationPrivacy(privacy) }
                                    .onSuccess { onToast("Gizlilik tercihleri kaydedildi"); reload() }
                                    .onFailure { onToast(it.message ?: "Kaydedilemedi") }
                                busy = false
                            }
                        }
                    }
                }
                if (socialOptIn) {
                    Card(Modifier.fillMaxWidth()) {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            Text("ÇALIŞMA GRUPLARI", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                GamField(groupName, { groupName = it }, "Yeni grup adı", Modifier.weight(1f))
                                Button(onClick = {
                                    scope.launch {
                                        busy = true
                                        runCatching { auth.newApiClient().createStudyGroup(groupName) }
                                            .onSuccess { onToast("Grup hazır · kod ${it.inviteCode}"); groupName = ""; reload() }
                                            .onFailure { onToast(it.message ?: "Oluşturulamadı") }
                                        busy = false
                                    }
                                }, enabled = !busy && groupName.length >= 3) { Text("Oluştur") }
                            }
                            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                GamField(inviteCode, { inviteCode = it.uppercase() }, "Davet kodu", Modifier.weight(1f))
                                Button(onClick = {
                                    scope.launch {
                                        busy = true
                                        runCatching { auth.newApiClient().joinStudyGroup(inviteCode) }
                                            .onSuccess { onToast("Çalışma grubuna katıldın"); inviteCode = ""; reload() }
                                            .onFailure { onToast(it.message ?: "Katılınamadı") }
                                        busy = false
                                    }
                                }, enabled = !busy && inviteCode.length >= 6) { Text("Katıl") }
                            }
                            exp.studyGroups.forEach {
                                Row(Modifier.fillMaxWidth().background(Th.surfaceAlt, RoundedCornerShape(9.dp)).padding(9.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween) {
                                    Text("${it.name} · ${it.memberCount}/8", color = Th.text, fontSize = 11.sp)
                                    Text(it.inviteCode, color = Th.accent, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun LearningCard(
    exp: GamificationExperience,
    selectedCode: String,
    selectedAnswer: Int?,
    onSelectLesson: (String) -> Unit,
    onSelectAnswer: (Int) -> Unit,
    busy: Boolean,
    onSubmit: (LessonStatus) -> Unit,
) {
    val lesson = exp.lessons.firstOrNull { it.code == selectedCode } ?: exp.lessons.firstOrNull() ?: return
    Card(Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(9.dp)) {
            Text("MİKRO DERS VE USTALIK", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
            ChoiceRow(exp.lessons.map { it.code to "${if (it.completed) "✓ " else if (it.dueForReview) "↻ " else ""}${it.title}" },
                lesson.code, onSelect = onSelectLesson)
            Text("${lesson.title} · %${lesson.mastery}", color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.Bold)
            Text(lesson.summary, color = Th.textDim, fontSize = 11.sp)
            lesson.cards.forEachIndexed { index, card ->
                Text("${index + 1}. $card", color = Th.text, fontSize = 11.sp,
                    modifier = Modifier.fillMaxWidth().background(Th.surfaceAlt, RoundedCornerShape(9.dp)).padding(9.dp))
            }
            Text(lesson.question.text, color = Th.text, fontSize = 12.sp, fontWeight = FontWeight.Bold)
            lesson.question.choices.forEachIndexed { index, choice ->
                TextButton(onClick = { onSelectAnswer(index) }, modifier = Modifier.fillMaxWidth()
                    .background(if (selectedAnswer == index) Th.accent.copy(alpha = .15f) else Th.surfaceAlt, RoundedCornerShape(9.dp))) {
                    Text(choice, color = Th.text, fontSize = 12.sp)
                }
            }
            ActionButton("Bilgimi Kontrol Et", busy || selectedAnswer == null) { onSubmit(lesson) }
        }
    }
}

@Composable
private fun SummaryTile(label: String, value: String, modifier: Modifier) {
    Column(modifier.background(Th.surfaceAlt, RoundedCornerShape(9.dp)).padding(9.dp)) {
        Text(label, color = Th.textFaint, fontSize = 9.sp)
        Text(value, color = Th.text, fontSize = 16.sp, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun ChoiceRow(
    choices: List<Pair<String, String>>,
    selected: String,
    modifier: Modifier = Modifier,
    onSelect: (String) -> Unit,
) {
    Row(modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(4.dp)) {
        choices.forEach { (value, title) ->
            TextButton(onClick = { onSelect(value) }, colors = ButtonDefaults.textButtonColors(
                contentColor = if (selected == value) Th.accent else Th.textDim,
            )) { Text(title, fontSize = 11.sp, fontWeight = FontWeight.SemiBold) }
        }
    }
}

@Composable
private fun GamField(
    value: String,
    onValueChange: (String) -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    minLines: Int = 1,
    keyboard: KeyboardType = KeyboardType.Text,
) {
    OutlinedTextField(
        value = value, onValueChange = onValueChange, label = { Text(label, fontSize = 10.sp) },
        modifier = modifier.fillMaxWidth(), minLines = minLines,
        keyboardOptions = KeyboardOptions(keyboardType = keyboard),
    )
}

@Composable
private fun SettingToggle(label: String, checked: Boolean, onChange: (Boolean) -> Unit) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(label, color = Th.text, fontSize = 12.sp)
        Switch(checked = checked, onCheckedChange = onChange)
    }
}

@Composable
private fun ActionButton(label: String, disabled: Boolean, onClick: () -> Unit) {
    Button(
        onClick = onClick, enabled = !disabled, modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = ButtonDefaults.buttonColors(containerColor = Th.accent, contentColor = Th.bg),
    ) { Text(label, fontWeight = FontWeight.Bold) }
}
