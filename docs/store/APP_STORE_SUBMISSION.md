# TSC Forex — App Store Yayın Dokümanı (iOS)

Bu doküman TSC Forex'in güncel iOS sürümünü App Store Connect'e hazırlamak, otomatik yenilenen abonelikleri göndermek ve App Review risklerini en aza indirmek için tek kaynak olarak kullanılmalıdır. ASO alanlarının authoritative kaynağı `docs/store/aso/metadata.json` dosyasıdır; Apple ekranında sonradan yapılan her değişiklik bu dosyaya da işlenmelidir.

> **Garanti sınırı:** Hiç kimse “sıfır red” garantisi veremez; nihai karar Apple App Review'a aittir ve kurallar değişebilir. Aşağıdaki **Gönderim Kapıları** tamamlanmadan build göndermeyin. Bir kapı doğrulanmadıysa dokümandaki “hazır” ifadesi kanıt sayılmaz.

## 0. Mevcut yayın durumu — 1.1.2 (9) Waiting for Review

- App Store Connect'te Türkçe, İngilizce (ABD), Almanca, İtalyanca, İspanyolca (İspanya) ve Arapça için ad, alt başlık, promosyon metni, açıklama, anahtar kelimeler ve sürüm notları arama niyetine göre güncellendi.
- Türkçe ana değer önerisi `TSC Forex: Altın & Döviz` ve `Canlı Kur, Alarm ve Portföy` olarak konumlandırıldı; diğer diller birebir çeviri yerine yerel sorgu niyetine göre yazıldı.
- İlk üç görsel için canlı fiyat, etkileşimli grafik ve alarm sırası belirlendi. Çalışan `Yeni Ekran Görüntüleri` Product Page Optimization testi bitmeden ikinci test açılmadı.
- Favoriye ekleme, başarılı fiyat alarmı ve başarılı kaynaklı araştırma gibi gerçek değer anlarından sonra Apple'ın yerel değerlendirme penceresini kontrollü isteyen akış eklendi. İstek ödül, ödeme veya olumlu görüş şartına bağlı değildir.
- `1.1.2 (9)` Release archive başarıyla üretildi, App Store Connect'e yüklendi, processing tamamlandı ve sürüme bağlandı.
- Build, altı dildeki metadata değişiklikleri ve güncel App Review notları aynı submission içinde 30 Temmuz 2026'da gönderildi. App Store Connect'teki authoritative durum `Waiting for Review`'dur; onay sonrası otomatik yayın seçilidir.

### App Store Connect'te tamamlananlar

- Uygulama kaydı: Apple ID `6790740484`, bundle ID `talhacan.tscforex`, sürüm `1.0`.
- App adı `TSC Forex`, subtitle `Piyasa ve Portföy Analizi`, primary category `Finance`, secondary category `News` olarak kaydedildi.
- Yaş derecelendirme formu mevcut kapsam için dolduruldu; hesaplanan global sonuç `4+`.
- Subscription group `premium_monthly` oluşturuldu; Group ID `22243838`.
- Dört aylık ürün Türkiye'de kullanılabilir, fiyatları ve Türkçe localization'ları kayıtlı; Build 6 ile birlikte durumları `Waiting for Review`.
- App Store Server Notifications V2 için production ve sandbox URL'leri aynı HTTPS endpoint'e kaydedildi.
- Billing Grace Period bilinçli olarak **OFF** bırakıldı; backend ve test matrisi grace-period entitlement davranışıyla doğrulanmadan açılmamalıdır.
- Önceki `1.0 (4)` gönderimi, fiziksel cihazda bulunan kullanım hatalarını düzeltebilmek için 20 Temmuz 2026'da geri çekildi; eski submission `Removed`, app version geçici olarak `Developer Rejected` durumuna geçti ve Build 4 sürümden ayrıldı.
- Düzeltmeleri içeren signed `1.0 (5)` archive Apple'a yüklendi, processing tamamlandı ve sürüm `1.0` için seçildi.
- iOS `1.0 (5)` app version, `premium_monthly` subscription group ve dört auto-renewable subscription aynı yeni submission'a eklendi.
- Dört subscription'ın **Review Information** görseli Apple'a yüklendi. Bu görsel storefront'ta yayınlanan pazarlama görseli değil, App Review ekibinin satın alma yolunu doğrulaması içindir.
- Altı öğe 20 Temmuz 2026 10:10'da App Review'a gönderildi. Submission ID: `35107ec9-3de3-4f01-b22a-b4f9e72e658f`. Uygulama sürümü, grup ve dört ürünün her biri Apple ekranında `Waiting for Review` olarak doğrulandı.
- Apple 21 Temmuz 2026'da `1.0 (5)` sürümünü Guideline **5.1.1(v)** nedeniyle reddetti: hesap temelli olmayan piyasa içeriği kayıt/giriş duvarının arkasındaydı. Elite Aylık ayrıca Guideline **2.3.2** kapsamında reddedildi; isteğe bağlı tanıtım görseli uygulama ikonuyla aynıydı. Diğer üç subscription ile subscription group, ilişkili app version reddedildiği için `Other` sonucu aldı.
- Elite Aylık'ın storefront/win-back için kullanılan isteğe bağlı tanıtım görseli kaldırıldı. App Review satın alma yolunu gösteren **Review Information screenshot** korunmuştur; bu iki görsel alanı birbirine karıştırılmamalıdır.
- Red düzeltmelerini içeren signed `1.0 (6)` archive 22 Temmuz 2026'da Apple'a yüklendi, processing tamamlandı ve sürüm `1.0` için seçildi. Güncellenen Review Notes misafir erişimini, hesap gerektiren alanları ve Elite metadata düzeltmesini açıklar.
- `1.0 (6)` app version, dört auto-renewable subscription ve `premium_monthly` subscription group aynı submission'a yeniden eklendi. Submission ID `35107ec9-3de3-4f01-b22a-b4f9e72e658f` korunarak altı öğenin tamamı 22 Temmuz 2026'da yeniden gönderildi ve App Store Connect'te `Waiting for Review` olarak doğrulandı.
- Apple `1.0 (6)` sürümünü onayladı; 26 Temmuz 2026 App Store Connect kontrolünde sürüm `Ready for Distribution` durumundadır. Önceki red, yeni bir açık red değildir.
- `1.1 (7)` için signed Release archive alındı ve Xcode ile App Store Connect'e başarıyla yüklendi. Processing tamamlandı, Build 7 sürüme bağlandı; altı dil, Arapça RTL, portföy donut grafiği ve önceki review düzeltmeleri Review Notes'a işlendi.
- Product Page Optimization içindeki **Yeni Ekran Görüntüleri** treatment'ı Guideline **2.3.3 — Accurate Metadata** nedeniyle reddedildi: yedi adet 13 inç iPad görseli yalnız giriş ekranını gösteriyor ve uygulamanın temel kullanımını temsil etmiyordu. Eski görseller kaldırıldı; temiz 13 inç iPad simülatöründe çalışan güncel build'den alınan gerçek **2064×2752 Piyasalar** ekranı yüklendi ve treatment yeniden incelemeye gönderildi. PPO submission `c4dae2d5-1b6d-45fa-ae03-624ee4b730de` 26 Temmuz 2026 23:54 itibarıyla `Waiting for Review` durumundadır.
- `1.1 (7)` app version 27 Temmuz 2026'da ayrı submission `57ab4e70-1fc0-4bc0-b058-f20ff8ff8d7c` ile App Review'a gönderildi ve App Store Connect'te `Waiting for Review` olarak doğrulandı. Onay sonrası otomatik yayın ve bütün kullanıcılara anında dağıtım seçilidir.
- Uygulama indirme fiyatı Türkiye baz ülke olarak `₺0,00` ayarlandı; ilk sürüm yalnız Türkiye storefront'unda, iPhone/iPad için public dağıtıma açıldı. Apple Silicon Mac, Apple Vision Pro ve Apple School Manager indirimi kapalı bırakıldı.
- App Review iletişim alanları ve production login'i doğrulanmış ayrı demo hesabı App Store Connect'in güvenli alanlarında kaydedildi.
- Exact `1284×2778` iPhone 6.5-inch ve `2064×2752` iPad 13-inch screenshot'ları sürüme yüklendi.
- App Privacy URL'si ile 10 veri türünün amaç/linked/tracking cevapları kaydedildi; Apple'ın submission validation kontrolü privacy/copyright/content-rights zorunlu alanları için geçti. Bu, beyanların hukuki doğruluğunu Apple'ın onayladığı anlamına gelmez; sorumluluk Account Holder'dadır.
- DeviceCheck ve Sign in with Apple anahtarları repository dışında GitHub Actions secret'larına kaydedildi; production pod'larında dosya/KID varlığı doğrulandı.
- Build 5, kabloyla bağlı iPhone 14 Pro Max'e Xcode üzerinden signed Release olarak kuruldu ve uygulamanın cihazda başlatıldığı Xcode'da doğrulandı. Bu kurulum kanıtı, aşağıdaki uçtan uca manuel test matrisindeki açık maddeleri tek başına PASS yapmaz.
- `26f2abeb7532078add0ed6453c69d6e2785deb0f` kod commit'inin `main` CI/CD çalışması (`29882173748`) test, image build, Kubernetes rollout ve production smoke testlerinden geçti. Canlı API aynı commit'i raporluyor; API'nin database/Redis bağımlılıkları `up`, dört App Store planı ve device-based free trial `ready=true`. API, web, privacy, terms ve support HTTPS kontrolleri HTTP 200 döndü.

