# TSC Forex — Google Play Yayın Dokümanı (Android)

Bu doküman, TSC Forex Android uygulamasını Google Play'e **ilk seferde, red/askı yemeden** göndermek için gereken tüm metinleri, Data Safety formu içeriğini, izin açıklamalarını, hesap silme gereksinimini ve adım adım kontrol listesini içerir.

> Amaç: Google Play Developer Program Policies'e tam uyum. Aşağıdaki metinler Play Console alanlarına birebir kopyalanabilir.

---

## 1. Uygulama Kimliği

| Alan | Değer |
|------|-------|
| Uygulama adı (App name, max 30) | **TSC Forex** |
| Paket adı (Application ID) | `com.talhacan.tscforex` |
| Kategori | Finance (Finans) |
| Etiketler | Investing, Stocks, News |
| İçerik derecelendirmesi | Everyone (herkes) |
| Fiyat | Ücretsiz |
| Uygulama içi ürün | Abonelik (Play Billing) — aşağıdaki ürün kimlikleri |
| Reklam | Yok (No ads) |
| Hedef API düzeyi | 36 (Android 16) — Play güncellik gereksinimini karşılar |
| Min SDK | 24 (Android 7.0) |

> **ÖNEMLİ:** `applicationId` artık `com.talhacan.tscforex`'tir. `com.example.*` paketleri Play tarafından **reddedilir**; bu düzeltildi. Dahili `namespace` (com.example.tsc.forex) yalnızca R/BuildConfig üretimi içindir ve yayınlanan pakete yansımaz.

### 1.1 Passkey (geçiş anahtarı) için ZORUNLU adım (release)
Passkey/WebAuthn login (`/auth/passkey/login/*`) Android'de Credential Manager ile bağlandı. Çalışması için:
1. Backend config `android_package_name = com.talhacan.tscforex` ve `android_sha256 = [<release SHA-256>]` doldurulmalı (assetlinks.json bunları yayınlar).
2. RP domaini `tsc.talhacan.com` altında `/.well-known/assetlinks.json` `delegate_permission/common.get_login_creds` ile erişilebilir olmalı (WebAuthn RP ID). API (`tsc-api`) yedeği de aynı dosyayı sunar.
3. Kullanıcı önce hesabında bir passkey kaydetmiş olmalı.
> Bu yapılandırma tamamlanana kadar passkey butonu "kayıtlı geçiş anahtarı bulunamadı" mesajı verir; e-posta/Google girişi etkilenmez.

### 1.2 Google Sign-In için ZORUNLU adım (release)
Google Play, indirilen uygulamayı Play App Signing anahtarıyla yeniden imzalar. Bu nedenle yerel/debug istemcisini değiştirmeden, mağaza sürümü için **ayrı bir OAuth Android istemcisi** tanımlanmalıdır:
1. Google Cloud Console → APIs & Services → Credentials → Create OAuth client ID → **Android**.
2. Package name: `com.talhacan.tscforex`.
3. SHA-1: Play Console → Google Play ile korunanlar → Uygulama imzalama → **Klasik anahtar** altındaki Play App Signing SHA-1'i.
4. Uygulama, Credential Manager `serverClientId` olarak **web client ID**'yi kullanır (`AppConfig.GOOGLE_WEB_CLIENT_ID`); Android istemcisi yalnızca doğrulama içindir.

> Upload anahtarının SHA-1'i, Play Store'dan indirilen uygulamanın çalışma zamanı imzası değildir. Aynı paket adıyla birden fazla imza kullanılacaksa her imza için ayrı Android OAuth istemcisi oluşturun.

---

## 2. Mağaza Girişi (Play Console → Main store listing)

### 2.1 Kısa açıklama (Short description — max 80 karakter)
```
Altın, döviz, kripto ve BIST hisselerini canlı takip et. Alarm kur, izle.
```

