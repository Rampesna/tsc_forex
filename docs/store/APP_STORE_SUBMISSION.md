# TSC Forex — App Store Yayın Dokümanı (iOS)

Bu doküman, TSC Forex iOS uygulamasını App Store'a **ilk seferde, red yemeden** göndermek için gereken tüm metinleri, gizlilik beyanlarını, izin açıklamalarını, App Review notlarını ve adım adım kontrol listesini içerir.

> Amaç: App Store Review Guidelines'a tam uyum. Aşağıdaki metinler App Store Connect alanlarına birebir kopyalanabilir.

---

## 1. Uygulama Kimliği

| Alan | Değer |
|------|-------|
| Uygulama adı (App Name) | **TSC Forex** |
| Alt başlık (Subtitle, max 30 karakter) | Canlı altın, döviz & kripto |
| Bundle ID | `talhacan.tscforex` |
| Kategori (Primary) | Finance (Finans) |
| Kategori (Secondary) | Utilities |
| Yaş sınırı (Age Rating) | 4+ (kumar/alkol vb. içerik yok) |
| Fiyat | Ücretsiz |
| Uygulama içi satın alma | Yok |

> **Görünen ad** cihaz ana ekranında `TSC Forex` olarak çıkar (`INFOPLIST_KEY_CFBundleDisplayName = "TSC Forex"`).

---

## 2. Mağaza Metinleri (App Store Connect → App Information / Version)

### 2.1 Promosyon Metni (Promotional Text — max 170 karakter, sürüm olmadan güncellenebilir)
```
Altın, döviz, kripto ve BIST hisselerini canlı takip et. Fiyat alarmları kur, favorilerini izle, portföyünü yönet. Bilgilendirme amaçlıdır, yatırım tavsiyesi değildir.
```

### 2.2 Açıklama (Description)
```
TSC Forex; altın, döviz, kripto para ve BIST hisse senetlerini tek uygulamada canlı olarak takip etmenizi sağlar. Fiyatlar saniyeler içinde güncellenir; interaktif grafiklerle piyasayı yakından izlersiniz.

TSC Forex bir bilgilendirme uygulamasıdır. Alım-satım, aracılık veya yatırım danışmanlığı hizmeti sunmaz. Görüntülenen fiyatlar yalnızca bilgi amaçlıdır ve yatırım tavsiyesi niteliği taşımaz.

ÖNE ÇIKAN ÖZELLİKLER

• Canlı fiyatlar: Gram/çeyrek/yarım/tam altın, ons ve ayarlı altınlar; onlarca döviz kuru; popüler kripto paralar ve BIST hisseleri.
• Gerçek zamanlı akış: WebSocket ile fiyatlar anında ekranınıza düşer.
• İnteraktif grafikler: Yakınlaştırılabilir, kaydırılabilir, dokunmatik imleçli mum ve çizgi grafikleri.
• Fiyat alarmları: Seçtiğiniz seviyenin üstüne/altına gelince anlık bildirim.
• Favoriler: İzlemek istediğiniz varlıkları kişisel listenizde sıralayın.
• Portföy: Varlıklarınızı ekleyin, güncel değer ve kâr/zararı görün.
• Günün kazananları: En çok yükselen varlıkları hızlıca keşfedin.
• Ödüller: Günlük giriş serileri, XP, seviyeler ve rozetlerle takipte kalın.
• Güvenli giriş: E-posta, Google, Apple ile Giriş ve Passkey desteği.

GİZLİLİK
Yalnızca hizmeti sunmak için gereken asgari veriyi toplarız. Reklam kimliği kullanmaz, verilerinizi satmayız. Hesabınızı ve tüm verilerinizi uygulama içinden tek dokunuşla kalıcı olarak silebilirsiniz.

Fiyat verileri üçüncü taraf kaynaklardan alınır; gecikme veya hata içerebilir.
```