### Build 5 düzeltme kapsamı ve teknik doğrulama

- Ana sayfadaki bildirim kutusu global tam ekran sunuma taşındı; bozuk tab/navigation yığını ile boş uyarı ekranı kaldırıldı, yükleme-hata-yeniden deneme durumları eklendi.
- Günlük Plan'ın eksik/null `new_badges` cevabında decode hatası düzeltildi ve akış üç açık adıma sadeleştirildi.
- Karar Günlüğü, öğrenme ve görev akışları son kullanıcıya amaç/sonraki adımı açıklayacak şekilde genişletildi; paper-decision butonları çalışan üç aşamalı akışa bağlandı.
- Alarmlara düzenleme eklendi; tetiklenen tek seferlik alarm transaction içinde atomik olarak kapatılıyor ve aktif listeden kaldırılıyor.
- Portföy performansı, varlık detay grafiğiyle aynı etkileşim modelini kullanan dönem ve metrik seçilebilir geçmiş grafiğine geçirildi.
- Otuz rozetin tamamı için mevcut değer, hedef, yüzde ilerleme, kalan koşul ve kazanıldıysa kazanım tarihi gösteriliyor.
- Backend `go test ./...`, web lint/build, plist/privacy manifest kontrolleri ve signed Release archive/upload tamamlandı. App Store Connect processing ile altı öğeli submission ayrıca doğrulandı.

### Build 6 — App Review red düzeltmesi

- Uygulama artık temiz kurulumda doğrudan **Piyasalar** ekranını açar; kayıt veya kişisel bilgi vermeden canlı fiyat listesi, arama, Altın/Döviz/Kripto/Hisse kategorileri, varlık detayları, istatistikler, dönüşüm bilgileri ve bütün grafik dönemleri kullanılabilir.
- Favoriler, bildirim merkezi, fiyat alarmı, portföy, ödül/öğrenme, hesap ayarları, abonelik ve Derin Araştırma hesap temelli kalır. Misafir bu alanlardan birini seçtiğinde neden hesap gerektiğini açıklayan ekran ve isteğe bağlı giriş/kayıt akışı görür; kamusal piyasa içeriğine dönmesi engellenmez.
- Varlık detayındaki favori, alarm ve Derin Araştırma eylemleri misafir kullanıcıyı sessiz hata veya devre dışı buton yerine giriş akışına yönlendirir.
- Çıkışta/fresh guest başlangıcında favori ve bildirim gibi account-scoped cihaz önbelleği temizlenir; public fiyat websocket/REST akışı token olmadan çalışmaya devam eder.
- `1.0 (6)` Debug simulator build'i başarıyla derlendi; yeni, oturumsuz iPhone 17 Pro Max simulator kurulumunda giriş duvarı olmadan canlı Piyasalar ekranının açıldığı görsel olarak doğrulandı.
- `1.0 (6)` arm64 Release archive başarıyla alındı, Xcode upload tamamlandı ve App Store Connect'te işlenmiş Build 6 seçilerek yeniden gönderildi.

### 1.1 (7) — Altı dil ve portföy dağılımı

- Uygulama Türkçe, İngilizce, Arapça, Almanca, İtalyanca ve İspanyolca destekler. İlk açılışta desteklenen sistem dili seçilir; Ayarlar → Uygulama dili alanından uygulama içinde değiştirilebilir.
- Arapça için sağdan sola yerleşim ve locale tabanlı sayı/tarih biçimlendirmesi etkindir. İngilizce ve Arapça görünümler iPhone simülatöründe görsel olarak doğrulanmıştır.
- Xcode localization export ile bulunan 323 görünür kaydın 295 anlamlı kullanıcı metni her hedef dilde karşılanır; kalan 28 kayıt sayı, sembol veya teknik bundle anahtarıdır.
- Portföy dağılımı, eski doğrusal listeden ilk yedi pozisyonu ve gerektiğinde `Diğer` dilimini gösteren erişilebilir donut grafik + yüzde listesine dönüştürüldü. Dar genişlikte grafik ve lejant dikey yerleşir.
- `1.0 (5)` reddine neden olan kamusal piyasa içeriğinin giriş duvarı arkasında kalması tekrar edilmedi: misafir Piyasalar, arama, kategoriler, varlık detayları ve grafiklere erişmeye devam eder.
- Guideline 2.3.2 kapsamında kaldırılan Elite storefront promotional image yeniden eklenmemelidir. IAP Review Information görselleri bundan farklıdır ve korunur.
- `1.1 (7)` signed archive upload/processing, build seçimi, metadata kaydı ve `Submit for Review` adımları tamamlandı; authoritative durum `Waiting for Review`'dur.

### İnceleme ve canlı satış öncesi takip edilecek açık riskler

| Engel | Güncel durum | Kapatma koşulu / yetki sahibi |
|---|---|---|
| Paid Apps Agreement | Son doğrulanmış durumda **New**; submission'ın kabul edilmesi ücretli sözleşmenin satış için aktif olduğu anlamına gelmez | Account Holder yasal kişi bilgisini doğrular; ücretli uygulamalar sözleşmesini imzalar; vergi ve banka bilgilerini onaylatır. Bu işlem geliştirici tarafından varsayılamaz veya onun adına tasdik edilemez. |
| Content Rights | Apple'ın zorunlu alan kontrolü geçti; seçilen cevabın dayandığı piyasa/haber/veri yeniden dağıtım kanıtları teknik çalışmada görülmedi | Hak sahibi/provider sözleşmeleri saklanır; reviewer isterse sunulur ve Account Holder beyanın doğruluğunu korur. |
| App Privacy | 10 veri türü ve privacy URL kayıtlı; submission validation geçti | Production veri akışı değişirse form ve politika release'ten önce yeniden güncellenir. |
| Review erişimi | İletişim alanları ve production login'i HTTP 200 ile doğrulanmış ayrı demo credentials kaydedildi | Account Holder review süresince iletişim bilgilerini erişilebilir tutar; demo hesabı silmez/değiştirmez. |
| Copyright / yasal sahip | Apple'ın zorunlu alan kontrolü geçti; girilen yasal unvanın dayanağı teknik çalışmada bağımsız doğrulanmadı | Account Holder gerçek hak sahibi/yıl bilgisinin doğruluğunu korur; değişiklik varsa Apple'a bildirilir. |
| Screenshots / iPad | PPO'daki login-only 13 inç iPad seti Guideline 2.3.3 nedeniyle reddedildi; eski set kaldırılıp güncel build'in gerçek 2064×2752 Piyasalar ekranıyla yeniden gönderildi. App version görselleri ve dört subscription Review Information görseli yüklendi. | Review görseli Account Holder'ın açık onayıyla yalnız Apple'a gönderildi; repository'ye eklenmez. Gelecek sürümlerde temel işlevi göstermeyen login-only set kullanılmaz. |
| Signed build | `1.0 (6)` onaylandı ve `Ready for Distribution`; `1.1 (7)` signed archive upload/processing ve build seçimi tamamlanarak submission `57ab4e70-1fc0-4bc0-b058-f20ff8ff8d7c` içinde `Waiting for Review` durumuna geçti. | App Review mesajları izlenir; Apple kararı gelmeden sürüm “yayında” sayılmaz. |
| Apple anahtarları | DeviceCheck ve Sign in with Apple `.p8`/Key ID değerleri GitHub Actions ve Kubernetes production secret'larında mevcut; pod içi dosya/KID kontrolleri geçti | Gerçek DeviceCheck free-trial claim ve Sign in with Apple revoke senaryolarının fiziksel cihaz/sandbox kanıtları saklanır. |
| StoreKit uçtan uca test | Purchase/restore/plan değişimi/refund/revoke matrisi tamamlanmadı | Bölüm 9.1'deki matris kanıtlarıyla PASS olur. |

Submission Apple'a ulaşmıştır; bundan sonraki authoritative durum App Store Connect → App Review ekranıdır. Apple kararı gelmeden “yayında” denmez. Özellikle sözleşme, vergi, banka, içerik hakkı ve yasal kimlik cevapları Account Holder'ın hukuki beyanıdır.

---

## 1. Uygulama kimliği ve sınıflandırma

| Alan | Değer |
|---|---|
| App Name | **TSC Forex** |
| Apple ID | `6790740484` |
| Subtitle (en fazla 30 karakter) | **Piyasa ve Portföy Analizi** |
| Primary Language | Turkish |
| SKU | `tscforex` |
| Bundle ID | `talhacan.tscforex` |
| Primary Category | Finance |
| Secondary Category | News |
| Fiyat | Ücretsiz indirme; uygulama içi otomatik yenilenen abonelikler var |
| İlk storefront | Yalnız Türkiye; public iPhone/iPad dağıtımı |
| IAP | 4 aylık auto-renewable subscription |
| Reklam / tracking | Yok |
| Copyright | App Store Connect'in zorunlu alan kontrolü geçti; girilen hak sahibi/yıl bilgisinin hukuki doğruluğu Account Holder sorumluluğunda |