### 2.2 Tam açıklama (Full description — max 4000 karakter)
```
TSC Forex; altın, döviz, kripto para ve BIST hisse senetlerini tek uygulamada canlı olarak takip etmenizi sağlar. Fiyatlar saniyeler içinde güncellenir; interaktif grafiklerle piyasayı yakından izlersiniz.

TSC Forex bir bilgilendirme uygulamasıdır. Alım-satım, aracılık veya yatırım danışmanlığı hizmeti sunmaz. Görüntülenen fiyatlar yalnızca bilgi amaçlıdır ve yatırım tavsiyesi niteliği taşımaz.

ÖNE ÇIKAN ÖZELLİKLER

• Canlı fiyatlar: Gram, çeyrek, yarım, tam altın, ons ve ayarlı altınlar; onlarca döviz kuru; popüler kripto paralar ve BIST hisseleri.
• Gerçek zamanlı akış: Fiyatlar WebSocket ile anında ekranınıza düşer.
• İnteraktif grafikler: Yakınlaştırılabilir, kaydırılabilir grafiklerle geçmişi inceleyin.
• Fiyat alarmları: Belirlediğiniz seviyenin üstüne/altına gelince anlık bildirim alın.
• Favoriler: İzlemek istediğiniz varlıkları kişisel listenizde tutun.
• Portföy: Varlıklarınızı ekleyin, güncel değer ve kâr/zararı görün.
• Günün kazananları: En çok yükselen varlıkları hızlıca keşfedin.
• Ödüller: Günlük giriş serileri, XP, seviyeler ve rozetlerle motive kalın.
• Güvenli giriş: E-posta, Google veya geçiş anahtarı (passkey) ile giriş.

GİZLİLİK
Yalnızca hizmeti sunmak için gereken asgari veriyi toplarız. Reklam kimliği kullanmaz, verilerinizi satmayız. Hesabınızı ve tüm verilerinizi uygulama içinden kalıcı olarak silebilirsiniz.

Fiyat verileri üçüncü taraf kaynaklardan alınır; gecikme veya hata içerebilir. Yatırım kararlarınızdan yalnızca siz sorumlusunuz.
```

### 2.3 Yenilikler (Release notes)
```
• Tüm ekranlar farklı telefon ve yazı boyutlarına uyumlu, daha modern bir tasarımla yenilendi
• Piyasa, varlık detayı ve portföy grafikleri daha okunaklı ve etkileşimli hale getirildi
• Portföye alım, satım ve diğer işlem ekleme akışı sadeleştirildi
• Alarmlar, planlar, ödüller ve rozet ilerlemeleri iyileştirildi
• Kapalı piyasa günlerinde günlük grafiklerin son işlem seansını göstermesi sağlandı
• Android 7 ve üzerindeki cihazlar için tarih/grafik uyumluluğu güçlendirildi
```

### 2.4 İletişim ve URL'ler
| Alan | Değer |
|------|-------|
| E-posta | `destek@talhacan.com` |
| Web sitesi | `https://tsc.talhacan.com` |
| Privacy Policy URL (zorunlu) | `https://tsc.talhacan.com/privacy` |

### 2.5 Abonelik ürünleri (Play Console → Monetize → Subscriptions)

iOS ile **aynı product ID**’leri kullanın (backend `planForProduct` bunları bekler):

| Plan | Product ID | Dönem |
|------|------------|-------|
| Başlangıç | `talhacan.tscforex.plus.monthly` | 1 ay |
| Pro | `talhacan.tscforex.pro.monthly` | 1 ay |
| Uzman | `talhacan.tscforex.expert.monthly` | 1 ay |
| Elite | `talhacan.tscforex.elite.monthly` | 1 ay |

Backend doğrulama için Google Play Developer API + service account gerekir (bkz. bölüm 2.6).

### 2.6 Sunucu tarafı Play Billing doğrulama (zorunlu)

1. Google Cloud’da bir proje seçin → **APIs & Services → Library** → **Google Play Android Developer API**’yi etkinleştirin.
2. **Service Account** oluşturun; JSON anahtar indirin.
3. Play Console → **Users and permissions** → service account’u davet edin; en azından **View financial data** + **Manage orders and subscriptions** (veya eşdeğeri abonelik okuma) yetkisi verin.
4. JSON’u cluster’a mount edin (Apple anahtarları gibi out-of-band):

```bash
kubectl -n tsc-forex create secret generic tsc-forex-google-play \
  --from-file=play-billing.json=/path/to/play-billing.json
kubectl -n tsc-forex rollout restart deploy/tsc-forex-app
```

Secret yokken Apple billing etkilenmez; Google sync `503` / unavailable döner.

5. License testers + Internal testing track ile gerçek satın alma akışını doğrulayın.

---

## 3. Data Safety Formu (Play Console → App content → Data safety)

Google Play'in **en sık askı sebebi** yanlış/eksik Data Safety formudur. Aşağıdaki gibi doldurun.

### Genel sorular
- Uygulama kullanıcı verisi topluyor/paylaşıyor mu? → **Evet, topluyor**
- Aktarım şifreleniyor mu (in transit)? → **Evet** (HTTPS/TLS)
- Kullanıcı verisinin silinmesini talep edebilir mi? → **Evet** (uygulama içi + web silme)
- Veriler üçüncü taraflarla **paylaşılıyor** mu (share)? → **Hayır** (yalnızca hizmet sağlayıcı "processing" — Google/Apple auth ve push; bu "paylaşım" sayılmaz)

