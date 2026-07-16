# Gamification 10× Ürün Yol Haritası

## Ürün ilkesi

Gamification; fiyat yönü tahmini, işlem sıklığı veya kazanılan para üzerinden değil, öğrenme, risk farkındalığı ve tutarlı süreç üzerinden ödül vermelidir. Kullanıcı daha çok risk aldığı için değil, kararını gerekçelendirdiği, planına uyduğu ve bilgisini kalıcı biçimde geliştirdiği için ilerlemelidir.

Kuzey yıldızı: **haftalık en az üç anlamlı öğrenme veya disiplin eylemi tamamlayan aktif kullanıcı oranı**.

Koruyucu metrikler:

- D7/D30 elde tutma artarken bildirim kapatma, görev terk etme ve destek şikâyeti artmamalı.
- XP'nin en az %70'i öğrenme ve risk-disiplini eylemlerinden gelmeli.
- Tekrarlanan/farm edilen etkinliklerden doğan XP oranı %1'in altında kalmalı.
- Gerçek veya kâğıt portföy getirisi hiçbir sıralamanın doğrudan girdisi olmamalı.
- Her özellik iOS, Android ve web için aynı sunucu durumunu ve aynı ödül kurallarını kullanmalı.

## Mevcut temel ve önce kapatılacak açık

Sistemde seviye, XP, günlük giriş, seri, rozet, günlük görev ve haftalık liderlik mevcut. Sunucu doğrulaması doğru başlangıçtır; ancak günlük giriş akışı tek DB transaction'ı ve benzersiz idempotency anahtarıyla korunmadığı için eşzamanlı isteklerde çift ödül riski vardır. Favori/alarm görevleri de kullanıcıyı anlamlı davranış yerine ekle-sil döngüsüne teşvik edebilir. İlk faz yeni görsel özellikten önce bu ekonomi açığını kapatır.

## Faz 1 — Güvenilir temel ve platform paritesi (0–6 hafta)

### 1. Transactional XP defteri ve anti-farm

- Etki: Çok yüksek
- Karmaşıklık: Orta
- Abuse riski: Uygulanmazsa çok yüksek; uygulandıktan sonra düşük
- Backend: `gamification_events` için `event_id/idempotency_key`, kaynak türü ve kaynak kimliği; `(user_id, idempotency_key)` unique index; kullanıcı state satırına kilit; event + XP + rozet + görev ilerlemesini tek transaction'da yazan `AwardEvent`.
- İstemciler: Her eylem için kalıcı UUID; offline retry aynı UUID ile yapılır; ödül yalnız sunucu cevabından gösterilir.
- Kabul kapısı: 100 eşzamanlı aynı check-in isteği tam olarak bir XP kaydı üretmeli.

### 2. Check-in'i “günü planla” ritüeline dönüştür

- Etki: Yüksek
- Karmaşıklık: Düşük
- Abuse riski: Düşük
- Deneyim: Kullanıcı risk iştahı, bugün takip edeceği en fazla üç varlık ve öğrenme hedefinden birini seçer. Ödül butona basmaya değil, kısa planın kaydedilmesine verilir.
- Backend: Günlük plan kaydı, İstanbul gün anahtarı, bir kez ödül.
- İstemciler: Üç platformda aynı iki adımlı sheet ve ertesi gün önceki plana kısa geri bakış.

### 3. Görevleri anlamlı ve tersine çevrilemez eylemlere bağla

- Etki: Yüksek
- Karmaşıklık: Orta
- Abuse riski: Orta
- Değişiklik: “Favori ekle” yerine “iki varlığı karşılaştır ve bir not kaydet”; “alarm kur” için alarmın en az belirli süre aktif kalması veya kullanıcı tarafından gerekçelendirilmesi.
- Backend: Pending → verified → rewarded görev durumu, gecikmeli doğrulama worker'ı, günlük/haftalık XP tavanları.
- İstemciler: Doğrulanmayı bekleyen ödülü açıkça gösterir; anında kazanılmış gibi animasyon oynatmaz.

### 4. Platform paritesi ve uzaktan özellik yönetimi

- Etki: Yüksek
- Karmaşıklık: Orta
- Abuse riski: Düşük
- Backend: Sürümlü görev kataloğu, eligibility kuralları, feature flag ve minimum istemci sürümü.
- İstemciler: Tek API şemasından görev, rozet, ilerleme ve ödül nedeni; deep link hedefleri; erişilebilirlik ve lokalizasyon.
- Kabul kapısı: Aynı hesapta üç platformun XP, seri, görev ve claim durumu birebir eşleşmeli.

### 5. XP ekonomi telemetrisi

- Etki: Orta
- Karmaşıklık: Düşük
- Abuse riski: Düşük
- Backend/analitik: `offered → started → verified → rewarded → retained` funnel'ı, event kaynağı, kural sürümü, red nedeni ve risk sinyali; PII içermeyen ürün analitiği.
- Operasyon: XP dağılımı, p95 kazanım hızı, şüpheli tekrar ve notification opt-out alarmı.