İlk yayın için yalnız hukuki/operasyonel destek verilebilen storefront'ları seçin; Türkçe ürün, destek ve yasal metinlerle Türkiye başlangıcı en düşük riskli seçenektir. Başka ülkelere açılmadan önce fiyat, vergi, tüketici hukuku, privacy ve metadata localization'ı tamamlayın.

### Content Rights / piyasa verisi

App Store Connect'teki Content Rights zorunlu alanı submission validation'dan geçmiştir; ancak teknik çalışma seçilen cevabın dayandığı lisansları bağımsız olarak doğrulamamıştır. BIST, kripto, döviz, haber/web kaynakları ve diğer piyasa verilerini uygulamada yeniden göstermeye izin veren sağlayıcı sözleşmeleri/lisansları saklanmalıdır. “Kamuya açık URL'den erişilebiliyor” yeniden dağıtım hakkı anlamına gelmez. Reviewer isterse veri sağlayıcı adları ve kullanım hakkı kanıtı sunulabilmelidir; marka/logolar için de aynı kontrol geçerlidir.

### Yaş derecelendirmesi

App Store Connect'in yaş derecelendirme formu mevcut kapsamla tamamlandı ve global sonuç `4+` hesaplandı. Bu, ürünün çocuklara yönelik olduğu anlamına gelmez. Derin araştırma, finansal içerik veya kullanıcı etkileşimi değişirse sorular yeniden cevaplanmalı; sonucu elle sabit tutmaya çalışılmamalıdır.

---

## 2. App Store metadata — kopyalanabilir metinler

### 2.1 Promotional Text — en fazla 170 karakter

```text
Piyasaları tek ekranda izle; alarm, portföy ve kaynaklı derin araştırmayla bağlamı gör. Bilgilendirme amaçlıdır, yatırım tavsiyesi değildir.
```

### 2.2 Description

```text
TSC Forex; altın, döviz, kripto para ve BIST hisselerini tek uygulamada takip etmenizi sağlar. Canlı fiyatları ve geçmiş grafikleri inceleyebilir, fiyat alarmı kurabilir, favorilerinizi ve portföylerinizi yönetebilirsiniz.

DERİN ARAŞTIRMA
Seçtiğiniz varlık için çoklu zaman dilimleri, ilgili piyasalar, makroekonomik gelişmeler ve jeopolitik riskleri kapsayan kaynaklı araştırma oluşturun. Üretilen çıktılar olasılıksaldır; yanlış, eksik veya güncelliğini yitirmiş olabilir.

ÖNE ÇIKAN ÖZELLİKLER
• Altın, döviz, kripto ve BIST için canlı fiyatlar
• Yakınlaştırılabilir geçmiş fiyat grafikleri
• Favoriler ve fiyat alarmları
• Çoklu portföy, kâr/zarar ve portföy sağlık görünümü
• Kaynak bağlantılı derin piyasa araştırması ve araştırma arşivi
• Günlük plan, öğrenme, XP, seviye, rozet ve süreç odaklı paper-trade özellikleri
• Bildirim merkezi ve ana ekran widget'ı
• E-posta, Apple, Google ve Passkey ile güvenli giriş

ABONELİKLER
Başlangıç, Pro, Uzman ve Elite aylık abonelikleri; farklı derin araştırma, alarm, izleme listesi, portföy, sağlık analizi, dışa aktarma ve arşiv kotaları sunar. Kesin yerel fiyat ve dönem satın alma ekranında gösterilir. Abonelik, mevcut dönemin bitiminden en az 24 saat önce iptal edilmezse otomatik yenilenir. Satın almaları uygulama içinden geri yükleyebilir, aboneliğinizi Apple hesabınızdan yönetebilirsiniz.

TSC Forex alım-satım, aracılık veya yatırım danışmanlığı hizmeti sunmaz. Fiyatlar ve araştırma çıktıları yalnızca bilgilendirme/eğitim amaçlıdır ve yatırım tavsiyesi değildir. Veriler üçüncü taraf kaynaklardan gelir; gecikme veya hata içerebilir.

Kullanım Şartları: https://tsc.talhacan.com/terms
Gizlilik Politikası: https://tsc.talhacan.com/privacy
```

### 2.3 Keywords — en fazla 100 karakter

```text
altın,döviz,dolar,euro,kripto,bitcoin,borsa,bist,hisse,alarm,portföy,finans,AI,kur
```

### 2.4 What's New

```text
• Kaynak bağlantılı, çoklu zaman dilimli Derin Araştırma
• Çoklu portföy ve portföy sağlık analizi
• Gelişmiş öğrenme, görev, sezon ve paper-trade deneyimi
• Aylık Başlangıç, Pro, Uzman ve Elite planları
• Satın alma geri yükleme ve abonelik yönetimi
• Performans ve güvenlik iyileştirmeleri
```

### 2.5 URL'ler ve iletişim

| Metadata/yayın alanı | Değer |
|---|---|
| Support URL | `https://tsc.talhacan.com/support` |
| Marketing URL | `https://tsc.talhacan.com` |
| Privacy Policy URL | `https://tsc.talhacan.com/privacy` |
| Description + paywall Terms of Use | `https://tsc.talhacan.com/terms` |
| Support email | `destek@talhacan.com` |
| Privacy email | `gizlilik@talhacan.com` |

Tüm URL'ler giriş yapmadan, geçerli TLS sertifikasıyla, mobil Safari'de ve App Review ağı üzerinden erişilebilir olmalıdır. Boş sayfa, SPA 404'ü, bot engeli, ülke/IP engeli veya giriş duvarı red riskidir.

App Store Connect → App Information → License Agreement alanında Apple Standard EULA kullanılabilir; ayrıca yukarıdaki Kullanım Şartları bağlantısı description ve paywall'da kalmalıdır. Özel EULA girilecekse hukukçu onayı ve bütün storefront'larla uyum gerekir.

---

## 3. Auto-renewable subscriptions — kesin App Store Connect kurulumu

### 3.1 Subscription group

Tek grup oluşturun:

| Alan | Değer |
|---|---|
| Reference Name | `premium_monthly` |
| Group ID | `22243838` |
| Display Name (TR) | `TSC Forex Premium Planları` |
| App Store Connect durumu | Oluşturuldu |

Kullanıcı aynı grupta aynı anda yalnızca bir aktif ürüne sahip olur. Dört ürün aynı grup içinde, sundukları erişim miktarına göre farklı seviyelerde sıralanmalıdır.

### 3.2 Oluşturulan ürünler, Apple kimlikleri ve seviye sırası

| Level | UI adı | Product ID | Apple ID | Süre | Türkiye fiyatı | Durum |
|---:|---|---|---:|---|---:|---|
| 1 — en yüksek | Elite | `talhacan.tscforex.elite.monthly` | `6791924071` | 1 ay | ₺999,99 | Waiting for Review; promotional image kaldırıldı |
| 2 | Uzman | `talhacan.tscforex.expert.monthly` | `6791924799` | 1 ay | ₺499,99 | Waiting for Review |
| 3 | Pro | `talhacan.tscforex.pro.monthly` | `6791925225` | 1 ay | ₺249,99 | Waiting for Review |
| 4 — en düşük | Başlangıç | `talhacan.tscforex.plus.monthly` | `6791925442` | 1 ay | ₺149,99 | Waiting for Review |

Bu tablo hedef değil, App Store Connect'te kayıtlı gerçek price point ve ürün kimliklerini gösterir. Dört ürün yalnız Türkiye için available ve Türkçe localization'a sahiptir. Dört ürün, Build 6 ve subscription group ile aynı submission'a yeniden eklenmiştir; `Waiting for Review` durumları ürünlerin henüz satışta olduğu anlamına gelmez.

- Fiyatları uygulamada sabit metin olarak göstermeyin; `StoreKit.Product.displayPrice` ve StoreKit dönem bilgisini kullanın.
- Paywall'da App Store Connect tablosundaki sayılar dahi sabit metin olarak kullanılmaz; her zaman StoreKit'in döndürdüğü yerelleştirilmiş fiyat esas alınır.
- Dört ürün için Availability, Tax Category, fiyat ve Review Information screenshot kaydedildi. Türkiye dışına açılmadan önce o storefront'lara uygun localization ve fiyatlandırma tamamlanmalıdır.
- Introductory offer/free trial yapılandırılmadıkça metadata veya paywall'da “ücretsiz deneme” demeyin.
- İlk sürümle birlikte dört IAP'yi de aynı app version'a **Submit for Review** ile iliştirin. Sadece kodda görünmeleri yetmez.

App Store Connect'te kayıtlı Türkçe localization metinleri:

| Product ID | Display Name | Description |
|---|---|---|
| `talhacan.tscforex.plus.monthly` | Başlangıç Aylık | 15 AI araştırması ve temel premium kotalar |
| `talhacan.tscforex.pro.monthly` | Pro Aylık | 25 AI araştırması ve gelişmiş premium kotalar |
| `talhacan.tscforex.expert.monthly` | Uzman Aylık | 50 AI araştırması ve yüksek premium kotalar |
| `talhacan.tscforex.elite.monthly` | Elite Aylık | 100 AI araştırması ve maksimum premium kota |

### 3.3 Plan hakları ve sunucu sözleşmesi