### Toplanan veri türleri

| Veri türü (Play kategorisi) | Toplanıyor | Amaç | Zorunlu/Opsiyonel |
|-----------------------------|:----------:|------|-------------------|
| Personal info → **Email address** | Evet | Hesap yönetimi, kimlik doğrulama | Zorunlu (hesap için) |
| Personal info → **Name** (opsiyonel) | Evet | Profil, liderlik tablosu | Opsiyonel |
| Personal info → **User IDs** | Evet | Hesap yönetimi, uygulama işlevi | Zorunlu |
| Photos and videos → **Photos** (opsiyonel profil foto) | Evet | Profil kişiselleştirme | Opsiyonel |
| App activity → **Other user-generated content** (favori/alarm/portföy) | Evet | Uygulama işlevi | Opsiyonel |
| App info & performance → **Crash logs** | Evet | Uygulama işlevi, hata ayıklama | Otomatik |
| App info & performance → **Diagnostics** | Evet | Performans, güvenlik | Otomatik |
| Device or other IDs → **Device ID** (push jetonu) | Evet | Bildirim gönderimi | Opsiyonel (bildirim izni) |

Her veri türü için "Is this data processed ephemerally?" → Hayır (kalıcı, hesapla ilişkili). "Is collection required?" → yukarıdaki tabloya göre.

> **Toplanmayanlar:** Konum, kişiler, takvim, SMS/arama kaydı, mikrofon, sağlık, finansal hesap/kart bilgisi, tarama geçmişi, reklam kimliği (AAID) → **HİÇBİRİ**.

---

## 4. İzinler (AndroidManifest) ve Gerekçeleri

| İzin | Neden gerekli | Kullanıcı reddedebilir mi |
|------|---------------|---------------------------|
| `android.permission.INTERNET` | API ve WebSocket ile canlı fiyat almak | Hayır (normal izin) |
| `android.permission.ACCESS_NETWORK_STATE` | Bağlantı durumunu izleyip yeniden bağlanmak | Hayır (normal izin) |
| `android.permission.POST_NOTIFICATIONS` (Android 13+) | Fiyat alarmı ve güvenlik bildirimlerini göstermek | Evet, çalışma zamanında istenir; reddedilirse uygulama çalışmaya devam eder |

> **Tehlikeli/kısıtlı izin YOK:** Konum, kamera, mikrofon, depolama (fotoğraf `PhotoPicker` ile alınır, izin gerektirmez), `QUERY_ALL_PACKAGES`, `MANAGE_EXTERNAL_STORAGE`, SMS/Call log → **kullanılmaz**. Bu, "Permissions Declaration" formunu basitleştirir ve red riskini düşürür.

### Bildirim kullanımı
Bildirimler yalnızca kullanıcının **kendi kurduğu fiyat alarmları** ve hesap/güvenlik uyarıları içindir. Pazarlama/spam bildirimi gönderilmez.

---

## 5. Hesap Silme (Play "Data deletion" gereksinimi) — KRİTİK

Hesap oluşturmayı destekleyen uygulamalar için Google Play, **hem uygulama içi hem de web üzerinden** hesap ve veri silme yolu ister.

✅ **Uygulama içi:** Ayarlar → Tehlikeli Bölge → "Hesabı Sil" (onay diyaloğu ile) → `DELETE /me/account`.
✅ **Web/URL üzerinden:** `https://tsc.talhacan.com/support` sayfasında silme adımları; giriş yapıp Ayarlar'dan silme. Data Safety formundaki "account deletion URL" alanına bu URL girilir.
✅ Silme, sunucuda tüm kullanıcı verisini tek işlemde kalıcı olarak kaldırır (izleme listesi, alarmlar, bildirimler, cihaz jetonları, kimlikler, passkey, oyunlaştırma).

Play Console → App content → **Data deletion** bölümüne:
- "Users can request that some or all data is deleted" → **Yes**
- "You provide a way for users to request account deletion" → **Yes**
- Account deletion URL: `https://tsc.talhacan.com/support`

---

## 6. Financial Features Bildirimi (Play → App content)

Play, finans temalı uygulamalar için ek beyan isteyebilir:
- Uygulama **kişisel kredi / borç verme** sunmuyor.
- **Kripto para alım-satımı / cüzdan / borsa** işlevi **yoktur**; yalnızca fiyat gösterimi yapılır.
- **Yatırım/portföy yönetimi** hizmeti sunulmuyor; portföy yalnızca kullanıcının kendi girdiği kayıtların bilgilendirici takibidir.
- Bu nedenle çoğu "Financial services" beyanı **uygulanamaz (N/A)**; yalnızca "provides financial information/news" seçilir.

