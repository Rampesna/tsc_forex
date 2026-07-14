import type { ReactNode } from "react";
import { Link } from "react-router-dom";

const UPDATED = "14 Temmuz 2026";
const SUPPORT_EMAIL = "destek@talhacan.com";
const PRIVACY_EMAIL = "gizlilik@talhacan.com";

function LegalShell({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div style={{ minHeight: "100vh", background: "var(--bg)" }}>
      <header
        style={{
          borderBottom: "1px solid var(--stroke)",
          padding: "16px 24px",
          display: "flex",
          alignItems: "center",
          gap: 12,
          position: "sticky",
          top: 0,
          background: "rgba(11,14,17,0.85)",
          backdropFilter: "blur(10px)",
          zIndex: 10,
        }}
      >
        <Link to="/" style={{ display: "flex", alignItems: "center", gap: 10, textDecoration: "none", color: "var(--text)" }}>
          <img src="/logo.png" width={30} height={30} style={{ borderRadius: 8 }} alt="" />
          <b style={{ fontSize: 16 }}>TSC Forex</b>
        </Link>
        <div style={{ marginLeft: "auto", display: "flex", gap: 18, fontSize: 13 }}>
          <Link to="/privacy" style={{ color: "var(--text-dim)" }}>Gizlilik</Link>
          <Link to="/terms" style={{ color: "var(--text-dim)" }}>Şartlar</Link>
          <Link to="/support" style={{ color: "var(--text-dim)" }}>Destek</Link>
        </div>
      </header>
      <article
        style={{
          maxWidth: 820,
          margin: "0 auto",
          padding: "40px 24px 96px",
          color: "var(--text)",
          lineHeight: 1.65,
          fontSize: 15,
        }}
        className="legal"
      >
        <h1 style={{ fontSize: 30, fontWeight: 800, marginBottom: 4 }}>{title}</h1>
        <div style={{ color: "var(--text-faint)", fontSize: 13, marginBottom: 28 }}>
          Son güncelleme: {UPDATED}
        </div>
        {children}
      </article>
      <style>{`
        .legal h2 { font-size: 20px; font-weight: 800; margin: 30px 0 10px; }
        .legal h3 { font-size: 16px; font-weight: 700; margin: 20px 0 6px; }
        .legal p, .legal li { color: var(--text-dim); }
        .legal ul { padding-left: 20px; margin: 8px 0; }
        .legal li { margin: 4px 0; }
        .legal a { color: var(--accent-hi); }
        .legal strong { color: var(--text); }
        .legal table { width: 100%; border-collapse: collapse; margin: 12px 0; font-size: 14px; }
        .legal th, .legal td { border: 1px solid var(--stroke); padding: 8px 10px; text-align: left; vertical-align: top; }
        .legal th { color: var(--text); background: var(--surface); }
      `}</style>
    </div>
  );
}