| Hak | Free | Başlangıç | Pro | Uzman | Elite |
|---|---:|---:|---:|---:|---:|
| Derin araştırma | Kullanıcı **ve** cihaz başına ömür boyu 1 | 15/ay | 25/ay | 50/ay | 100/ay |
| Aktif alarm | 2 | 10 | 25 | 60 | 150 |
| İzleme listesi | 10 | 30 | 75 | 200 | 500 |
| Portföy | 1 | 2 | 5 | 15 | 50 |
| Portföy sağlık raporu | 0 | 2/ay | 5/ay | 15/ay | 40/ay |
| Dışa aktarma | 0 | 2/ay | 10/ay | 30/ay | 100/ay |
| Derin araştırma arşivi | Yok | 30 gün | 90 gün | 180 gün | 365 gün |

Kotalar hem paywall'da hem backend catalog'unda aynı olmalıdır. Plan adı tek başına yeterli değildir; kullanıcı satın almadan önce hangi değerin/kotanın açıldığını görmelidir. Aylık kullanım kotaları abonelik dönemiyle yenilenir; kullanılmayan haklar devretmez.

### 3.4 Paywall zorunlu içerik ve davranış

Satın alma düğmesinden **önce ve aynı ekranda** okunabilir biçimde gösterin:

- Seçili planın adı, tüm esas kotaları ve bir aylık erişim verdiği.
- StoreKit'ten gelen yerelleştirilmiş fiyat ve “aylık / otomatik yenilenir” ifadesi.
- Ödemenin Apple hesabından tahsil edileceği.
- Mevcut dönemin bitiminden en az 24 saat önce iptal edilmezse otomatik yenileneceği ve yenileme ücretinin Apple hesabından alınacağı.
- `Kullanım Şartları` ve `Gizlilik Politikası` için tıklanabilir bağlantılar.
- Görünür `Satın Almaları Geri Yükle` düğmesi.
- Görünür `Aboneliği Yönet` düğmesi veya Apple yönetim ekranı bağlantısı.
- Satın alma sürerken ilerleme; pending, cancelled, failed, verified ve unverified durumları için dürüst mesaj.
- Satın alma tamamlanmadan premium özellikleri açmama; yalnız doğrulanmış StoreKit 2 JWS/transaction + backend entitlement sonucuna güvenme.

Satın alma düğmesi yanıltıcı hiyerarşi, önceden seçilmiş trial, sahte geri sayım, kapatılamayan paywall veya satın almaya zorlayan sistem izni kullanmamalıdır. Ücretsiz piyasa görüntüleme ve mevcut ücretsiz haklar paywall tarafından kilitlenmemelidir.

### 3.5 Restore / manage / cancel / refund

- Restore: kullanıcı etkileşimiyle `AppStore.sync()`/StoreKit restore; uygulama açılışında otomatik credential prompt'u yok.
- Manage: StoreKit manage-subscriptions sheet veya `https://apps.apple.com/account/subscriptions`.
- Cancel: TSC Forex doğrudan Apple aboneliğini iptal etmez; Apple yönetim ekranına götürür.
- Refund: StoreKit refund request sheet veya `https://reportaproblem.apple.com/`; uygunluğa Apple karar verir.
- Plan değişimi: aynı group ranking nedeniyle yükseltme/düşürme/crossgrade Apple kurallarına göre gerçekleşir. Backend notification ve transaction state ile aynı sonuca ulaşmalıdır.

### 3.6 App Store Server Notifications V2

App Store Connect → App Information → App Store Server Notifications alanlarında:

| Ortam | URL |
|---|---|
| Production Server URL | `https://tsc-api.talhacan.com/api/v1/billing/apple/notifications` |
| Sandbox Server URL | `https://tsc-api.talhacan.com/api/v1/billing/apple/notifications` |
| Version | Version 2 |

Production deployment:

```text
APP_BILLING_ENABLED=true
APP_BILLING_BUNDLE_ID=talhacan.tscforex
APP_BILLING_ALLOW_SANDBOX=true
```

`ALLOW_SANDBOX=true`, TestFlight ve App Review sandbox işlemleri için gereklidir. Sunucu signed JWS zincirini/imzasını doğrulamalı; bundle ID ve environment bilgisini kaydetmeli; aynı notification'ı idempotent işlemeli; yalnız istemciden gelen plan adına güvenmemelidir. Apple API key gerekmemesi, imza/bundle/environment doğrulamasını atlama izni değildir. Notification URL `200` dönmeli ve tekrar teslimlerde yan etki üretmemelidir.

17 Temmuz 2026 durumu: production ve sandbox alanlarının ikisinde de yukarıdaki URL, Version 2 olarak yapılandırıldı. Dashboard'a URL yazılması tek başına doğrulama değildir; Apple'ın test bildirimi, imza doğrulama ve tekrar teslim idempotency testi ayrıca PASS olmalıdır.

### 3.7 Billing Grace Period

App Store Connect'te Billing Grace Period **OFF** durumundadır ve ilk gönderimde böyle kalacaktır. Backend, StoreKit UI ve Server Notifications akışı grace-period sırasında yetkiyi koruma davranışıyla uçtan uca doğrulanmadan bu ayar açılmaz. Bu nedenle ilk release test matrisinde `GRACE_PERIOD` zorunlu bir production entitlement senaryosu değildir; `DID_FAIL_TO_RENEW`/billing retry ve süre sonu davranışı yine test edilmelidir. Daha sonra grace period açılırsa privacy/terms/paywall değişmese bile entitlement testleri ve bu doküman yeniden güncellenir.

---

## 4. App Privacy / Privacy Nutrition Label

**Güncel durum:** Privacy Policy URL `https://tsc.talhacan.com/privacy` kaydedildi. Aşağıdaki 10 veri türü için amaçlar, kimliğe bağlantı ve tracking cevapları App Store Connect'te dolduruldu; Product Page Preview yalnız **Data Linked to You** gösteriyor ve tracking yok. Son `Publish` ekranındaki doğruluk/yasal uygunluk attestation'ı Account Holder tarafından verilmelidir.

### Genel cevaplar

- **Do you or your third-party partners collect data from this app?** → **Yes**
- **Data Used to Track You / Tracking** → **No**
- ATT istemi → Gerekmez; IDFA/reklam/ölçüm için çapraz uygulama-site takibi yoktur.

App Store Connect'te yalnız mevcut uygulama ve backend'in gerçekten topladığı verileri işaretleyin. Aşağıdaki tablo güncel ürün sözleşmesidir; kod/altyapı farklıysa formu koda göre düzeltmeden göndermeyin.

| Apple veri türü | Örnek | Amaç | Kimliğe bağlı | Tracking |
|---|---|---|:---:|:---:|
| Contact Info → Name | İsteğe bağlı görünen ad | App Functionality; Product Personalization | Evet | Hayır |
| Contact Info → Email Address | Hesap/giriş | App Functionality | Evet | Hayır |
| User Content → Photos or Videos | İsteğe bağlı profil fotoğrafı | App Functionality | Evet | Hayır |
| User Content → Other User Content | Alarm, favori, paper-trade tez/retrospektif, çalışma grubu takma adı | App Functionality; Product Personalization | Evet | Hayır |
| Financial Info → Other Financial Info | Kullanıcının girdiği varlık, miktar, maliyet ve portföy verisi | App Functionality | Evet | Hayır |
| Purchases → Purchase History | Ürün/plan, subscription/transaction state | App Functionality (entitlement, security/fraud prevention) | Evet | Hayır |
| Identifiers → User ID | TSC kullanıcı kimliği, sosyal giriş eşlemesi | App Functionality | Evet | Hayır |
| Identifiers → Device ID | APNs token ve tek yönlü cihaz claim hash'i | App Functionality (bildirim, security/fraud prevention) | Evet/takma adlı | Hayır |
| Usage Data → Product Interaction | Kota tüketimi, seçilen derin araştırma ve özellik kullanımı | App Functionality, Analytics* | Evet | Hayır |
| Diagnostics → Other Diagnostic Data | Hata, performans ve erişim günlükleri | App Functionality (service/security) | Evet (bazı güvenlik kayıtları hesap/istekle ilişkilendirilebilir) | Hayır |

`Analytics` yalnız veri gerçekten ürün analitiği için kullanılıyorsa işaretlenmelidir; yalnız quota/fonksiyon yürütme ise App Functionality seçin. Apple ödeme kartı/banka bilgisi geliştiriciye ulaşmıyorsa `Payment Info` geliştirici tarafından collected değildir. OpenAI'a seçilen varlık/piyasa bağlamı aktarımı uygulamanın/üçüncü taraf partnerin veri uygulamasına dahildir; kullanım ve saklama gerçeği App Privacy formuyla uyumlu olmalıdır.

### Derin araştırma veri paylaşımı — Guideline 5.1.2(i)

- İlk derin araştırmadan önce ayrı, anlaşılır ve zorlamasız açık izin alın.
- Ekran; üçüncü tarafın **OpenAI** olduğunu, gönderilen veriyi (seçilen varlık + ilgili piyasa bağlamı + hesaptan tek yönlü türetilen takma adlı güvenlik kimliği), amacı ve privacy/consent yönetimini belirtmelidir.
- Reddetmek piyasa izleme gibi AI dışı işlevleri engellememelidir.
- İzin Ayarlar'dan geri çekilebilmeli; geri çekme yeni AI aktarımlarını durdurmalıdır.
- AI isteğine e-posta, ad, avatar, parola, ödeme bilgisi veya kişisel portföy pozisyonu eklenmemelidir.
- AI raporunda kaynak bağlantıları ve “yatırım tavsiyesi değildir / hata içerebilir” uyarısı görünür kalmalıdır.

