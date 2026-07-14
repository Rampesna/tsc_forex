# TSC Forex — Mağaza Yayın Dokümanları

Bu klasör, uygulamayı **red yemeden** mağazalara göndermek için gereken her şeyi içerir.

| Doküman | Kapsam |
|---------|--------|
| [APP_STORE_SUBMISSION.md](./APP_STORE_SUBMISSION.md) | iOS — App Store Connect metinleri, App Privacy etiketi, izin dizeleri, Sign in with Apple/hesap silme uyumu, App Review notları, checklist |
| [PLAY_STORE_SUBMISSION.md](./PLAY_STORE_SUBMISSION.md) | Android — Play Console metinleri, Data Safety formu, izin gerekçeleri, hesap silme (uygulama içi + URL), applicationId, checklist |

## Ortak altyapı (her iki mağaza için de hazır)

- **Gizlilik Politikası:** https://tsc.talhacan.com/privacy
- **Kullanım Şartları:** https://tsc.talhacan.com/terms
- **Destek / Hesap silme rehberi:** https://tsc.talhacan.com/support
- Bu sayfalar web uygulamasında **giriş gerektirmeyen** public route'lardır (`web/src/pages/Legal.tsx`).

## Hesap silme (App Store 5.1.1(v) & Play Data deletion)

- Backend uç noktası: `DELETE /me/account` — kullanıcıyı ve tüm verisini tek transaction'da kalıcı olarak siler
  (`internal/repository/user_repository.go → DeleteUser`, `internal/api/user_handler.go → DeleteAccount`).
- iOS: Ayarlar → "Hesabı Sil" (onaylı).
- Android: Ayarlar → "Hesabı Sil" (onay diyaloglu).
- Web: Ayarlar → "Hesabı kalıcı olarak sil".

## Yapılması gereken manuel adımlar (yayından önce)

1. Demo inceleme hesabı oluştur: `review@tscforex.app` (her iki mağaza için).
2. Android: release SHA-1 + `com.talhacan.tscforex` için Google OAuth Android istemcisi tanımla.
3. iOS: `ITSAppUsesNonExemptEncryption = NO` ekle (isteğe bağlı, önerilir).
4. Ekran görüntüleri ve grafik varlıkları hazırla (boyutlar ilgili dokümanlarda).
5. Legal sayfaların (`/privacy`, `/terms`, `/support`) canlıda erişilebilir olduğunu doğrula.
