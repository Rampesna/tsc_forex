import type { ReactNode } from "react";
import { Link } from "react-router-dom";

const UPDATED = "17 Temmuz 2026";
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
          <tr><td>Görünen ad ve profil fotoğrafı (isteğe bağlı)</td><td>Profilinizi kişiselleştirmek</td><td>Hayır</td></tr>
          <tr><td>Sosyal giriş kimliği (Google/Apple benzersiz kimliği ve e-posta)</td><td>Google/Apple ile giriş yaptığınızda hesabınızı eşlemek</td><td>Bu yöntemi seçerseniz</td></tr>
          <tr><td>İzleme listesi, fiyat alarmları, portföy kayıtları</td><td>Seçtiğiniz varlıkları takip etmek ve alarm göndermek</td><td>Hayır (kullanırsanız)</td></tr>
          <tr><td>Cihaz bildirim jetonu (APNs/FCM)</td><td>Alarm ve önemli uyarıları bildirim olarak göndermek</td><td>Bildirime izin verirseniz</td></tr>
          <tr><td>Oyunlaştırma verileri (XP, seviye, seri, rozetler)</td><td>Günlük görev/ödül sistemini çalıştırmak</td><td>Giriş yapan kullanıcılar için</td></tr>
          <tr><td>Abonelik ve satın alma geçmişi (ürün kimliği, plan, durum, yenileme/sona erme zamanı, Apple işlem kimlikleri)</td><td>Premium erişimi ve kullanım kotalarını tanımlamak, satın almayı geri yüklemek ve dolandırıcılığı önlemek</td><td>Abone olursanız</td></tr>
          <tr><td>AI araştırma kullanımı (seçilen varlık, piyasa bağlamı, oluşturulan rapor, kaynaklar ve kota kaydı)</td><td>İstediğiniz piyasa araştırmasını üretmek, geçmişinizi göstermek ve kotayı uygulamak</td><td>AI özelliğini kullanırsanız</td></tr>
          <tr><td>Portföy içeriği (varlık, miktar, maliyet ve türetilen performans)</td><td>Portföy değeri, dağılımı ve sağlık analizini hesaplamak</td><td>Portföy kullanırsanız</td></tr>
          <tr><td>iOS kötüye kullanım önleme verisi (Keychain'de üretilen rastgele, uygulamaya özel kurulum kimliğinin tek yönlü özeti; Apple App Attest doğrulama kaydı ve DeviceCheck deneme durumu)</td><td>Tek seferlik ücretsiz AI hakkının hesap, yeniden kurulum veya doğrulanmış iOS uygulaması değiştirilerek aşılmasını ve otomatik kötüye kullanımı önlemek</td><td>iOS'ta ücretsiz AI denemesini kullanırsanız</td></tr>
          <tr><td>Teknik günlükler (IP adresi, hata/erişim kayıtları)</td><td>Güvenlik, kötüye kullanımı önleme, hata ayıklama</td><td>Otomatik, kısa süreli</td></tr>
        </tbody>
      </table>
      <p>
        Web uygulamasındaki tarayıcı depolama değeri bir donanım kimliği veya fiziksel cihaz
        doğrulaması değildir ve tek seferlik ücretsiz denemeyi açmaz. Ücretsiz deneme yalnız
        desteklenen fiziksel iOS cihazında Apple App Attest ile uygulama bütünlüğü doğrulandıktan
        sonra kullanılabilir. DeviceCheck, Apple nezdinde bu cihazın ücretsiz hakkı kullanıp
        kullanmadığını iki bitlik uygulama durumuyla kaydeder; bize seri numarası, IDFA veya reklam
        kimliği vermez. Etkin ücretli aboneliğin AI hakları ise aynı hesapla web üzerinden de
        kullanılabilir.
      </p>
      <p>
        <strong>Toplamadıklarımız:</strong> Reklam kimliği, konum verisi, kişiler, fotoğraf
        kütüphanesi (profil fotoğrafı için seçtiğiniz tek görsel hariç), sağlık verisi veya üçüncü
        taraf reklam/izleme SDK'ları kullanmıyoruz. Apple ödeme kartı/banka bilgilerinizi bize
        iletmez; ödeme Apple tarafından işlenir. Verilerinizi <strong>satmıyoruz</strong> ve reklam
        amacıyla cihazlar veya hizmetler arasında takip etmiyoruz.
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
        <li><strong>Apple StoreKit / App Store:</strong> abonelik satın alma, yenileme, iptal, geri yükleme ve iade süreçleri için.</li>
        <li><strong>OpenAI:</strong> yalnızca AI araştırmasını siz başlattığınızda; seçtiğiniz varlık ve ilgili piyasa verileri üzerinden rapor üretmek, istekleri güvenli biçimde tekilleştirmek ve kötüye kullanımı önlemek için. Bu işlemde hesabınızdan tek yönlü türetilen takma adlı bir güvenlik kimliği de gönderilir. Ayrıntılar için <a href="https://openai.com/policies/privacy-policy/">OpenAI Gizlilik Politikası</a>.</li>
        <li><strong>Barındırma altyapısı:</strong> verilerin güvenli sunucularda saklanması.</li>
      </ul>
      <p>
        Piyasa fiyatları kamuya açık kaynaklardan (ör. borsa/piyasa veri sağlayıcıları) alınır ve
        kişisel verinizle ilişkilendirilmez.
      </p>

      <h3>AI için açık izin</h3>
      <p>
        AI araştırması ilk kez başlatılmadan önce, hangi verilerin OpenAI'a gönderileceğini açıklayan
        ayrı bir izin ekranı gösterilir. Açıkça onaylamazsanız hiçbir AI isteği gönderilmez ve
        uygulamanın piyasa izleme özelliklerini kullanmaya devam edebilirsiniz. İzninizi Ayarlar'dan
        geri çekebilirsiniz; geri çekme yeni paylaşımları durdurur. AI'a parola, ödeme bilgisi,
        e-posta, profil fotoğrafı veya portföy pozisyonlarınız gönderilmez. Yalnızca seçtiğiniz
        varlık kodu, güncel/geçmiş piyasa bağlamı, hesabınızdan tek yönlü türetilen ve sizi doğrudan
        tanımlamayan takma adlı güvenlik kimliği ile istek tekilleştirme bilgisi gönderilir. Bu
        takma adlı kimlik güvenlik ve kötüye kullanım önleme amacıyla kullanılır. OpenAI'ın verileri işlemesi ve
        saklaması, TSC Forex ile OpenAI arasındaki hizmet yapılandırması ile OpenAI'ın yürürlükteki
        <a href="https://openai.com/policies/privacy-policy/"> gizlilik koşulları</a> kapsamında da
        gerçekleşir.
      </p>

      <h2>5. Saklama Süresi</h2>
      <p>
        Hesap verileriniz, hesabınız aktif olduğu sürece saklanır. Hesabınızı sildiğinizde, tüm
        kişisel verileriniz (izleme listesi, alarmlar, portföy, AI araştırma geçmişi, bildirimler,
        cihaz jetonları, oyunlaştırma verileri, kimlik eşlemeleri ve abonelik erişim kaydı) kalıcı
        olarak ve gecikmeksizin silinir. Yasal/mali kayıt yükümlülüğü kapsamında Apple işlem
        referansları yasal yükümlülük süresince tutulabilir. Aktif veya geri yüklenebilir bir App
        Store aboneliği varken hesabınızı silerseniz; aboneliği daha sonra yalnız gerçek sahibine
        geri bağlayabilmek ve başka hesaba geçirilmesini önlemek için özgün Apple işlem kimliğinin,
        appAccountToken değerinin ve cihazda üretilen abonelik kurtarma anahtarının tek yönlü
        özetlerini içeren kullanıcı bağlantısı kaldırılmış bir kurtarma kaydı tutulabilir. Ham
        kurtarma anahtarı sunucuda saklanmaz ve bu kayıt reklam/profil çıkarma amacıyla kullanılmaz.
        Hesap silindiğinde kullanıcı bağlantısı
        derhal kaldırılan tek yönlü iOS kurulum claim hash'i, yalnız kullanıcı ve doğrulanmış iOS kurulumu başına tek ücretsiz
        AI hakkının tekrar suistimalini önlemek için ücretsiz deneme programı devam ettiği sürece
        tutulur; program sona erdiğinde silinir. Bu kayıt reklam, profil çıkarma veya cihazlar arası
        pazarlama takibi için kullanılmaz. Teknik günlükler kısa süre sonra otomatik temizlenir.
      </p>

      <h2>6. Abonelikler ve Ödemeler</h2>
      <p>
        Premium planlar aylık, otomatik yenilenen App Store abonelikleridir. Satın alma Apple
        hesabınızla ve Apple'ın gösterdiği yerel fiyatla gerçekleşir. Biz kart veya banka
        bilgilerinizi görmeyiz. Aboneliğinizi iPhone Ayarları → Apple Hesabı → Abonelikler'den ya da{" "}
        <a href="https://apps.apple.com/account/subscriptions">Apple abonelik yönetimi</a> üzerinden
        değiştirebilir veya iptal edebilirsiniz. Satın alma sorunlarında Apple'ın{" "}
        <a href="https://reportaproblem.apple.com/">Sorun Bildir / İade İste</a> hizmetini kullanabilirsiniz.
      </p>

      <h2>7. Haklarınız</h2>
      <p>KVKK ve GDPR kapsamında; verilerinize erişme, düzeltme, silme, işlemeyi kısıtlama ve itiraz etme haklarına sahipsiniz. Bu hakları kullanmak için:</p>
      <ul>
        <li>Profil bilgilerinizi Uygulama içinden düzenleyebilirsiniz.</li>
        <li>Hesabınızı ve ilişkili verilerinizi, aşağıda açıklanan yasal kayıt ve amaçla sınırlı kötüye kullanım önleme istisnaları dışında, <strong>Uygulama içinden Ayarlar → Hesabı Sil</strong> ile kalıcı olarak silebilirsiniz.</li>
        <li>Diğer talepler için <a href={`mailto:${PRIVACY_EMAIL}`}>{PRIVACY_EMAIL}</a>.</li>
      </ul>

      <h2>8. Güvenlik</h2>
      <p>
        Parolalar modern algoritmalarla özetlenerek (Argon2id) saklanır, tüm trafik HTTPS/TLS ile
        şifrelenir, oturumlar yenilenebilir/iptal edilebilir jetonlarla korunur. Buna rağmen
        internet üzerinden hiçbir aktarımın %100 güvenli olmadığını hatırlatırız.
      </p>

      <h2>9. Çocukların Gizliliği</h2>
      <p>
        Uygulama 13 yaşından küçük çocuklara yönelik değildir ve bilerek onlardan veri toplamayız.
      </p>

      <h2>10. Değişiklikler</h2>
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
        izleme listesi, fiyat alarmları, portföy takibi, oyunlaştırma ve kaynaklı AI piyasa
        araştırması özellikleri sunan bir
        <strong> bilgilendirme aracıdır</strong>.
      </p>

      <h2>2. Yatırım Tavsiyesi Değildir</h2>
      <p>
        Uygulamadaki hiçbir içerik, fiyat, grafik, bildirim veya AI çıktısı <strong>yatırım, finans,
        hukuk veya vergi tavsiyesi niteliği taşımaz</strong>. AI çıktıları olasılıksal olarak
        üretilir; yanlış, güncelliğini yitirmiş, eksik veya çelişkili olabilir. Alım-satım
        kararlarınız tamamen kendi
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

      <h2>6. Abonelik, Yenileme ve Kotalar</h2>
      <ul>
        <li>Başlangıç, Pro, Uzman ve Elite planları birer aylık otomatik yenilenen aboneliktir.</li>
        <li>Onayladığınızda ödeme Apple hesabınızdan tahsil edilir. Abonelik, mevcut dönemin bitiminden en az 24 saat önce iptal edilmezse otomatik yenilenir; yenileme Apple hesabınızdan tahsil edilir.</li>
        <li>Geçerli yerel fiyat, vergi ve para birimi satın alma ekranında App Store tarafından gösterilir. Web sitesindeki örnek/önerilen fiyat yerine her zaman sistem satın alma ekranındaki fiyat geçerlidir.</li>
        <li>Aboneliği iPhone Ayarları → Apple Hesabı → Abonelikler'den veya <a href="https://apps.apple.com/account/subscriptions">Apple abonelik yönetimi</a> sayfasından değiştirebilir ya da iptal edebilirsiniz.</li>
        <li>"Satın Almaları Geri Yükle", aynı Apple hesabıyla geçerli abonelik erişimini yeniden eşler; yeni bir ücret oluşturmaz.</li>
        <li>Planlara dahil AI araştırması, aktif alarm, izleme listesi, portföy, portföy sağlık raporu, dışa aktarma ve arşiv limitleri satın alma ekranında gösterilir. Kotalar plan dönemine göre yenilenir; kullanılmayan kota sonraki döneme devretmez ve nakde çevrilemez.</li>
        <li>İptal, mevcut ücretli dönemin sonuna kadar erişimi genel olarak sürdürür ve sonraki yenilemeyi durdurur. Plan yükseltme/düşürme ve ücretlendirme zamanı Apple'ın aynı abonelik grubu kurallarına göre uygulanır.</li>
        <li>Daha düşük plana geçildiğinde limit üstündeki mevcut portföy/izleme/alarm kayıtlarınız görüntülenebilir; kullanım yeni plan limitinin altına inene kadar yeni kayıt oluşturmanız engellenebilir.</li>
        <li>Satın alma anında gösterilen plan hakları geçerli dönem için esas alınır. Esaslı fiyat veya hak değişiklikleri yürürlükteki hukuk ve Apple süreçlerine uygun biçimde önceden bildirilir.</li>
        <li>Ücretsiz hesapta kullanıcı ve Apple App Attest ile doğrulanmış fiziksel iOS uygulama kurulumu başına toplam bir AI araştırması bulunur. Bu ücretsiz hak web tarayıcısında sunulmaz; etkin ücretli planın AI hakları web'de kullanılabilir. Hesap/kurulum değiştirme, otomasyon veya başka yollarla kotayı aşmaya çalışmak erişimin sınırlandırılmasına neden olabilir.</li>
        <li>İade kararlarını Apple verir. Uygunluk değerlendirmesi için <a href="https://reportaproblem.apple.com/">reportaproblem.apple.com</a> kullanılmalıdır.</li>
      </ul>

      <h2>7. AI Hizmeti ve Açık Rıza</h2>
      <p>
        AI araştırması, siz özelliği açıkça başlattığınızda ve üçüncü taraf AI paylaşımına izin
        verdiğinizde OpenAI hizmetinden yararlanır. İzin vermemek temel uygulama işlevlerini
        engellemez. AI çıktısının doğruluğu, belirli bir sonuca ulaşması, kesintisiz sunulması veya
        yatırım performansı garanti edilmez. Piyasa koşulları, sağlayıcı erişimi, kota, güvenlik
        kontrolleri ve teknik kapasite nedeniyle istek reddedilebilir ya da gecikebilir.
      </p>

      <h2>8. Fikri Mülkiyet</h2>
      <p>Uygulama, tasarımı, markası ve yazılımı bize aittir ve telif hakkıyla korunur.</p>

      <h2>9. Sorumluluğun Sınırlandırılması</h2>
      <p>
        Uygulama "olduğu gibi" sunulur. Yürürlükteki yasaların izin verdiği azami ölçüde, Uygulamanın
        kullanımından veya fiyat verilerine dayanılarak alınan kararlardan doğan doğrudan veya
        dolaylı zararlardan sorumlu değiliz.
      </p>

      <h2>10. Fesih ve Hesap Silme</h2>
      <p>
        Hesabınızı istediğiniz zaman Uygulama içinden silebilirsiniz. Şartları ihlal etmeniz halinde
        erişiminizi askıya alabilir veya sonlandırabiliriz. Hesap silindiğinde kullanıcı bağlantısı
        kaldırılmış tek yönlü iOS kurulum claim hash'i, yalnız kullanıcı ve doğrulanmış iOS kurulumu başına tek ücretsiz AI
        hakkının tekrar suistimalini önlemek için ücretsiz deneme programı devam ettiği sürece
        tutulabilir; program sona erdiğinde silinir. Hesabın silinmesi App Store aboneliğini
        otomatik olarak iptal etmez; yinelenen ücretleri durdurmak için aboneliğinizi ayrıca Apple
        üzerinden iptal etmelisiniz. Hesabınızı silmeden önce uygulama bu durumu ve abonelik yönetimi
        bağlantısını gösterir.
      </p>

      <h2>11. Uygulanacak Hukuk</h2>
      <p>Bu Şartlar Türkiye Cumhuriyeti kanunlarına tabidir.</p>

      <h2>12. İletişim</h2>
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
        <li><strong>iOS:</strong> Ayarlar sekmesi → "Hesabı Sil" → onaylayın. Apple ile bağlı hesaplarda güvenli silme için Apple ile yeniden doğrulama istenir.</li>
        <li><strong>Web:</strong> Ayarlar → "Hesabı Sil". Apple ile bağlı hesapların silinmesini iOS uygulamasından tamamlayın.</li>
      </ul>
      <p>
        Silme işlemi geri alınamaz; izleme listeniz, alarmlarınız, portföyünüz, bildirimleriniz ve
        oyunlaştırma/AI araştırma verileriniz kaldırılır. Yasal işlem kayıtları ve kullanıcı
        bağlantısı kaldırılmış iOS kurulum claim hash'i için Gizlilik Politikası'ndaki sınırlı saklama
        istisnaları geçerlidir.
      </p>
      <p>
        <strong>Önemli:</strong> Hesabı silmek Apple aboneliğini iptal etmez. Aktif aboneliğiniz
        varsa önce <a href="https://apps.apple.com/account/subscriptions">Apple Abonelikleri</a>
        ekranından iptal edin; aksi halde Apple üzerinden faturalandırma sürebilir.
      </p>

      <h2>Abonelik ve satın alma yardımı</h2>
      <h3>Aboneliğimi nasıl yönetir veya iptal ederim?</h3>
      <p>
        Uygulamada Ayarlar → Aboneliği Yönet'i kullanın veya{" "}
        <a href="https://apps.apple.com/account/subscriptions">Apple Abonelikleri</a> sayfasını açın.
        İptal, mevcut ücretli dönemin sonuna kadar erişiminizi genellikle etkilemez; yenilemeyi durdurur.
      </p>
      <h3>Satın almam görünmüyor.</h3>
      <p>
        Satın alırken kullandığınız Apple hesabıyla giriş yaptığınızı doğrulayın ve ödeme ekranındaki
        "Satın Almaları Geri Yükle" seçeneğine dokunun. Geri yükleme yeni ücret oluşturmaz. Sorun
        sürerse satın alma tarihi, plan adı ve Apple sipariş numarasıyla bize yazın; tam makbuz veya
        Apple hesabı parolanızı göndermeyin.
      </p>
      <h3>Nasıl iade isterim?</h3>
      <p>
        Ödeme ve iadeler Apple tarafından yönetilir. <a href="https://reportaproblem.apple.com/">Apple
        Sorun Bildir</a> sayfasından talep oluşturabilirsiniz; iade uygunluğuna Apple karar verir.
      </p>

      <h2>AI araştırması ve kotalar</h2>
      <p>
        İlk AI araştırmasından önce OpenAI ile paylaşılacak piyasa bağlamı için açık izin istenir.
        İzni reddedebilir veya Ayarlar'dan geri çekebilirsiniz. Plan kotanız dolduysa bir sonraki
        aylık kota dönemini bekleyebilir ya da planınızı yönetebilirsiniz. Kullanılmayan kota
        devretmez. AI raporları bilgilendirme amaçlıdır; doğruluk veya yatırım sonucu garantisi yoktur.
      </p>

      <h2>Sık Sorulan Sorular</h2>
      <h3>Fiyatlar neden gerçek zamanlı görünmüyor?</h3>
      <p>Fiyatlar üçüncü taraf kaynaklardan periyodik olarak güncellenir ve piyasa koşullarına göre kısa gecikmeler olabilir.</p>
      <h3>Bildirim alamıyorum.</h3>
      <p>Cihaz ayarlarından TSC Forex için bildirim izninin açık olduğundan emin olun ve bir fiyat alarmı tanımlayın.</p>
      <h3>Giriş yapmadan kullanabilir miyim?</h3>
      <p>Hayır. iOS uygulaması şu anda piyasa ekranı dahil tüm özellikler için giriş ister; e-posta, Apple, Google veya geçiş anahtarıyla giriş yapabilirsiniz.</p>
      <h3>Ücretsiz AI hakkı neden kullanılmış görünüyor?</h3>
      <p>
        Ücretsiz AI araştırması kötüye kullanımı önlemek için hesap ve Apple App Attest ile
        doğrulanmış fiziksel iOS uygulama kurulumu başına toplam bir kezdir. iOS uygulamasının
        Keychain'de ürettiği rastgele, uygulamaya özel kurulum kimliğinin sunucuda tek yönlü özeti
        ve App Attest doğrulaması kullanılır; bunlar IDFA değildir ve reklam ya da cihazlar arası
        pazarlama takibi için kullanılmaz. Tarayıcı yerel depolaması fiziksel cihaz doğrulaması
        sayılmaz ve web'de ücretsiz deneme sunulmaz. Etkin ücretli planın varsa AI haklarını web'de
        kullanabilirsin. Hata olduğunu düşünüyorsan destek adresine ulaş.
      </p>
    </LegalShell>
  );
}