### Cihaz claim hash'i ve saklama

Ücretsiz derin araştırma hakkı kullanıcı **ve cihaz** başına ömür boyu birdir. Uygulamanın Keychain'de ürettiği rastgele, uygulamaya özel kurulum kimliği sunucuda tek yönlü özetlenir; donanım kimliği veya IDFA kullanılmaz. Takma adlı claim hash'i yalnız tekrar ücretsiz hak suistimalini önlemek için kullanılır; reklam, profil çıkarma veya cihazlar arası pazarlama takibi için kullanılmaz. Hesap silmede kullanıcı bağlantısı hemen kaldırılır. Hash, ücretsiz kullanım programı devam ettiği sürece amaçla sınırlı tutulur ve program sona erdiğinde silinir. Bu tek saklama açıklaması Privacy Policy, hesap silme ekranı ve Review Notes'ta aynı anlamla kullanılmalıdır.

### Üçüncü taraflar

Privacy Policy en az şunları açıklar: OpenAI (kullanıcı başlattığında AI), Apple StoreKit/App Store (ödeme ve abonelik), APNs/FCM (bildirim), Google/Apple (seçilirse giriş), barındırma/obje depolama ve piyasa veri sağlayıcıları. Her partner aynı/eşdeğer veri koruması sunacak sözleşmesel ve teknik tedbirlere tabi olmalıdır.

---

## 5. İzinler, entitlements ve Privacy Manifest

| Yetki/izin | Gerekçe | Gönderim kapısı |
|---|---|---|
| Push Notifications / `aps-environment` | Kullanıcının fiyat alarmı, hesap/güvenlik ve açıkça seçtiği öğrenme bildirimi | Sistem izni bağlam içinde istenir; ret halinde uygulama çalışır |
| Sign in with Apple | Google/social login sunulduğu için eşdeğer gizlilik dostu giriş | Capability, Service ID/key ve production callback doğrulanır |
| App Attest + DeviceCheck | Bir kullanıcı ve bir fiziksel cihaz için ömür boyu tek ücretsiz derin araştırma; sahte istemci ve yeniden kurulum suistimalini azaltma | App ID capability, Debug/Release environment, sunucu doğrulaması ve DeviceCheck key uçtan uca test edilir |
| Associated Domains | Passkey ve gerekiyorsa universal link | AASA production domain'de doğrulanır |
| App Groups | Widget ile gerekli veri paylaşımı | App ve widget entitlement aynı group olmalı |
| Photo picker | Kullanıcının seçtiği tek avatar | PHPicker kullanılıyorsa geniş Photos usage string gerekmez |
| Local Network | Yalnız geliştirme sunucusu | **Release build'de kaldırılmalı**; production kullanıcıya local network prompt'u çıkmamalı |

- Kamera, mikrofon, konum, kişiler, takvim ve tracking kullanılmıyorsa usage string/capability eklemeyin.
- Bildirimler yalnız kullanıcının alarmı, hesap/güvenlik ve açık seçtiği öğrenme bildirimleri içindir; pazarlama bildirimi açık opt-in olmadan gönderilmez.
- `PrivacyInfo.xcprivacy` hem app hem kullanılan üçüncü taraf SDK'lar/required-reason API'ler bakımından Archive üzerinde doğrulanmalıdır. Organizer → Generate Privacy Report çıktısını App Privacy formuyla karşılaştırın.
- App Attest entitlement'ı Debug'da `development`, TestFlight/App Store Release'te `production` olmalıdır. Apple Developer portalındaki `talhacan.tscforex` App ID için App Attest/DeviceCheck etkin değilse signed archive göndermeyin.
- Ücretsiz derin araştırma hakkı yalnız meşru iOS uygulamasından gelen App Attest assertion'ı ve Apple DeviceCheck cihaz durumu doğrulandıktan sonra açılır. Web/localStorage kimliği fiziksel cihaz kanıtı sayılmaz; web yalnız doğrulanmış ücretli entitlement'ı kullanır.
- Release archive'da ATS arbitrary load olmamalı; bütün production istekleri HTTPS olmalı.
- `ITSAppUsesNonExemptEncryption = NO` yalnız gerçekten standart/istisna kapsamındaki HTTPS/TLS kullanılıyorsa archive Info.plist'inde bulunmalıdır.

### 5.1 Apple `.p8` anahtarları ve production secret kapısı

İki ayrı sunucu anahtarı gerekir; ikisi de Git'e, Xcode target'ına, uygulama bundle'ına, log'a veya dokümana konmaz:

| Amaç | Apple bilgisi | GitHub Actions secret | Pod içi yol / config |
|---|---|---|---|
| DeviceCheck sunucu doğrulaması | DeviceCheck yetkili Key ID + `.p8` | `APP_BILLING_DEVICE_CHECK_KEY_ID`, `APP_BILLING_DEVICE_CHECK_PRIVATE_KEY` | `APP_BILLING_DEVICE_CHECK_PRIVATE_KEY_PATH=/etc/apple-auth/devicecheck.p8` |
| Sign in with Apple token revoke | Sign in with Apple yetkili Key ID + `.p8` | `APP_OAUTH_APPLE_KEY_ID`, `APP_OAUTH_APPLE_PRIVATE_KEY` | `APP_OAUTH_APPLE_PRIVATE_KEY_PATH=/etc/apple-auth/sign-in-with-apple.p8` |

Deployment'ta ayrıca `APP_BILLING_DEVICE_CHECK_ENABLED=true`, `APP_OAUTH_APPLE_REVOKE_CLIENT_ID=talhacan.tscforex` ve doğru Apple Team ID doğrulanır. CI secret değerleri satır sonları korunmuş gerçek `.p8` içeriği olmalıdır; base64/kaçış yöntemi workflow'un beklediği formatla aynı değilse pod içindeki dosya geçersiz olur. Secret adını görmek PASS sayılmaz: rollout sonrası pod'da dosyanın varlığı/izinleri, KID eşleşmesi, gerçek DeviceCheck çağrısı ve Apple token revoke yanıtı hassas değeri yazdırmadan doğrulanır.

---

## 6. Giriş, hesap ve hesap silme

### Sign in with Apple — Guideline 4.8

Google ile giriş sunulduğu için Apple ile Giriş görünür, eşdeğer ve çalışır olmalıdır. E-posta ve Passkey ayrıca sunulabilir. Apple private relay e-postaları desteklenmeli; hesap bağlama aynı kişiye istemeden ikinci hesap açmamalıdır.

### Hesap silme — Guideline 5.1.1(v)

- Ayarlar → Tehlikeli Bölge → `Hesabı Sil` kolay bulunur olmalıdır.
- Silme bütün hesabı kaldırır; yalnız deaktivasyon yeterli değildir.
- Son onay ekranı silinecek verileri, geri alınamazlığı ve Bölüm 4'te tek kez tanımlanan cihaz claim saklama istisnasını açıklar.
- **Aktif auto-renewable subscription varsa:** “Hesabı silmek aboneliği iptal etmez; Apple faturalandırması sürebilir” uyarısı ve `Aboneliği Yönet` bağlantısı silmeden önce gösterilir. Kullanıcı isterse yine hemen silebilir.
- Silme; favori, alarm, portföy, derin araştırma arşivi, bildirim, cihaz push tokenı, avatar, kimlik/passkey eşlemeleri ve oyunlaştırma verilerini kaldırır.
- Sign in with Apple hesabı için Apple token revoke işlemi uygulanmalıdır.

#### Sign in with Apple revoke sunucu kurulumu — gönderimden önce zorunlu

Hesap silme endpoint'i Apple ile bağlı hesaplarda iOS'un yeniden doğrulama sırasında aldığı tek kullanımlık authorization code'u Apple token endpoint'inde değiştirir, dönen refresh token'ı revoke eder ve ancak bundan sonra hesabı siler. Private key uygulamaya veya Git'e konmaz.

1. Apple Developer → Certificates, Identifiers & Profiles → Keys altında **Sign in with Apple** yetkili bir key oluşturun; Key ID'yi güvenli envantere kaydedin ve `.p8` dosyasını yalnız bir kez indirin.
2. Gerçek Key ID'yi GitHub Actions `APP_OAUTH_APPLE_KEY_ID`, `.p8` içeriğini `APP_OAUTH_APPLE_PRIVATE_KEY` secret'ına kaydedin. Secret değerlerini terminal çıktısında veya CI log'unda göstermeyin.
3. Production config'te `APP_OAUTH_APPLE_REVOKE_CLIENT_ID=talhacan.tscforex` ve `APP_OAUTH_APPLE_PRIVATE_KEY_PATH=/etc/apple-auth/sign-in-with-apple.p8` değerlerini; pod secret'ında KID ve dosya mount'unu doğrulayın.
4. Pod rollout sonrası Apple ile giriş yapılmış sandbox hesapta Ayarlar → Hesabı Sil akışını çalıştırın. Yeniden Apple doğrulaması, başarılı revoke, hesap silme ve sonraki girişin yeni hesap akışı olduğu doğrulanmadan gönderim yapmayın.
5. Key ID veya dosya yoksa backend silmeyi güvenli biçimde reddeder; bu durum submit blocker'dır. Apple-linked kullanıcıyı sadece yerel olarak çıkış yaptırmak hesap silme sayılmaz.

