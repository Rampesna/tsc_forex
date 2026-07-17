# TSC Forex — Mağaza Yayın Dokümanları

Bu klasör, mağaza inceleme riskini azaltmak ve yayın kapılarını kanıtla takip etmek için gereken dokümanları içerir. Apple veya Google onayı garanti edilemez; eksik kapı varken gönderim yapılmaz.

| Doküman | Kapsam |
|---------|--------|
| [APP_STORE_SUBMISSION.md](./APP_STORE_SUBMISSION.md) | iOS — gerçek App Store Connect durumu, dört auto-renewable subscription/IAP, StoreKit test matrisi, Derin Araştırma açık izni, App Privacy, hesap silme, screenshot, signed archive/upload ve gönderim kapıları |
| [PLAY_STORE_SUBMISSION.md](./PLAY_STORE_SUBMISSION.md) | Android — Play Console metinleri, Data Safety formu, izin gerekçeleri, hesap silme (uygulama içi + URL), applicationId, checklist |

## Ortak yayın bağlantıları

- **Gizlilik Politikası:** https://tsc.talhacan.com/privacy
- **Kullanım Şartları:** https://tsc.talhacan.com/terms
- **Destek / Hesap silme rehberi:** https://tsc.talhacan.com/support
- Bu sayfalar web uygulamasında **giriş gerektirmeyen** public route olmalıdır (`web/src/pages/Legal.tsx`); her release öncesi canlıda TLS/200/render kontrolü yapılır.

## Hesap silme (App Store 5.1.1(v) & Play Data deletion)

- Backend uç noktası: `DELETE /me/account` — kullanıcıyı ve tüm verisini tek transaction'da kalıcı olarak siler
  (`internal/repository/user_repository.go → DeleteUser`, `internal/api/user_handler.go → DeleteAccount`).
- iOS: Ayarlar → "Hesabı Sil" (onaylı).
- Android: Ayarlar → "Hesabı Sil" (onay diyaloglu).
- Web: Ayarlar → "Hesabı kalıcı olarak sil".

## Açık manuel/yetkili hesap adımları (yayından önce)

1. Account Holder: `New` durumundaki Paid Apps Agreement'ı, doğrulanmış yasal kişi, vergi ve banka bilgileriyle tamamla.
2. Content Rights: piyasa/haber/veri yeniden dağıtım haklarını belgeleyip App Store Connect cevabını gerçek sözleşmelere göre ver.
3. App Privacy, gerçek App Review iletişim kişisi/telefonu, copyright ve production'da çalışan demo credentials alanlarını tamamla; parola veya hassas yasal veriyi repository'ye yazma.
4. iOS: oluşturulmuş `premium_monthly` group (`22243838`) içindeki dört ürüne review screenshot'ı ekle ve ürünleri sürüm `1.0` submission'ına ilişkilendir.
5. DeviceCheck ve Sign in with Apple revoke Key ID/`.p8` değerlerini GitHub/Kubernetes secret olarak yükle; gerçek cihazda ücretsiz hak suistimal kontrolünü ve Apple-linked hesap silme revoke akışını doğrula.
6. Production ve Sandbox Notifications V2 URL'leri kaydedildi; Apple test notification + sandbox/TestFlight purchase/restore/upgrade/downgrade/refund/revoke matrisini yine de uçtan uca geçir.
7. `ITSAppUsesNonExemptEncryption` cevabını ve `PrivacyInfo.xcprivacy` doğruluğunu signed Release archive/Privacy Report üzerinde kontrol et.
8. iPhone görsellerini App Store Connect'in kabul ettiği exact **1284×2778** veya **1242×2688** ölçüde hazırla. Mevcut universal destek nedeniyle iPad QA ve istenen iPad setini de tamamla; 1290×2796 native telefon çıktısını doğrudan yükleme.
9. Signed Release archive'ı validate/upload et, işlenen build'i sürüme seç ve APP_STORE_SUBMISSION.md'deki bütün kapılar PASS olduktan sonra review'a gönder.
10. Android: release SHA-1 + `com.talhacan.tscforex` için Google OAuth Android istemcisini doğrula.