### 2.3 Anahtar Kelimeler (Keywords — max 100 karakter, virgülle)
```
altın,döviz,kur,dolar,euro,kripto,bitcoin,borsa,bist,hisse,gram altın,fiyat,alarm,portföy,finans
```

### 2.4 Yenilikler (What's New — bu sürüm)
```
• İlk sürüm: canlı altın, döviz, kripto ve BIST hisse fiyatları
• İnteraktif grafikler ve fiyat alarmları
• Favoriler, portföy ve günlük ödül sistemi
• Google, Apple ve Passkey ile güvenli giriş
```

### 2.5 Destek & Pazarlama URL'leri
| Alan | URL |
|------|-----|
| Support URL | `https://tsc.talhacan.com/support` |
| Marketing URL | `https://tsc.talhacan.com` |
| Privacy Policy URL | `https://tsc.talhacan.com/privacy` |

---

## 3. App Privacy (Gizlilik Beslenme Etiketi) — App Store Connect → App Privacy

App Store Connect'te "App Privacy" bölümünde aşağıdaki gibi doldurun. **Doğru doldurmak red sebebini ortadan kaldırır.**

### Genel
- **Do you or your third-party partners collect data from this app?** → **Yes**
- Reklam/izleme (Tracking) yapıyor musunuz? → **No** (IDFA kullanmıyoruz, `AppTrackingTransparency` gerektirmez).

### Toplanan Veri Türleri ve Amaçları

| Veri Türü (Apple kategorisi) | Toplanıyor mu | Kullanım Amacı | Kimliğe bağlı mı? | Takip için mi? |
|------------------------------|:-------------:|----------------|:-----------------:|:--------------:|
| Contact Info → **Email Address** | Evet | App Functionality (hesap, giriş) | Evet | Hayır |
| User Content → **Photos** (profil fotoğrafı, isteğe bağlı) | Evet | App Functionality | Evet | Hayır |
| User Content → **Other User Content** (favori/alarm/portföy) | Evet | App Functionality | Evet | Hayır |
| Identifiers → **User ID** | Evet | App Functionality | Evet | Hayır |
| Identifiers → **Device ID** (push jetonu) | Evet | App Functionality (bildirim) | Evet | Hayır |
| Diagnostics → **Crash / Performance / Other Diagnostic** | Evet | App Functionality, güvenlik | Hayır | Hayır |

> **Name** ve **Photos** yalnızca kullanıcı profilini doldurursa toplanır (isteğe bağlı).
> **Location, Contacts, Health, Financials (banka/kart), Browsing History, Search History, Purchases, Sensitive Info, Advertising Data → HİÇBİRİ toplanmaz.**

### Data Not Linked / Linked
- E-posta, User ID, Device ID, User Content → **Linked to identity** (hesaba bağlı).
- Diagnostics → **Not linked to identity**.

---

## 4. İzin Açıklama Metinleri (Info.plist Usage Strings)

Aşağıdaki metinler `Info.plist` içinde tanımlıdır; App Review bunları izin isteminde görür.

| Anahtar | Metin | Neden gerekli |
|---------|-------|---------------|
| `NSLocalNetworkUsageDescription` | "TSC Forex canlı fiyatları yerel ağdaki sunucudan alır." | Yalnızca DEBUG derlemesinde aynı Wi-Fi'daki geliştirme sunucusuna bağlanmak için. Yayın sürümü HTTPS uzak sunucu kullanır. |
| Push Notifications (izin istemi) | Sistem metni | Fiyat alarmı ve önemli uyarıları bildirim olarak göndermek. Kullanıcı reddedebilir; uygulama çalışmaya devam eder. |

> **Kamera / Fotoğraf kütüphanesi:** Profil fotoğrafı `PHPickerViewController` (kullanıcı tarafından seçilen tek görsel) ile alınır; bu API **izin dizesi gerektirmez**. Kamera erişimi istenmez.
> **Konum, mikrofon, kişiler, takvim:** Kullanılmaz, izin istenmez.