---

## 7. Demo hesap ve App Review erişimi

| Alan | Değer |
|---|---|
| Sign-in required? | Kamusal piyasa içeriği için **No**; yalnız account-based özellikler için **Yes** |
| App Review iletişim adı/e-postası/telefonu | App Store Connect'te dolduruldu; gönderim öncesi Account Holder güncelliğini doğrulayacak |
| Demo kullanıcı adı/e-posta | `appreview2-6790740484@talhacan.com`; production login testi HTTP 200 ile doğrulandı |
| Demo parola | Yalnız App Store Connect → App Review Information parola alanına girildi; repository/doküman/Review Notes'a yazılmadı |

Demo kullanıcı production backend'de gerçekten giriş yapabilmeli ve şunlara sahip olmalıdır: örnek favoriler, en az bir alarm, örnek portföy, bildirim/oyunlaştırma geçmişi. Reviewer'a gerçek kişisel veri göstermeyin. App Store Connect'teki username/password alanları doldurulduktan sonra farklı bir ağdan aynı bilgilerle smoke test yapın. IAP'ler App Review sandbox ortamında gerçek ücret oluşturmadan satın alınabilir; reviewer'ın izleyebileceği paywall yolu Review Notes'ta bulunmalıdır. “Premium hazır” özel server override'ı varsa production kullanıcıya açık olmamalı ve Review Notes'ta dürüstçe açıklanmalıdır.

### App Review Notes — kopyalanabilir

```text
TSC Forex; altın, döviz, kripto ve BIST fiyatlarını, kullanıcı alarmlarını/portföylerini ve kaynaklı derin piyasa araştırmasını sunan bilgilendirme uygulamasıdır. Alım-satım, aracılık veya kişiselleştirilmiş yatırım danışmanlığı sunmaz. Fiyatlar ve araştırma sonuçları hata/gecikme içerebilir; yatırım tavsiyesi değildir.

MİSAFİR ERİŞİMİ (5.1.1(v) DÜZELTMESİ): Build 6 temiz açılışta doğrudan Piyasalar ekranına gelir. Kayıt/giriş olmadan canlı piyasa listesi, arama, Altın/Döviz/Kripto/Hisse kategorileri, varlık detayları, istatistikler, dönüştürücü ve 1G/1H/1A/1Y/5Y/MAX grafik dönemleri serbestçe kullanılabilir. Favoriler, alarm, bildirim, portföy, ödüller, hesap ayarları, abonelik ve Derin Araştırma kullanıcı hesabına kaydedildiği/eşitlendiği için bu eylemler seçildiğinde giriş istenir. Misafir her zaman Piyasa sekmesine dönebilir.

DEMO HESAP: Hesap temelli akışları incelemek için çalışan, kişisel veri içermeyen demo kullanıcı adı ve parola App Store Connect > App Review Information alanlarında ayrı olarak sağlanmıştır.

GİRİŞ YÖNTEMLERİ: E-posta, Sign in with Apple, Google ve Passkey. Sign in with Apple görünür ve Google ile eşdeğerdir.

ABONELİK TESTİ: Ayarlar > Planları İncele yolunu izleyin. Bu yol ücretsiz araştırma hakkının durumundan bağımsız olarak satın alma ekranını açar. Başlangıç, Pro, Uzman ve Elite aylık ürünleri aynı premium_monthly subscription group içindedir. Satın Almaları Geri Yükle ve Aboneliği Apple'da Yönet seçenekleri satın alma/Ayarlar ekranındadır. App Review sandbox işlemleri backend tarafından kabul edilir ve signed JWS doğrulanır.

DERİN ARAŞTIRMA AÇIK İZNİ: İlk araştırma isteğinden önce OpenAI'a seçilen varlık, piyasa bağlamı ve hesaptan tek yönlü türetilen takma adlı güvenlik kimliğinin gönderileceğini açıklayan izin gösterilir. Reddetmek araştırma dışı özellikleri etkilemez; izin Ayarlar'dan geri çekilebilir. OpenAI'a ad, e-posta, profil fotoğrafı, ödeme veya kişisel portföy verisi gönderilmez.

HESAP SİLME: Ayarlar > Tehlikeli Bölge > Hesabı Sil. Aktif Apple aboneliğinin hesap silmeyle iptal olmayacağı ve Apple üzerinden yönetilmesi gerektiği silme öncesi gösterilir. Hesap bağlantısı derhal kaldırılır. Tek yönlü, takma adlı cihaz claim hash'i yalnız kullanıcı+cihaz başına tek ücretsiz derin araştırma hakkının tekrar suistimalini önlemek için ücretsiz kullanım programı boyunca tutulur; tracking/reklam amacıyla kullanılmaz.

BİLDİRİMLER: Kullanıcının kurduğu fiyat alarmları, hesap/güvenlik ve seçtiği öğrenme bildirimleri içindir. Bildirim izni zorunlu değildir.

APP STORE SERVER NOTIFICATIONS V2:
https://tsc-api.talhacan.com/api/v1/billing/apple/notifications
Production ve Sandbox aynı HTTPS endpoint'i kullanır; environment ve bundle ID signed JWS üzerinden doğrulanır.
```

---

## 8. Ekran görüntüleri ve App Review görselleri

### Kabul edilen iPhone boyutları — mevcut sürüm sayfası

App Store Connect'in bu uygulama için gösterdiği 6.5-inch portrait yükleme alanı yalnız şu exact ölçüleri kabul ediyor:

- **1284 × 2778 px**, veya
- **1242 × 2688 px**.

Bağlı iPhone 14 Pro Max'in native **1290 × 2796 px** görüntüsü bu alana doğrudan yüklenebilir ölçü değildir. 1290×2796 dosyayı kabul edilecekmiş gibi belgelemeyin ve görüntüyü orantısız esnetmeyin. Tercih edilen yöntem, kabul edilen çözünürlüğü üreten uygun Xcode Simulator/device sınıfında aynı production build ekranlarını yeniden çekmektir. Marketing canvas hazırlanırsa gerçek UI oranı korunmalı, cihaz çerçevesi/başlık App Store metadata'sını yanıltmamalı ve son dosya exact ölçüyle doğrulanmalıdır.

Uygulama kaydı şu anda universal iPhone + iPad desteği içeriyor. Bu nedenle iPad üzerinde bütün kritik akışlar ayrıca QA edilmeli ve App Store Connect'in sürüm sayfasında istediği iPad screenshot seti yüklenmelidir. iPad desteğini kaldırma kararı yalnız ürün ve teknik kapsam kararı olarak, signed archive'daki device family ile tutarlı biçimde alınabilir; eksik screenshot'ı saklamak için son anda metadata hilesi yapılmaz.

17 Temmuz 2026'da `docs/store/screenshots/ios/iphone-6.5/01-markets.png` exact **1284×2778** olarak doğrulanıp App Store Connect'e yüklendi. Aynı tarihteki iPad login görseli, uygulamanın temel kullanımını göstermediği için PPO incelemesinde Guideline 2.3.3 reddine yol açtı. 26 Temmuz 2026'da bu görsel ve treatment içindeki diğer login-only iPad görselleri kaldırıldı; temiz 13 inç iPad simülatöründe güncel build'in gerçek Piyasalar ekranından alınan exact **2064×2752** görsel hem `1.1` app version'a hem PPO treatment'a yüklendi. Her iki kayıt da App Review'a yeniden gönderildi. Aşağıdaki kapsamlı pazarlama seti hâlâ önerilir.

### Önerilen 8 screenshot sırası

1. **Piyasalar** — “Altın, döviz, kripto ve hisseler tek ekranda”
2. **Varlık detayı** — “Çoklu zaman dilimleriyle fiyat hareketini incele”
3. **Derin Araştırma** — “Makro, jeopolitik ve çapraz piyasa bağlamı; kaynaklarıyla”
4. **Alarm** — “Hedef seviyene gelince haber al”
5. **Portföy** — “Dağılımını ve kâr/zararını takip et”
6. **Öğrenme/ödüller** — “Plan, ders, seri ve süreç odaklı gelişim”
7. **Paywall** — “İhtiyacına uygun aylık planı seç”
8. **Güvenli hesap** — “Apple, Google veya Passkey ile giriş”

### Screenshot QA

- Signed Release adayı ve production API verisi kullan; debug URL, log, test etiketi, kişisel e-posta/avatar, bildirim banner'ı ve cihaz geliştirici göstergesi görünmesin.
- Status bar tutarlı; saat/batarya temiz; hiçbir modal kesilmemiş olsun.
- Derin Araştırma ekranında en az bir kaynak, zaman ufku ve “yatırım tavsiyesi değildir” uyarısı görünsün; garanti veya “kesin yükselir” dili olmasın.
- Paywall screenshot'ında StoreKit'ten gelen gerçek yerel fiyat, süre, auto-renew açıklaması, restore, Terms ve Privacy bağlantıları görünür olsun.
- Apple ürün/logolarını yanıltıcı biçimde kullanma; “Apple tarafından önerilir” izlenimi verme.
- Grafik/fiyatlar gerçekçi ve güncel olmalı; yanıltıcı kazanç vaadi, getiri yüzdesi garantisi veya sahte testimonial kullanma.
- Her screenshot'ın piksel boyutunu `sips -g pixelWidth -g pixelHeight DOSYA` ile doğrula; iPhone çıktısı tam olarak `1284×2778` veya `1242×2688` değilse yükleme.

