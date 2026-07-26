package com.example.tsc.forex.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.outlined.MenuBook
import androidx.compose.material.icons.outlined.Checklist
import androidx.compose.material.icons.outlined.MyLocation
import androidx.compose.material.icons.outlined.Shield
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.example.tsc.forex.data.GamificationExperience
import com.example.tsc.forex.data.GamificationPrivacy
import com.example.tsc.forex.data.GamProfile
import com.example.tsc.forex.data.LessonStatus
import com.example.tsc.forex.store.AuthStore
import com.example.tsc.forex.store.PriceStore
import com.example.tsc.forex.ui.Card
import com.example.tsc.forex.ui.PrimaryActionButton
import com.example.tsc.forex.ui.theme.Th
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GamificationExperienceSection(
    auth: AuthStore,
    prices: PriceStore,
    initial: GamificationExperience,
    onExperience: (GamificationExperience) -> Unit,
    onProfile: (GamProfile) -> Unit,
    onToast: (String) -> Unit,
) {
    val scope = rememberCoroutineScope()
    var exp by remember { mutableStateOf(initial) }
    var openJourney by remember { mutableStateOf<Int?>(null) }
    var busy by remember { mutableStateOf(false) }
    var risk by remember { mutableStateOf("moderate") }
    var codes by remember { mutableStateOf<List<String>>(emptyList()) }
    var showInstrumentPicker by remember { mutableStateOf(false) }
    var goal by remember { mutableStateOf(initial.lessons.firstOrNull()?.code.orEmpty()) }
    var reflection by remember { mutableStateOf("") }
    var selectedLesson by remember { mutableStateOf(initial.lessons.firstOrNull()?.code.orEmpty()) }
    var selectedAnswer by remember { mutableStateOf<Int?>(null) }
    var disciplineCode by remember { mutableStateOf(initial.disciplineQuests.firstOrNull()?.code.orEmpty()) }
    var disciplineNote by remember { mutableStateOf("") }
    var paperCode by remember { mutableStateOf("GRAM_GOLD") }
    var paperKind by remember { mutableStateOf("gold") }
    var showPaperKindPicker by remember { mutableStateOf(false) }
    var showPaperAssetPicker by remember { mutableStateOf(false) }
    var paperSide by remember { mutableStateOf("buy") }
    var paperQuantity by remember { mutableStateOf("1") }
    var paperPrice by remember { mutableStateOf("") }
    var paperRisk by remember { mutableStateOf("1") }
    var paperThesis by remember { mutableStateOf("") }
    var paperInvalidation by remember { mutableStateOf("") }
    val paperInstrument = prices.prices[paperCode]
    val paperLabel = paperInstrument?.displayName ?: paperCode
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
            codes = it.watchCodes.take(3)
            goal = it.learningGoal
            reflection = it.reflection
        }
        alias = initial.privacy.alias
        leagueOptIn = initial.privacy.leagueOptIn
        personalized = initial.privacy.personalizedLearning
        socialOptIn = initial.privacy.socialOptIn
        notifications = initial.privacy.learningNotifications
    }

    BoxWithConstraints(Modifier.fillMaxWidth()) {
        val compact = maxWidth < 360.dp
        if (compact) {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                JourneyHubCard(Icons.Outlined.MyLocation, "Günlük Plan", "3 adım · yaklaşık 1 dakika", Modifier.fillMaxWidth()) { openJourney = 0 }
                JourneyHubCard(Icons.AutoMirrored.Outlined.MenuBook, "Öğrenme", "Kısa ders · tek soru", Modifier.fillMaxWidth()) { openJourney = 1 }
                JourneyHubCard(Icons.Outlined.Checklist, "Karar Günlüğü", "Sanal karar ve sonradan değerlendirme", Modifier.fillMaxWidth()) { openJourney = 2 }
                JourneyHubCard(Icons.Outlined.Shield, "Gizlilik", "Paylaşım tercihleri", Modifier.fillMaxWidth()) { openJourney = 3 }
            }
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    JourneyHubCard(Icons.Outlined.MyLocation, "Günlük Plan", "3 adım · yaklaşık 1 dakika", Modifier.weight(1f)) { openJourney = 0 }
                    JourneyHubCard(Icons.AutoMirrored.Outlined.MenuBook, "Öğrenme", "Kısa ders · tek soru", Modifier.weight(1f)) { openJourney = 1 }
                }
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    JourneyHubCard(Icons.Outlined.Checklist, "Karar Günlüğü", "Sanal karar ve sonradan değerlendirme", Modifier.weight(1f)) { openJourney = 2 }
                    JourneyHubCard(Icons.Outlined.Shield, "Gizlilik", "Paylaşım tercihleri", Modifier.weight(1f)) { openJourney = 3 }
                }
            }
        }
    }

    openJourney?.let { tab ->
        val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
        val title = when (tab) {
            0 -> "Günlük Plan"
            1 -> "Öğrenme"
            2 -> "Karar Günlüğü"
            else -> "Gizlilik"
        }
        ModalBottomSheet(
            onDismissRequest = { openJourney = null },
            sheetState = sheetState,
            containerColor = Th.bg,
        ) {
            Column(
                Modifier
                    .fillMaxWidth()
                    .fillMaxHeight(0.92f)
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 16.dp)
                    .padding(bottom = 28.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(
                    title,
                    color = Th.text,
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = androidx.compose.ui.text.style.TextAlign.Center,
                )
                when (tab) {
            0 -> {
                Card(Modifier.fillMaxWidth()) {
                    Column(verticalArrangement = Arrangement.spacedBy(9.dp)) {
                        Text(if (exp.dailyPlan == null) "BUGÜNÜ PLANLA" else "BUGÜNÜN PLANI",
                            color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        ChoiceRow(listOf("low" to "Düşük", "moderate" to "Orta", "high" to "Yüksek"), risk) { risk = it }
                        Text("İzlenecek varlıklar (${codes.size}/3)", color = Th.textDim, fontSize = 11.sp)
                        Button(
                            onClick = { showInstrumentPicker = true },
                            modifier = Modifier.fillMaxWidth(),
                            colors = ButtonDefaults.buttonColors(containerColor = Th.surfaceAlt, contentColor = Th.text),
                        ) {
                            val names = codes.mapNotNull { prices.prices[it]?.displayName }
                            Text(if (names.isEmpty()) "Varlık seç" else names.joinToString(" · "))
                        }
                        Text("Öğrenme hedefi", color = Th.textDim, fontSize = 11.sp)
                        ChoiceRow(exp.lessons.map { it.code to it.title }, goal) { goal = it }
                        GamField(reflection, { reflection = it }, "Dünkü plandan kısa not", minLines = 2)
                        ActionButton("Planı Kaydet ve Seriyi Sürdür", busy || goal.isEmpty() || codes.isEmpty()) {
                            scope.launch {
                                busy = true
                                runCatching {
                                    auth.newApiClient().saveDailyPlan(
                                        risk, codes.take(3),
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
            2 -> {
                Card(Modifier.fillMaxWidth()) {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("PAPER KARAR LABORATUVARI", color = Th.textDim, fontSize = 11.sp, fontWeight = FontWeight.Bold)
                        Text("Getiri değil tez, risk, geçersizleşme ve retrospektif puanlanır.", color = Th.textDim, fontSize = 12.sp)
                        TextButton(onClick = { showPaperKindPicker = true }) {
                            Text(
                                "Varlık: $paperLabel (${instrumentKindTitle(paperKind)})  · seç",
                                color = Th.accent,
                                fontSize = 12.sp,
                                fontWeight = FontWeight.SemiBold,
                            )
                        }
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            ChoiceRow(listOf("buy" to "Alış", "sell" to "Satış"), paperSide, Modifier.fillMaxWidth()) { paperSide = it }
                        }
                        BoxWithConstraints(Modifier.fillMaxWidth()) {
                            if (maxWidth < 380.dp) {
                                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                    GamField(paperQuantity, { paperQuantity = it }, "Miktar", keyboard = KeyboardType.Decimal)
                                    GamField(paperPrice, { paperPrice = it }, "Fiyat", keyboard = KeyboardType.Decimal)
                                    GamField(paperRisk, { paperRisk = it }, "Risk %", keyboard = KeyboardType.Decimal)
                                }
                            } else {
                                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                    GamField(paperQuantity, { paperQuantity = it }, "Miktar", Modifier.weight(1f), keyboard = KeyboardType.Decimal)
                                    GamField(paperPrice, { paperPrice = it }, "Fiyat", Modifier.weight(1f), keyboard = KeyboardType.Decimal)
                                    GamField(paperRisk, { paperRisk = it }, "Risk %", Modifier.weight(1f), keyboard = KeyboardType.Decimal)
                                }
                            }
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
                        SettingToggle("Anonim ligde görün", leagueOptIn) { leagueOptIn = it }
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
    }

    if (showInstrumentPicker) {
        InstrumentMultiPickerDialog(
            instruments = prices.all(),
            initialCodes = codes,
            maxSelection = 3,
            onDismiss = { showInstrumentPicker = false },
        ) { codes = it }
    }
    if (showPaperKindPicker) {
        InstrumentKindPickerDialog(
            instruments = prices.all(),
            selectedKind = paperKind,
            onDismiss = { showPaperKindPicker = false },
            onSelect = {
                paperKind = it
                showPaperKindPicker = false
                showPaperAssetPicker = true
            },
        )
    }
    if (showPaperAssetPicker) {
        InstrumentPickerDialog(
            instruments = prices.all(),
            kind = paperKind,
            selectedCode = paperCode,
            onDismiss = { showPaperAssetPicker = false },
            onSelect = {
                paperCode = it.code
                paperPrice = it.price.toString()
                showPaperAssetPicker = false
            },
        )
    }
}

@Composable
private fun JourneyHubCard(
    icon: ImageVector,
    title: String,
    subtitle: String,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    Surface(
        onClick = onClick,
        modifier = modifier.heightIn(min = 104.dp),
        color = Th.surface,
        contentColor = Th.text,
        shape = RoundedCornerShape(18.dp),
        border = androidx.compose.foundation.BorderStroke(1.dp, Th.stroke),
    ) {
        Column(
            Modifier.padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.Top) {
                Icon(icon, contentDescription = null, tint = Th.accent, modifier = Modifier.size(22.dp))
                Icon(
                    Icons.AutoMirrored.Filled.KeyboardArrowRight,
                    contentDescription = null,
                    tint = Th.textFaint,
                    modifier = Modifier.size(18.dp),
                )
            }
            Text(title, color = Th.text, fontSize = 14.sp, fontWeight = FontWeight.Bold)
            Text(subtitle, color = Th.textDim, fontSize = 11.sp, lineHeight = 14.sp)
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
            val isSelected = selected == value
            FilterChip(
                selected = isSelected,
                onClick = { onSelect(value) },
                label = { Text(title, fontSize = 11.sp, fontWeight = FontWeight.SemiBold) },
                colors = FilterChipDefaults.filterChipColors(
                    containerColor = Th.surfaceAlt,
                    labelColor = Th.textDim,
                    selectedContainerColor = Th.accent.copy(alpha = 0.18f),
                    selectedLabelColor = Th.accent,
                ),
                border = FilterChipDefaults.filterChipBorder(
                    enabled = true,
                    selected = isSelected,
                    borderColor = Th.stroke,
                    selectedBorderColor = Th.accent.copy(alpha = 0.65f),
                ),
            )
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
    PrimaryActionButton(text = label, onClick = onClick, enabled = !disabled)
}