export function Privacy() {
  return (
    <LegalShell title="Gizlilik Politikası">
      <p>
        Bu Gizlilik Politikası, TSC Forex mobil ve web uygulamalarını ("Uygulama") kullandığınızda
        hangi kişisel verileri işlediğimizi, bunları neden ve nasıl işlediğimizi, kimlerle
        paylaştığımızı ve haklarınızı açıklar. Uygulama, canlı döviz, altın, kripto ve hisse
        senedi fiyatlarını <strong>yalnızca bilgilendirme amacıyla</strong> sunar; yatırım tavsiyesi
        vermez.
      </p>

      <h2>1. Veri Sorumlusu ve İletişim</h2>
      <p>
        Veri sorumlusu TSC Forex'tir. Gizlilikle ilgili tüm talepleriniz için{" "}
        <a href={`mailto:${PRIVACY_EMAIL}`}>{PRIVACY_EMAIL}</a> adresinden bize ulaşabilirsiniz.
      </p>

      <h2>2. Topladığımız Veriler ve Amaçları</h2>
      <p>Yalnızca hizmeti sunmak için gerekli olan asgari veriyi toplarız:</p>
      <table>
        <thead>
          <tr><th>Veri</th><th>Neden Topluyoruz</th><th>Zorunlu mu?</th></tr>
        </thead>
        <tbody>
          <tr><td>E-posta adresi</td><td>Hesap oluşturma, giriş, güvenlik bildirimleri</td><td>Evet (hesap için)</td></tr>
          <tr><td>Parola (yalnızca güvenli özetlenmiş/hash biçiminde saklanır)</td><td>Hesabınızı güvence altına almak</td><td>E-posta ile kayıtta</td></tr>
          <tr><td>Görünen ad ve profil fotoğrafı (isteğe bağlı)</td><td>Profilinizi ve liderlik tablosunu kişiselleştirmek</td><td>Hayır</td></tr>
          <tr><td>Sosyal giriş kimliği (Google/Apple benzersiz kimliği ve e-posta)</td><td>Google/Apple ile giriş yaptığınızda hesabınızı eşlemek</td><td>Bu yöntemi seçerseniz</td></tr>
          <tr><td>İzleme listesi, fiyat alarmları, portföy kayıtları</td><td>Seçtiğiniz varlıkları takip etmek ve alarm göndermek</td><td>Hayır (kullanırsanız)</td></tr>
          <tr><td>Cihaz bildirim jetonu (APNs/FCM)</td><td>Alarm ve önemli uyarıları bildirim olarak göndermek</td><td>Bildirime izin verirseniz</td></tr>
          <tr><td>Oyunlaştırma verileri (XP, seviye, seri, rozetler)</td><td>Günlük görev/ödül sistemini çalıştırmak</td><td>Giriş yapan kullanıcılar için</td></tr>
          <tr><td>Teknik günlükler (IP adresi, hata/erişim kayıtları)</td><td>Güvenlik, kötüye kullanımı önleme, hata ayıklama</td><td>Otomatik, kısa süreli</td></tr>
        </tbody>
      </table>
      <p>
        <strong>Toplamadıklarımız:</strong> Reklam kimliği, konum verisi, kişiler, fotoğraf
        kütüphanesi (profil fotoğrafı için seçtiğiniz tek görsel hariç), sağlık verisi veya üçüncü
        taraf reklam/izleme SDK'ları kullanmıyoruz. Verilerinizi <strong>satmıyoruz</strong>.
      </p>

      <h2>3. Hukuki Dayanak</h2>
      <p>
        Verilerinizi; sizinle yaptığımız sözleşmenin (hizmet sunumu) ifası, meşru menfaat
        (güvenlik ve hizmet iyileştirme), açık rızanız (bildirimler, isteğe bağlı profil verileri)
        ve yasal yükümlülükler kapsamında işleriz.
      </p>

      <h2>4. Üçüncü Taraflarla Paylaşım</h2>
      <p>Verilerinizi yalnızca hizmeti sağlamak için gerekli hizmet sağlayıcılarla paylaşırız:</p>
      <ul>
        <li><strong>Apple Push Notification service / Firebase Cloud Messaging:</strong> yalnızca cihazınıza bildirim iletmek için.</li>
        <li><strong>Google / Apple:</strong> yalnızca bu yöntemlerle giriş yaptığınızda kimlik doğrulama için.</li>
        <li><strong>Barındırma altyapısı:</strong> verilerin güvenli sunucularda saklanması.</li>
      </ul>
      <p>
        Piyasa fiyatları kamuya açık kaynaklardan (ör. borsa/piyasa veri sağlayıcıları) alınır ve
        kişisel verinizle ilişkilendirilmez.
      </p>

      <h2>5. Saklama Süresi</h2>
      <p>
        Hesap verileriniz, hesabınız aktif olduğu sürece saklanır. Hesabınızı sildiğinizde, tüm
        kişisel verileriniz (izleme listesi, alarmlar, bildirimler, cihaz jetonları, oyunlaştırma
        verileri, kimlik eşlemeleri) kalıcı olarak ve gecikmeksizin silinir. Teknik günlükler kısa
        bir süre sonra otomatik olarak temizlenir.
      </p>

      <h2>6. Haklarınız</h2>
      <p>KVKK ve GDPR kapsamında; verilerinize erişme, düzeltme, silme, işlemeyi kısıtlama ve itiraz etme haklarına sahipsiniz. Bu hakları kullanmak için:</p>
      <ul>
        <li>Profil bilgilerinizi Uygulama içinden düzenleyebilirsiniz.</li>
        <li>Hesabınızı ve tüm verilerinizi <strong>Uygulama içinden Ayarlar → Hesabı Sil</strong> ile kalıcı olarak silebilirsiniz.</li>
        <li>Diğer talepler için <a href={`mailto:${PRIVACY_EMAIL}`}>{PRIVACY_EMAIL}</a>.</li>
      </ul>

      <h2>7. Güvenlik</h2>
      <p>
        Parolalar modern algoritmalarla özetlenerek (Argon2id) saklanır, tüm trafik HTTPS/TLS ile
        şifrelenir, oturumlar yenilenebilir/iptal edilebilir jetonlarla korunur. Buna rağmen
        internet üzerinden hiçbir aktarımın %100 güvenli olmadığını hatırlatırız.
      </p>

      <h2>8. Çocukların Gizliliği</h2>
      <p>
        Uygulama 13 yaşından küçük çocuklara yönelik değildir ve bilerek onlardan veri toplamayız.
      </p>

      <h2>9. Değişiklikler</h2>
      <p>
        Bu politikayı zaman zaman güncelleyebiliriz. Önemli değişikliklerde Uygulama içinde
        bilgilendirme yaparız. Güncel sürüm her zaman bu sayfada yayınlanır.
      </p>

      <p style={{ marginTop: 28 }}>
        Sorularınız için: <a href={`mailto:${PRIVACY_EMAIL}`}>{PRIVACY_EMAIL}</a>
      </p>
    </LegalShell>
  );
}