IAP Review Information için ayrıca satın alma düğmesi, ürün adı/fiyatı, restore ve legal bağlantıları gösteren temiz bir paywall screenshot'ı dört ürünün her birine yüklenmelidir.

---

## 9. Build, archive ve export compliance

- Marketing Version ve Build numarası daha önce yüklenen bütün build'lerden yüksek olmalı.
- Release configuration, production API URL, production Sign in with Apple/Google config ve distribution profile kullanmalı.
- Archive Organizer validation sıfır error ile tamamlanmalı; widget extension imzası/capability'leri eşleşmeli.
- Uygulama yalnız standart HTTPS/TLS ve Apple platform şifrelemesi kullanıyorsa `ITSAppUsesNonExemptEncryption = NO` eklenebilir. Özel/non-exempt kriptografi eklenirse cevabı yeniden değerlendirin.
- StoreKit Configuration file yerel test içindir; production ürünleri App Store Connect'ten gelir. Release build sahte ürün catalog'una bağlı kalmamalıdır.

### 9.1 StoreKit sandbox / TestFlight test matrisi

Her satır tarih, build numarası, test Apple hesabı, transaction/original transaction ID'nin maskelenmiş son kısmı, backend request/notification correlation ID ve ekran kaydıyla kanıtlanır. Secret/JWS/parola kanıta konmaz.

| Senaryo | Ortam | Beklenen sonuç |
|---|---|---|
| Dört ürünün fetch edilmesi | StoreKit sandbox + TestFlight | Exact dört Product ID gelir; ad, bir aylık dönem ve `displayPrice` App Store Connect ile eşleşir. |
| İlk satın alma — her tier | Sandbox, ayrı temiz hesaplar | Verified transaction backend'e sync olur; doğru plan/kotalar bir kez açılır; restart sonrası korunur. |
| Kullanıcı iptali | Sandbox purchase sheet | Premium açılmaz; hata gibi gösterilmez; tekrar satın alma mümkün kalır. |
| Pending / Ask to Buy | Sandbox | İşlem beklemede görünür; onay gelmeden premium açılmaz; daha sonra verified update tek kez uygulanır. |
| Unverified/bozuk JWS | Backend test + sandbox negatif test | Entitlement verilmez, güvenli hata gösterilir ve hassas veri loglanmaz. |
| Restore — aynı cihazda reinstall | Sandbox | `AppStore.sync()` sonrası doğru entitlement döner; çift kota/transaction oluşmaz. |
| Restore — ikinci cihaz | Sandbox | Aynı Apple satın alımı ve uygulama hesabı doğru entitlement'a bağlanır; appAccountToken politikası tutarlı çalışır. |
| Plus → Pro/Uzman/Elite upgrade | Sandbox | Apple'ın effective-date davranışı izlenir; backend en yüksek geçerli entitlement'ı verir; notification tekrarında çift uygulamaz. |
| Elite/Uzman → düşük tier downgrade | Sandbox | Düşük plan Apple'ın bildirdiği yenileme tarihinde başlar; erken kota düşürülmez. |
| Aylık yenileme | Sandbox hızlandırılmış renewal | Kota dönemi bir kez yenilenir; unused hak devretmez; duplicate notification ikinci reset üretmez. |
| Billing retry / `DID_FAIL_TO_RENEW` | Sandbox | Geçerli signed entitlement süresine göre erişim belirlenir; Grace Period OFF olduğu için belgelenmemiş ek grace verilmez. |
| Expiration | Sandbox | Süre bittikten ve geçerli entitlement kalmadıktan sonra plan free'ye döner; paywall/restore çalışır. |
| Refund / revoke | Sandbox + notification V2 | Yetki kaldırılır, kota durumu tutarlı kalır; `REFUND`/`REVOKE` tekrar teslimi idempotenttir. |
| Out-of-order / duplicate notification | Backend fixture + Apple test notification | Signed date/transaction state'e göre deterministik sonuç; eski event yeni entitlement'ı geri almaz. |
| Satın alma sırasında ağ kesilmesi/restart | Sandbox | Kalıcı operation ID ve transaction listener ile sonuç toparlanır; çift satın alma/kota yoktur. |
| Ücretsiz hak suistimal testi | Gerçek cihaz + production-benzeri backend | Aynı kullanıcı veya aynı DeviceCheck cihazı yeni hesap/reinstall ile ikinci ücretsiz derin araştırma alamaz. |
| Hesap silme + aktif Apple aboneliği | Sandbox | Manage uyarısı gösterilir; hesap silinir fakat Apple aboneliği gizlice iptal edilmiş sayılmaz; sonraki restore/recovery politikası deterministiktir. |

Grace Period **OFF** olduğu için grace-period erişim satırı ilk release için N/A'dır. Ayar açılacaksa `GRACE_PERIOD`, recovery ve expiration senaryoları ayrı PASS olmadan production'a alınmaz.

### 9.2 Signed archive, upload ve review sırası

1. Bölüm 10'daki teknik kapıları tamamlayın; Paid Apps Agreement/Tax/Banking, Content Rights, App Privacy ve gerçek review erişimi çözülmeden review'a göndermeyin.
2. Xcode target'larında Marketing Version `1.1`, daha önce yüklenmemiş Build `7`, bundle ID `talhacan.tscforex` ve widget ID `talhacan.tscforex.TSCWidget` olduğunu doğrulayın. Release scheme production API ve production entitlements kullanmalıdır.
3. Fiziksel iPhone'da signed Release smoke testi yapın; ardından iPad desteği korunuyorsa fiziksel cihaz veya uygun simulator'da bütün kritik layout/akışları test edin.
4. Xcode'da `Any iOS Device (arm64)` / generic device seçip **Product → Archive** çalıştırın. Organizer'da app + widget signing team, provisioning profile, capabilities, privacy manifest ve export compliance özetini inceleyin.
5. Organizer → **Validate App** çalıştırın. Error kalmamalı; warning'ler de tek tek açıklanıp release riski taşımadığı doğrulanmalıdır.
6. Organizer → **Distribute App → App Store Connect → Upload** ile build'i yükleyin. Upload başarı mesajı yetmez; App Store Connect'te processing tamamlanıp build görünene kadar bekleyin.
7. İşlenen build'i `1.1 Prepare for Submission` sürümüne seçin; encryption/export compliance sorusunu gerçek archive davranışına göre cevaplayın. Build 7 için bu adım tamamlandı; `ITSAppUsesNonExemptEncryption=false` archive ile uyumludur.
8. 6.5-inch exact iPhone screenshot setini ve mevcut universal destek nedeniyle iPad setini yükleyin. Description, promotional text, keywords, Support/Marketing/Privacy URL'lerini son production UI ile karşılaştırın.
9. Dört subscription'ı aynı app version'ın **In-App Purchases and Subscriptions** bölümüne ekleyin; her üründe localization, availability, fiyat ve ayrı review screenshot'ı bulunduğunu doğrulayın.
10. App Privacy formunu production veri akışına göre publish edin; Content Rights, copyright ve review contact alanlarını Account Holder doğrulanmış bilgilerle tamamlasın. Çalışan demo credentials yalnız güvenli review alanlarına girilsin.
11. Review Notes'u güncel navigation yollarıyla kaydedin. İlk sürümde kontrol için **manual release** seçmek operasyonel olarak daha güvenlidir; automatic release seçilecekse bunun bilinçli ürün kararı olduğunu doğrulayın.
12. Önce **Add for Review** ile Apple'ın eksik alan kontrollerini çalıştırın. App ve dört IAP'nin aynı submission içinde olduğunu görün; bütün uyarılar kapandıktan sonra Account Holder/authorized role **Submit for Review** işlemini yapar.
13. Submission ID, build numarası ve gönderim zamanını release kaydına yazın. Build 7 submission'ı `57ab4e70-1fc0-4bc0-b058-f20ff8ff8d7c`, durumu `Waiting for Review`'dur. App Review mesajlarını izleyin; metadata/build değişikliği gerekirse hangi submission'ın geri çekildiğini açıkça kaydedin.

---

## 10. Gönderim kapıları — hepsi PASS olmadan submit yok

### Ürün ve UX

- [x] App name, bundle ID ve ikon doğru; `1.0 (5)` red nedenleri belgelendi, `1.0 (6)` onaylandı ve yeni sürüm `1.1 (7)` olarak artırıldı.
- [x] Türkçe, İngilizce, Arapça, Almanca, İtalyanca ve İspanyolca bundle localization'ları mevcut; Arapça RTL ve uygulama içi dil seçimi doğrulandı.
- [x] Portföy dağılımı erişilebilir donut grafik + lejant olarak uygulandı; dar ekran yerleşimi için dikey fallback var.
- [x] Kamusal piyasa listesi, arama, kategoriler, varlık detayları ve grafikler temiz oturumda kayıt/giriş olmadan açılıyor; account-based alanlar açıklamalı giriş kapısı kullanıyor.
- [x] Uygulama fiyatı Türkiye bazında `₺0,00`; ilk sürüm yalnız Türkiye'de public iPhone/iPad dağıtımına açık, Mac/Vision Pro kapalı.
- [ ] iPhone ve destek sürüyorsa iPad'de launch, login/logout, cold start ve offline/error durumları çalışıyor.
- [x] Production API ve web erişilebilir; deployed commit doğru, database/Redis `up`, public web ile privacy/terms/support URL'leri HTTP 200. Local network izni Release'te yok.
- [ ] Piyasa, grafik, Derin Araştırma, alarm, portföy, bildirim, widget, profil/avatar, passkey ve oyunlaştırma smoke test edildi.
- [ ] Derin Araştırma çıktısı kaynak, risk, no-advice uyarısı içeriyor; hata/timeout kullanıcıyı kilitlemiyor.