Faz 1 çıkış koşulları: duplicate ödül yok; üç platform paritesi sözleşme testleriyle doğrulanmış; XP kaynakları denetlenebilir; farm edilebilir favori/alarm döngüsü kapalı.

## Faz 2 — Öğrenme ve portföy disiplini (6–14 hafta)

### 1. 2–4 dakikalık mikro dersler

- Etki: Çok yüksek
- Karmaşıklık: Orta
- Abuse riski: Düşük
- İçerik: spread, kur riski, pozisyon boyutlama, çeşitlendirme, stop mantığı, enflasyon ve kaldıraç.
- Backend: Sürümlü lesson/card modeli, önkoşullar, ilerleme, içerik yayınlama ve A/B varyantı.
- İstemciler: Offline okunabilir kartlar, kaldığı yerden devam, erişilebilir grafik örnekleri.
- Ödül: Sayfayı açmaya değil, kısa bilgi kontrolü ve aralıklı tekrar başarısına.

### 2. Mastery tabanlı quiz ve aralıklı tekrar

- Etki: Çok yüksek
- Karmaşıklık: Yüksek
- Abuse riski: Orta
- Backend: Soru havuzu, sunucu taraflı varyant sıralama, attempt ledger, soru bazlı mastery ve tekrar tarihi. Aynı sorudan tekrar XP kazanımı kademeli olarak sıfırlanır.
- İstemciler: Cevaptan sonra neden doğru/yanlış açıklaması; hız bonusu ve sonsuz tekrar puanı yok.
- Ölçüm: 7/30 gün gecikmeli bilgi koruma testi; yalnız anlık quiz skoru başarı sayılmaz.

### 3. Kişiselleştirilmiş öğrenme yolu

- Etki: Yüksek
- Karmaşıklık: Yüksek
- Abuse riski: Düşük
- Girdiler: Açık kullanıcı hedefi, zayıf mastery konuları, takip edilen varlık sınıfları ve portföyün yoğunlaşma/kur riski. Gerçekleşen kâr/zarar öneri girdisi değildir.
- Backend: Açıklanabilir kural motoru ile başlanır; yeterli veri oluşmadan kara kutu model kullanılmaz.
- İstemciler: “Bunu neden görüyorum?” açıklaması ve kişiselleştirmeyi kapatma seçeneği.

### 4. Portföy disiplin görevleri

- Etki: Çok yüksek
- Karmaşıklık: Orta
- Abuse riski: Orta
- Görevler: Yoğunlaşma uyarısını incele, yatırım tezini yaz, maksimum risk bütçesi belirle, aylık portföy kontrolü yap, ücret/temettü kayıtlarını tamamla.
- Backend: Portföy ledger'ından doğrulanan fakat al/sat yapmayı gerektirmeyen kurallar; görev snapshot'ı ve tamamlanma kanıtı.
- Kural: “İşlem yap”, “daha çok kazan” veya “tahmini doğru bil” görevi kesinlikle yok.

### 5. Haftalık öğrenme özeti ve geri kazanılabilir seri

- Etki: Yüksek
- Karmaşıklık: Orta
- Abuse riski: Düşük
- Deneyim: Öğrenilen konular, tutarlı davranışlar ve gelecek hafta tek odak. Ayda bir kez, kısa bir telafi dersiyle seri korunabilir; ücretli veya sınırsız streak freeze yok.
- Backend: Haftalık snapshot ve bildirim tercihleri.
- İstemciler: Uygulama içi özet; push yalnız açık izin ve sessiz saat kurallarıyla.

Faz 2 çıkış koşulları: öğrenme görevleri XP'nin çoğunluğunu oluşturuyor; gecikmeli bilgi koruma metriği kontrol grubundan yüksek; portföy görevleri işlem sıklığını yapay biçimde artırmıyor.

## Faz 3 — Sezon, süreç ligi ve gizlilik güvenli sosyal katman (14–24 hafta)

### 1. 6–8 haftalık sezonlar ve beceri ligleri

- Etki: Yüksek
- Karmaşıklık: Yüksek
- Abuse riski: Yüksek
- Sıralama: Öğrenme mastery artışı, disiplin görevleri, tutarlılık ve doğrulanmış katkı. Ham toplam XP yerine haftalık tavanlı sezon puanı.
- Backend: Season, division, enrollment, score snapshot, promotion/relegation ve audit tabloları; minimum aktivite ve risk incelemesi.
- Gizlilik: Varsayılan takma ad; portföy değeri, pozisyon ve getiri asla gösterilmez.

### 2. Süreç odaklı kâğıt portföy

- Etki: Çok yüksek
- Karmaşıklık: Çok yüksek
- Abuse riski: Yüksek
- Puan: Emirden önce tez, risk bütçesi ve invalidation koşulu; sonrasında plana uyum ve retrospektif. Getiri yalnız kişisel geri bildirimdir, lig puanı değildir.
- Backend: Ayrı paper ledger, tarihsel fiyat yürütme, piyasa saatleri, spread/slippage, ileriye bakma engeli ve immutable karar snapshot'ı.
- İstemciler: Gerçek portföyle açık görsel ayrım; “yatırım tavsiyesi değildir” ve eğitim amacı.