export function Terms() {
  return (
    <LegalShell title="Kullanım Şartları">
      <p>
        TSC Forex uygulamasını kullanarak bu Kullanım Şartları'nı kabul etmiş olursunuz. Lütfen
        dikkatle okuyun.
      </p>

      <h2>1. Hizmetin Tanımı</h2>
      <p>
        TSC Forex; döviz, altın, kripto para ve hisse senedi için canlı ve geçmiş fiyat bilgisi,
        izleme listesi, fiyat alarmları, portföy takibi ve oyunlaştırma özellikleri sunan bir
        <strong> bilgilendirme aracıdır</strong>.
      </p>

      <h2>2. Yatırım Tavsiyesi Değildir</h2>
      <p>
        Uygulamadaki hiçbir içerik, fiyat, grafik veya bildirim <strong>yatırım, finans, hukuk veya
        vergi tavsiyesi niteliği taşımaz</strong>. Alım-satım kararlarınız tamamen kendi
        sorumluluğunuzdadır. Yatırım kararlarından önce yetkili bir danışmana başvurmanızı öneririz.
      </p>

      <h2>3. Veri Doğruluğu</h2>
      <p>
        Fiyat verileri üçüncü taraf kaynaklardan alınır ve gecikmeli, hatalı veya eksik olabilir.
        Verilerin kesintisizliğini, doğruluğunu veya belirli bir amaca uygunluğunu garanti etmeyiz.
      </p>

      <h2>4. Hesap ve Uygunluk</h2>
      <ul>
        <li>Bazı özellikler (favoriler, alarmlar, portföy, ödüller) için hesap gerekir.</li>
        <li>Hesap bilgilerinizin gizliliğinden ve hesabınız altındaki tüm etkinlikten siz sorumlusunuz.</li>
        <li>En az 13 yaşında olmalısınız.</li>
      </ul>

      <h2>5. Kabul Edilebilir Kullanım</h2>
      <p>Uygulamayı yasa dışı amaçlarla, sistemleri kötüye kullanarak, otomatik istismar veya
        tersine mühendislik yaparak kullanamazsınız. Oyunlaştırma sistemini hile ile manipüle etmek
        yasaktır.</p>

      <h2>6. Fikri Mülkiyet</h2>
      <p>Uygulama, tasarımı, markası ve yazılımı bize aittir ve telif hakkıyla korunur.</p>

      <h2>7. Sorumluluğun Sınırlandırılması</h2>
      <p>
        Uygulama "olduğu gibi" sunulur. Yürürlükteki yasaların izin verdiği azami ölçüde, Uygulamanın
        kullanımından veya fiyat verilerine dayanılarak alınan kararlardan doğan doğrudan veya
        dolaylı zararlardan sorumlu değiliz.
      </p>

      <h2>8. Fesih</h2>
      <p>
        Hesabınızı istediğiniz zaman Uygulama içinden silebilirsiniz. Şartları ihlal etmeniz halinde
        erişiminizi askıya alabilir veya sonlandırabiliriz.
      </p>

      <h2>9. Uygulanacak Hukuk</h2>
      <p>Bu Şartlar Türkiye Cumhuriyeti kanunlarına tabidir.</p>

      <h2>10. İletişim</h2>
      <p>Sorularınız için: <a href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a></p>
    </LegalShell>
  );
}

export function Support() {
  return (
    <LegalShell title="Destek ve Yardım">
      <p>
        Sorularınız, geri bildirimleriniz veya bir sorun için buradayız.
      </p>

      <h2>İletişim</h2>
      <p>
        E-posta: <a href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a>
        <br />
        Genellikle 1-2 iş günü içinde yanıt veririz.
      </p>

      <h2>Hesabımı nasıl silerim?</h2>
      <p>Hesabınızı ve tüm verilerinizi istediğiniz zaman kalıcı olarak silebilirsiniz:</p>
      <ul>
        <li><strong>iOS / Android:</strong> Ayarlar sekmesi → "Hesabı Sil" → onaylayın.</li>
        <li><strong>Web:</strong> Ayarlar → "Hesabı Sil".</li>
      </ul>
      <p>
        Silme işlemi geri alınamaz; izleme listeniz, alarmlarınız, portföyünüz, bildirimleriniz ve
        oyunlaştırma verileriniz dahil tüm kişisel verileriniz kaldırılır.
      </p>

      <h2>Sık Sorulan Sorular</h2>
      <h3>Fiyatlar neden gerçek zamanlı görünmüyor?</h3>
      <p>Fiyatlar üçüncü taraf kaynaklardan periyodik olarak güncellenir ve piyasa koşullarına göre kısa gecikmeler olabilir.</p>
      <h3>Bildirim alamıyorum.</h3>
      <p>Cihaz ayarlarından TSC Forex için bildirim izninin açık olduğundan emin olun ve bir fiyat alarmı tanımlayın.</p>
      <h3>Giriş yapmadan kullanabilir miyim?</h3>
      <p>Evet. Piyasaları giriş yapmadan izleyebilirsiniz; favoriler, portföy, alarmlar ve ödüller için ücretsiz hesap gerekir.</p>
    </LegalShell>
  );
}