### Abonelik/IAP

- [ ] Paid Applications Agreement **New** durumundan Active'a geçti; yasal kişi, vergi ve banka bilgileri Account Holder tarafından onaylandı.
- [x] `premium_monthly` group (`22243838`) ve dört exact Product ID/Apple ID oluşturuldu.
- [x] Level sırası Elite → Uzman → Pro → Başlangıç olarak kaydedildi.
- [x] Her ürünün bir aylık duration, Türkçe localization, Türkiye availability ve exact TRY price point'i kaydedildi.
- [x] Dört ürünün Review Information screenshot'ı tamamlandı.
- [x] Dört IAP ve subscription group Build 6 app version ile birlikte submission `35107ec9-3de3-4f01-b22a-b4f9e72e658f` içinde onaylandı; yeni 1.1 güncellemesinde mevcut ürün metadata'sı değiştirilmeden korunur.
- [ ] Paywall StoreKit fiyatı, ay, auto-renew, Terms, Privacy, Restore ve Manage gösteriyor.
- [ ] Satın alma yalnız verified transaction/backend entitlement sonrası açılıyor.
- [ ] Restore, upgrade/downgrade, expiration, refund/revoke ve billing retry testleri geçti.
- [x] Production + Sandbox Notifications V2 URL aynı HTTPS endpoint'e Version 2 olarak kaydedildi.
- [ ] Apple test notification, signed JWS doğrulaması ve tekrar teslim idempotency geçti.
- [x] Billing Grace Period ilk release için OFF bırakıldı.
- [x] Server env ve Apple credential mount'ları production podlarında doğrulandı; sandbox yalnız imzalı App Review/TestFlight transaction'ları için kabul ediliyor.
- [ ] App, backend ve dokümandaki bütün kotalar birebir aynı.

### Privacy / legal / account

- [x] Privacy, Terms ve Support public URL'leri canlı ve 200 dönüyor.
- [ ] Privacy Policy'deki veri sorumlusu, gerçek kişi/şirketin tam yasal unvanı ve geçerli tebligat/iletişim adresiyle tamamlandı; yalnız marka adı bırakılmadı.
- [ ] Privacy Policy abonelik, purchase ID, portföy, Derin Araştırma/OpenAI, quota, device claim hash ve retention'ı açıklıyor.
- [ ] App Privacy label Purchases, Other Financial Info, User Content, User ID, Device ID ve Product Interaction dahil kodla eşleşiyor.
- [ ] İlk Derin Araştırma kullanımında OpenAI için açık izin var; ret ve consent revoke test edildi.
- [ ] App Attest challenge/register/assertion ve DeviceCheck available/reserved/consumed akışları gerçek cihazda geçti; uygulamayı silip yeniden kurma + yeni hesap denemesi ikinci ücretsiz hak vermiyor.
- [x] DeviceCheck KID/`.p8` ve Sign in with Apple revoke KID/`.p8` GitHub/Kubernetes secret'larında mevcut; repository/app bundle/log'da yok.
- [ ] `PrivacyInfo.xcprivacy` + archive privacy report incelendi; SDK manifest/signature uyarısı yok.
- [ ] Sign in with Apple production'da çalışıyor; hesap silmede token revoke doğrulandı.
- [ ] Hesap silme aktif abonelik uyarısı + Manage bağlantısı gösteriyor; kullanıcı yine hemen silebiliyor.
- [ ] Silme sonrası kişisel veriler kalkıyor; device claim yalnız belgelenen amaç/süreyle ayrık tutuluyor.

### Metadata / review

- [x] Age Rating formu mevcut özelliklere göre dolduruldu; global sonuç `4+`.
- [ ] Copyright gerçek ve doğrulanmış yasal hak sahibiyle dolduruldu; App Review iletişim adı, telefon ve e-posta çalışan kişiye ait.
- [ ] Bütün piyasa/haber/web kaynakları ve marka görselleri için içerik/yeniden dağıtım hakları belgelendi; Content Rights cevabı kanıtla uyumlu.
- [x] Build 7 archive `ITSAppUsesNonExemptEncryption=false` ile yüklendi; App Store Connect processing ve build seçimi export-compliance engeli olmadan tamamlandı.
- [x] App Review için ayrı demo hesabın production login'i doğrulandı ve credentials yalnız App Store Connect'in güvenli alanlarında tutuluyor.
- [x] Review Notes Build 7, altı dili, Arapça RTL'yi, donut dağılımı, misafir erişimini ve Guideline 2.3.3 ekran görüntüsü düzeltmesini içeriyor; credentials yalnız ayrı güvenli username/password alanlarında tutuluyor.
- [x] Exact `1284×2778` iPhone ve temel Piyasalar işlevini gösteren exact `2064×2752` iPad görseli yüklendi; login-only PPO seti kaldırılıp yeniden incelemeye gönderildi.
- [ ] App ve IAP review screenshots kişisel veri/debug öğesi içermiyor.
- [ ] Support iletişimi izleniyor; reviewer sorusuna hızlı yanıt verilecek.

---

## 11. Başlıca red riskleri ve önlemler

| Risk | Apple alanı | Gerekli önlem |
|---|---|---|
| Paywall yenileme/fiyat bilgisini saklıyor | 3.1.2 | StoreKit fiyatı, süre, auto-renew, Terms/Privacy satın almadan önce |
| Restore yok veya çalışmıyor | 3.1.1 / StoreKit | Görünür restore ve gerçek verified entitlement sync |
| IAP bulunamıyor | 2.1 / 3.1 | Ürünleri version'a ekle; agreements ve availability tamamla |
| Derin Araştırma verisi üçüncü tarafa izinsiz gidiyor | 5.1.2(i) | OpenAI adı + veri + amaç için işlem öncesi explicit consent |
| App Privacy eksik | 5.1.1 / 5.1.2 | Purchases, financial/user content, IDs, usage ve partnerleri doğru beyan et |
| Hesap silme Apple aboneliğini gizliyor | 5.1.1(v) | Silme öncesi billing uyarısı + Manage; immediate delete seçeneğini koru |
| Google var, Apple yok/bozuk | 4.8 | Production Sign in with Apple ve private relay testi |
| Derin Araştırma yatırım danışmanlığı izlenimi | Finance / misleading claims | Kaynak, belirsizlik, no-advice; kişisel al/sat emri ve garanti yok |
| Production build local network ister | 5.1.1 / completeness | Local network key/capability Release'ten çıkar |
| Demo hesap/özellik erişimi yok | 2.1 | Canlı credentials, review path, erişilebilir backend |
| Screenshot yanlış boyut/yanıltıcı | Metadata | Accepted exact dimensions, gerçek UI, no guarantees |
| iPad destekleniyor ama UI bozuk | 2.1 / 4.0 | iPad QA + screenshots veya bilinçli device-family kararı |

---

## 12. Resmî Apple kaynakları

- [App Review Guidelines](https://developer.apple.com/app-store/review/guidelines/)
- [Offer auto-renewable subscriptions](https://developer.apple.com/help/app-store-connect/manage-subscriptions/offer-auto-renewable-subscriptions/)
- [Auto-renewable subscription information and levels](https://developer.apple.com/help/app-store-connect/reference/in-app-purchases-and-subscriptions/auto-renewable-subscription-information)
- [Subscription pricing and availability](https://developer.apple.com/help/app-store-connect/reference/pricing-and-availability/in-app-purchase-and-subscriptions-pricing-and-availability/)
- [Manage subscription pricing](https://developer.apple.com/help/app-store-connect/manage-subscriptions/manage-pricing-for-auto-renewable-subscriptions/)
- [Restoring purchased products](https://developer.apple.com/documentation/storekit/restoring-purchased-products)
- [Handling subscription billing](https://developer.apple.com/documentation/storekit/handling-subscriptions-billing)
- [Offering account deletion in your app](https://developer.apple.com/support/offering-account-deletion-in-your-app)
- [App Privacy Details](https://developer.apple.com/app-store/app-privacy-details/)
- [Privacy manifests](https://developer.apple.com/documentation/bundleresources/privacy_manifest_files)
- [Establishing your app’s integrity with App Attest](https://developer.apple.com/documentation/devicecheck/establishing-your-app-s-integrity)
- [Accessing and modifying per-device data with DeviceCheck](https://developer.apple.com/documentation/devicecheck/accessing-and-modifying-per-device-data)
- [Apple Standard EULA](https://www.apple.com/legal/internet-services/itunes/dev/stdeula/)
- [Screenshot specifications](https://developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications)
- [Manage Apple subscriptions](https://apps.apple.com/account/subscriptions)
- [Request a refund / report a problem](https://reportaproblem.apple.com/)

---

_Son güncelleme: 22 Temmuz 2026_