### Push Notification Kullanımı (Review notu)
- `aps-environment` entitlement mevcuttur.
- Bildirimler yalnızca kullanıcının **kendi kurduğu fiyat alarmları** ve hesap/güvenlik uyarıları için gönderilir. Pazarlama/spam bildirimi gönderilmez (Guideline 4.5.4 uyumu).

---

## 5. Sign in with Apple Zorunluluğu (Guideline 4.8) — KRİTİK

Uygulama Google ile Giriş sunduğu için, Apple **Sign in with Apple** (veya eşdeğer gizlilik dostu bir seçenek) sunmayı zorunlu kılar.

✅ Uygulamada **Apple ile Giriş** mevcut ve çalışır durumda.
✅ Ayrıca e-posta ile giriş de sunulur (üçüncü taraf gerektirmeyen seçenek).

> Bu zorunluluk karşılanmazsa **kesin red** gelir. Build göndermeden önce Apple ile Giriş butonunun görünür ve çalışır olduğunu doğrulayın.

---

## 6. Hesap Silme (Guideline 5.1.1(v)) — KRİTİK

Hesap oluşturmayı destekleyen uygulamalar, **uygulama içinden hesap silme** sunmak zorundadır.

✅ **Ayarlar → Tehlikeli Bölge → "Hesabı Sil"** akışı mevcuttur.
✅ Silme, sunucuda tüm kullanıcı verisini (izleme listesi, alarmlar, bildirimler, cihaz jetonları, kimlik eşlemeleri, passkey'ler, oyunlaştırma) **tek işlemde kalıcı olarak** kaldırır (`DELETE /me/account`).
✅ Silme sonrası kullanıcı otomatik çıkış yapar.

App Review notuna şu cümleyi ekleyin (bkz. Bölüm 9).

---

## 7. Demo Hesap (App Review Bilgileri)

App Review'ın tüm özellikleri test edebilmesi için hazır bir hesap verin:

| Alan | Değer |
|------|-------|
| Sign-in required? | **Yes** (bazı özellikler için) |
| Demo e-posta | `review@tscforex.app` (yayından önce oluşturun) |
| Demo parola | `Review!2026` (yayından önce belirleyin) |

> Not: Piyasa listesi ve varlık detayları **giriş yapılmadan** da görüntülenebilir. Favoriler, portföy, alarmlar ve ödüller için giriş gerekir. Demo hesabı bu özellikleri test etmek içindir.

---

## 8. Export Compliance (İhracat Uyumu)
- Uygulama yalnızca **standart HTTPS/TLS** kullanır; özel/patentli şifreleme içermez.
- App Store Connect'te sorulduğunda: **"Uses encryption? Yes"** → **"Only standard encryption (HTTPS/TLS)"** → İstisna kapsamındadır.
- `Info.plist`'e `ITSAppUsesNonExemptEncryption = NO` eklenmesi önerilir (böylece her sürümde soru tekrar sorulmaz).

---

## 9. App Review Notları (Review Notes'a birebir yapıştırın)

```
TSC Forex, altın/döviz/kripto/BIST hisse fiyatlarını gösteren bir bilgilendirme uygulamasıdır. Alım-satım, aracılık veya yatırım danışmanlığı YAPMAZ; uygulama içi satın alma yoktur.

GİRİŞ: Piyasa listesi ve varlık detayları giriş olmadan görüntülenebilir. Favoriler, portföy, alarmlar ve ödüller için giriş gerekir.
Demo hesap: review@tscforex.app / Review!2026

GİRİŞ YÖNTEMLERİ: E-posta, Sign in with Apple, Google ve Passkey desteklenir (Guideline 4.8 uyumu için Sign in with Apple mevcuttur).

HESAP SİLME (5.1.1(v)): Ayarlar > "Hesabı Sil" ile kullanıcı hesabını ve tüm verilerini kalıcı olarak silebilir.

BİLDİRİMLER: Yalnızca kullanıcının kendi kurduğu fiyat alarmları ve güvenlik uyarıları için gönderilir; pazarlama bildirimi yoktur.

VERİ KAYNAĞI: Fiyatlar üçüncü taraf piyasa veri sağlayıcılarından alınır ve yalnızca bilgi amaçlıdır.
```

---

## 10. Ekran Görüntüleri & Önizleme (Screenshots)

App Store zorunlu boyutlar (en az bu iki set):

| Cihaz | Çözünürlük (px) | Zorunlu |
|-------|-----------------|:-------:|
| iPhone 6.9" (15/16 Pro Max) | 1320 × 2868 | Evet |
| iPhone 6.5" (11 Pro Max/XS Max) | 1242 × 2688 | Önerilir |
| iPad 13" (varsa iPad desteği) | 2064 × 2752 | iPad destekliyorsa |

Önerilen 5–8 ekran ve başlıkları:
1. **Piyasalar** — "Altın, döviz, kripto ve hisseler tek ekranda"
2. **Varlık detayı + grafik** — "İnteraktif grafiklerle piyasayı yakından izle"
3. **Fiyat alarmı** — "Hedef fiyatına gelince haber ver"
4. **Favoriler** — "İzleme listeni kendine göre sırala"
5. **Portföy** — "Varlıklarını ve kâr/zararını takip et"
6. **Ödüller** — "Günlük seriler, XP ve rozetler"
7. **Giriş** — "Apple, Google veya Passkey ile güvenli giriş"

> Ekran görüntülerinde gerçek fiyat gösterirken "Bilgilendirme amaçlıdır" ibaresini korumak red riskini azaltır.

---

## 11. Yayın Öncesi Kontrol Listesi (Checklist)

- [ ] Uygulama görünen adı: **TSC Forex** (doğrulandı)
- [ ] Sürüm/Build numarası artırıldı (ör. 1.0.0 / build 1)
- [ ] Apple ile Giriş görünür ve çalışıyor (Guideline 4.8)
- [ ] Hesap silme akışı çalışıyor (Guideline 5.1.1(v))
- [ ] App Privacy formu Bölüm 3'e göre dolduruldu
- [ ] Privacy Policy URL erişilebilir: https://tsc.talhacan.com/privacy
- [ ] Support URL erişilebilir: https://tsc.talhacan.com/support
- [ ] Demo hesap oluşturuldu ve Review Notes'a yazıldı
- [ ] Export compliance: yalnızca standart şifreleme
- [ ] Push bildirimleri yalnızca kullanıcı alarmları için (4.5.4)
- [ ] Yatırım tavsiyesi olmadığına dair uyarı açıklamada mevcut
- [ ] ATS: `NSAllowsArbitraryLoads` kaldırıldı, yalnızca `NSAllowsLocalNetworking` (DEBUG) var
- [ ] Ekran görüntüleri 6.9" için yüklendi
- [ ] `ITSAppUsesNonExemptEncryption = NO` eklendi (isteğe bağlı ama önerilir)

---

## 12. Sık Görülen Red Sebepleri ve Bizim Durumumuz

| Red Sebebi | Kılavuz | Durumumuz |
|-----------|---------|-----------|
| Google girişi var ama Apple girişi yok | 4.8 | ✅ Apple ile Giriş mevcut |
| Hesap silme yok | 5.1.1(v) | ✅ Uygulama içi silme mevcut |
| Gizlilik politikası yok/erişilemiyor | 5.1.1 | ✅ Public URL mevcut |
| App Privacy formu eksik/yanlış | 5.1.2 | ✅ Bölüm 3 rehberi |
| Spam/pazarlama push | 4.5.4 | ✅ Sadece kullanıcı alarmları |
| Finansal tavsiye izlenimi | 3.1.1 / genel | ✅ "Yatırım tavsiyesi değildir" ibaresi |
| Eksik demo hesap | 2.1 | ✅ Demo hesap sağlanır |
| Anlamsız yerel ağ izni | 5.1.1 | ✅ Yalnızca DEBUG, açıklaması net |

---

_Son güncelleme: 14 Temmuz 2026_