### 3. Gizlilik güvenli çalışma grupları

- Etki: Orta–yüksek
- Karmaşıklık: Yüksek
- Abuse riski: Yüksek
- Özellik: 3–8 kişilik opt-in gruplar, ortak ders hedefi, yalnız tamamlanma yüzdesi ve öğrenme serisi.
- Güvenlik: Takma ad, blokla/şikâyet et, yaş/ülke politikası, rate limit, moderasyon kuyruğu; pozisyon, para ve iletişim bilgisi paylaşımı varsayılan kapalı.

### 4. Açıklama katkısı ve akran değerlendirmesi

- Etki: Orta
- Karmaşıklık: Yüksek
- Abuse riski: Yüksek
- Özellik: Kullanıcı bir kavramı kendi cümlesiyle açıklar; yapılandırılmış rubrikle kör akran değerlendirmesi.
- Ödül: Gönderi sayısına değil, moderasyon sonrası faydalılık ve değerlendirici tutarlılığına; karşılıklı oy halkaları tespit edilir.

### 5. Sezonluk koleksiyon ve kozmetik statü

- Etki: Orta
- Karmaşıklık: Orta
- Abuse riski: Düşük–orta
- Ödül: Profil çerçevesi, öğrenme unvanı ve rozet vitrini; finansal ayrıcalık, daha yüksek risk limiti veya manipülatif kıtlık yok.
- Backend/istemci: Entitlement ledger, sezon arşivi ve tüm platformlarda aynı kozmetik render sözleşmesi.

Faz 3 çıkış koşulları: lig puanının getiriyle korelasyonu ürün kuralı gereği yok; moderasyon SLA'ı sağlanıyor; paper portföyde ileriye bakma ve yeniden oynatma açıkları kapalı; sosyal özellikler açık rıza ve gizlilik denetiminden geçmiş.

## XP ekonomisi

- Ödül sırası: mastery ve aralıklı tekrar > risk-disiplini > plan/retrospektif > basit check-in.
- Günlük XP'nin bir üst sınırı, her etkinlik ailesinin ayrı azalan getirisi ve sunucu taraflı kural sürümü olmalı.
- Check-in serisi küçük kolaylaştırıcıdır; toplam ekonomiyi belirlememeli. Uzun seri kaybı cezaya dönüşmemeli.
- Rozet XP'si bir sonraki rozeti tetikleyen sonsuz döngüye girmemeli; mevcut seviye rozetlerindeki sıfır XP yaklaşımı korunmalı.
- Silinip yeniden yaratılan favori/alarm, aynı kaynak kimliği için tekrar ödül üretmemeli.
- Quiz doğruluğu için ilk deneme ve gecikmeli tekrar ödüllendirilir; cevap hızı, sürekli tekrar ve soru ezberi ödüllendirilmez.
- XP geri alma gerektiğinde kayıt silinmez; immutable ledger'a ters kayıt yazılır.

## Önerilen teknik omurga

1. Domain eylemi aynı transaction içinde bir `gamification_outbox` kaydı üretir.
2. Worker sürümlü kural kataloğuna göre eligibility ve anti-farm kontrolü yapar.
3. `xp_ledger` benzersiz idempotency anahtarıyla ödül veya ters kayıt yazar.
4. Görev/mastery/season projection'ları ledger'dan tekrar üretilebilir.
5. WebSocket/push bildirimi transaction commit'inden sonra çıkar.
6. API her ödülle `reason`, `rule_version`, `awarded_xp`, `daily_cap_remaining` ve güncel projection döndürür.

Temel yeni varlıklar: `gamification_rules`, `gamification_events` genişletmesi, `xp_ledger`, `quest_instances`, `quest_progress`, `lessons`, `quiz_items`, `learning_attempts`, `mastery_state`, `daily_plans`, `portfolio_discipline_snapshots`, `seasons`, `season_scores`, `paper_portfolios`, `paper_orders`, `privacy_preferences`, `reports`.

## Yayın ve deney planı

- Her faz önce çalışanlar/test hesapları, sonra %5, %25 ve %100 feature flag yayınıyla açılır.
- A/B testinin birincil metriği yalnız günlük kullanım değil, 4 haftalık anlamlı öğrenme günleri ve gecikmeli bilgi korumadır.
- Her deneyde notification opt-out, görev terk oranı, işlem sıklığı, şüpheli event oranı ve destek şikâyeti guardrail olarak izlenir.
- Bir özellik D7'yi artırıp riskli işlem sıklığını veya farm davranışını artırıyorsa başarısız sayılır.
- Faz 3 sosyal özellikleri ayrı tehdit modeli, gizlilik incelemesi, moderasyon playbook'u ve kill switch olmadan açılmaz.