> Açıklama ve uygulama içinde "yatırım tavsiyesi değildir" ibaresi bulunduğundan yanıltıcı finansal içerik riski yoktur.

---

## 7. Test Erişimi (Play Console → App access)

Uygulamanın bir kısmı giriş gerektirdiğinden, inceleme ekibine erişim verin:
- "All or some functionality is restricted" → **Yes**
- Talimat: "Piyasa listesi ve varlık detayları girişsiz görüntülenir. Favoriler, portföy, alarmlar ve ödüller için giriş gerekir."
- Demo hesap: `review@tscforex.app` / `Review!2026` (yayından önce oluşturun)

---

## 8. İçerik Derecelendirme Anketi (Content rating)
- IARC anketini doldurun: şiddet/cinsellik/kumar/uyuşturucu → **Hayır**.
- Kullanıcı etkileşimi: sınırlı (liderlik tablosunda görünen ad görünür). Sonuç: **Everyone / 3+** beklenir.

---

## 9. Reklam ve Hedef Kitle
- "Contains ads?" → **No**.
- Target audience: **13+** (çocuklara yönelik değil). "Designed for Families" programına dahil DEĞİL.

---

## 10. Yayın Öncesi Kontrol Listesi (Checklist)

- [ ] `applicationId = com.talhacan.tscforex` (com.example kaldırıldı — doğrulandı)
- [ ] Uygulama adı/etiket: **TSC Forex** (doğrulandı: `app_name`)
- [ ] `versionCode`/`versionName` artırıldı
- [ ] Release AAB, Play App Signing ile imzalandı
- [ ] Google Sign-In için Play App Signing SHA-1 + paket adı ayrı OAuth istemcisine eklendi (Bölüm 1.2)
- [ ] Data Safety formu Bölüm 3'e göre dolduruldu
- [ ] Privacy Policy URL erişilebilir: https://tsc.talhacan.com/privacy
- [ ] Hesap silme: uygulama içi + URL (Bölüm 5)
- [ ] App access: demo hesap + talimat verildi
- [ ] İçerik derecelendirme anketi tamamlandı
- [ ] Reklam yok / hedef kitle 13+ işaretlendi
- [ ] POST_NOTIFICATIONS çalışma zamanı izni doğru isteniyor (Android 13+)
- [ ] Ekran görüntüleri (telefon; en az 2, min 320px) yüklendi
- [ ] Feature graphic (1024×500) ve app icon (512×512) yüklendi
- [ ] "Financial features" beyanı Bölüm 6'ya göre yapıldı

---

## 11. Grafik Varlıklar (Store Assets)

| Varlık | Boyut | Zorunlu |
|--------|-------|:-------:|
| App icon | 512 × 512 PNG (32-bit, alfa) | Evet |
| Feature graphic | 1024 × 500 | Evet |
| Telefon ekran görüntüsü | min 320px, 16:9 veya 9:16, 2–8 adet | Evet |
| 7" tablet | opsiyonel | Hayır |
| 10" tablet | opsiyonel | Hayır |

Önerilen ekranlar: Piyasalar, Varlık detayı+grafik, Alarm, Favoriler, Portföy, Ödüller, Giriş.

---

## 12. Sık Görülen Red/Askı Sebepleri ve Bizim Durumumuz

| Sebep | Politika | Durumumuz |
|-------|----------|-----------|
| `com.example` paket adı | Yükleme reddi | ✅ `com.talhacan.tscforex` |
| Data Safety eksik/yanlış | User Data | ✅ Bölüm 3 rehberi |
| Hesap silme yolu yok | Data deletion | ✅ Uygulama içi + URL |
| Gizlilik politikası yok | User Data | ✅ Public URL |
| Gereksiz hassas izin | Permissions | ✅ Yalnızca INTERNET/NETWORK_STATE/POST_NOTIFICATIONS |
| Yanıltıcı finansal içerik | Financial services | ✅ "Bilgilendirme, tavsiye değildir" |
| Test edilemeyen giriş | App access | ✅ Demo hesap sağlanır |
| Hedef API düzeyi eski | Target API level | ✅ targetSdk 36 |
| Spam bildirim | Spam/Notifications | ✅ Yalnızca kullanıcı alarmları |

---

_Son güncelleme: 14 Temmuz 2026_
