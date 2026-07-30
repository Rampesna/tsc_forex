# TSC Forex ASO çalışma sistemi

Bu klasör App Store ve Google Play mağaza metinlerinin tek kaynağıdır. Güncel altı dildeki alanlar `metadata.json` içinde tutulur. Mağazada doğrudan düzenleme yapılırsa aynı değişiklik bu dosyaya da işlenmelidir.

> Bir arama sonucunda ilk 10 sırası garanti edilemez. Sıralama; mağaza metinlerinin yanında indirme hızı, dönüşüm, elde tutma, puanlar, yorumlar, ülke, kişiselleştirme ve rakiplerin performansına bağlıdır. Sahte yorum, teşvikli puan veya yanıltıcı anahtar kelime kullanılmaz.

## 30 Temmuz 2026 başlangıç ölçümü

- Türkiye App Store'da `altın fiyatları`, `canlı altın`, `döviz kuru`, `dolar kuru`, `portföy takip` ve `kripto takip` sorgularının ilk 15 sonucunda TSC Forex görünmüyordu.
- Apple ürün sayfasında puan/yorum birikimi henüz yoktu.
- En büyük fırsat, Türkçe `altın + döviz + canlı kur + alarm + portföy` niyetini isim, alt başlık, anahtar kelimeler ve ilk üç görselde tutarlı biçimde karşılamaktır.
- İngilizce, Almanca, İtalyanca, İspanyolca ve Arapça alanlar kelime kelime çeviri yerine ilgili mağazadaki arama niyetine göre yerelleştirilmiştir.

## Uygulanan mağaza yapısı

- App Store: 30 karakter sınırına uygun yerelleştirilmiş ad ve alt başlık; 100 karaktere uygun, tekrarsız keyword alanı; doğal açıklama; sürüm notu ve promosyon metni.
- Google Play: 30 karakter ad, 80 karakter kısa açıklama ve 4.000 karakter tam açıklama; altı yerel mağaza girişi.
- Ana kategori `Finance`, ikincil kategori `News` olarak korunur.
- Uygulama içindeki gerçek değer anlarından sonra, platformun yerel değerlendirme penceresi kontrollü biçimde istenir. İstek ödeme, ödül veya olumlu görüş şartına bağlanmaz.
- Apple'daki mevcut Product Page Optimization testi çalışırken ikinci bir test başlatılmaz; sonuç istatistiksel olarak anlamlı hale gelince kazanan varyasyon uygulanır.

## Görsel sıralama ve deney planı

İlk üç ekran mağaza arama sonuçlarında en yüksek etkiye sahip alanlardır:

1. Canlı altın ve döviz / Live Gold & FX Rates
2. Etkileşimli piyasa grafikleri
3. Akıllı fiyat alarmları
4. Portföy değeri ve dağılımı
5. Kaynak bağlantılı varlık araştırması

`metadata.json` içindeki `screenshotHeadlines` her dil için onaylı başlıklardır. Görseller yalnız gerçek uygulama ekranlarından üretilmeli; cihaz çerçevesi, fiyat ve özellikler güncel build ile birebir uyuşmalıdır.

## Haftalık ölçüm

Her pazartesi ülke ve mağaza bazında şu metrikler kaydedilir:

- ana sorgulardaki görünürlük ve sıra;
- product-page impression → indirme dönüşümü;
- ilk açılış, kayıt ve 7/30 günlük elde tutma;
- çökmesiz kullanıcı ve ANR oranı;
- puan ortalaması, yeni yorum adedi ve yanıt süresi;
- ücretli paket ürün sayfası → satın alma dönüşümü.

Tek seferde yalnız bir ana değişken test edilir. Düşük hacimde en az 14 gün; yeterli trafikte mağazanın güven aralığı sonuçlanana kadar test sürdürülür.

## 90 günlük sıra

1. **0–30 gün:** yeni yerelleştirilmiş metadata, stabil build, doğal değerlendirme akışı ve mevcut Apple görsel testinin tamamlanması.
2. **31–60 gün:** kazanan ilk üç görseli uygulama; Play Store Listing Experiment ile aynı değer önerisini test etme; yorumlara 48 saat içinde cevap verme.
3. **61–90 gün:** sorgu bazlı görünürlüğe göre tek alanlı metadata iterasyonu; yüksek dönüşümlü ülkelere özel görsel set; sezonluk altın/döviz temalı In-App Event veya Promotional Content yalnız gerçek bir etkinlik varsa.

## Yayın kapısı

- `metadata.json` alan uzunlukları otomatik veya manuel doğrulandı.
- Ekran görüntüleri özellikleri ve fiyatları doğru gösteriyor.
- Uygulama çökme/ANR, giriş, misafir piyasa erişimi, hesap silme ve abonelik akışları geçti.
- Apple/Google gizlilik beyanları mevcut veri akışıyla eşleşiyor.
- Yeni build'in sürüm kodu benzersiz ve mağazanın zorunlu SDK/billing şartlarını karşılıyor.
