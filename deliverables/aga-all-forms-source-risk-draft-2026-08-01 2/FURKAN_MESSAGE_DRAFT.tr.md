# Furkan abi için Türkçe mesaj taslağı

Merhaba Furkan abi,

Bu ZIP, AGA arşivindeki yalnızca Form 048 için değil, arşivde bulunan bütün
52 form için hazırlanmış kaynak ve risk inceleme taslağıdır. Arşiv güvenli
intake sınırında stream/hash edilmiştir; ham ZIP/PDF byte'ları repository'ye
ve bu pakete kopyalanmamıştır.

Paketin mevcut durumu `candidate-only`'dir. Arşivde 1 register ve 52 form
olmak üzere 53 PDF kaydı doğrulandı. 31 formda toplam 1.310 adet soru biçimli
aday sınır tespit edildi; 21 form da soru sınırı bulunmayan başvuru, atama,
bilgi veya değerlendirme formları olarak envantere dahil edildi. Form 048'in
28 sorusu daha önce hash ile doğrulanmış dikey dilimden korunuyor; diğer soru
metinleri parser tarafından çıkarılmış aday sınırlardır, henüz kabul edilmiş
sorular değildir.

Her form ve soru için:

- AGA içinde görünen NAMCAR/NAMCATS referansları ayrı bir kaynak eşleme
  kuyruğuna alındı;
- yerel NAMCATS Part 139 hash'i olan aday kaynak ve henüz yerel byte/hash'i
  bulunmayan resmi NAMCAR Part 139 (2023) URL adayı ayrıştırıldı;
- kaynak eşlemesi `SOURCE_MAPPING_REQUIRED` ve kaynak yetkisi
  `NOT_ATTESTED` bırakıldı;
- risk bandı yalnızca kelime-temelli bir ön yorumdur (`PROPOSED_*`),
  `safetyCritical` kararı ve bulgu şiddeti değildir;
- karar alanı `NOT_SUPPLIED`, aday importu `NOT_IMPORTED`, yayınlama
  `NOT_AUTHORIZED` bırakıldı.

Senden/gerçek Admin ve kaynak sahibinden şu incelemeyi rica ediyoruz:

1. 52 formun kimliğini, başlığını, sürüm/tarihini ve kapsamını doğrulayın.
   Soru çıkmayan 21 formun gerçekten protokolsüz form mu, yoksa extraction
   düzeltmesi mi olduğunu belirtin.
2. 1.310 aday sınırın her biri için exact `proposalId` ve `textDigest` ile
   actor-bound `ACCEPT`, `SPLIT`, `MERGE`, `TRANSCRIBE` veya `EXCLUDE` kararı
   verin; PDF sayfasını/locator'ı ve gerekçeyi yazın.
3. Her AGA referansını yürürlükteki resmi belgeye (`source mapping`), immutable source hash'ine,
   effective date'e, maddeye, sayfaya/locator'a, applicability kararına ve
   named source owner'a bağlayın. NAMCAR Part 139 (2023) için tam byte/hash
   ve kaynak sahibi teyidi özellikle eksik kapıdır.
4. Önerilen risk bandını ve safety-critical işaretini onaylayın veya düzeltin;
   gerekçe yazın. Bunlar sistem tarafından otomatik hukuki karar ya da bulgu
   şiddeti olarak kullanılmayacaktır.
5. Ek aday importu için form ID kapsamı, batch limiti, actor ve decision ID
   içeren isimli bir Phase 2 yetkisi verin. Functional assignment, Department
   Manager yayını ve production/release bu yetkinin dışında ayrı kapılardır.

ZIP içindeki `TASK9_ALL_FORMS_EXPANSION_AUTHORIZATION_TEMPLATE.json` bu
alanları boş olarak gösterir; gerçek karar yerine geçmez. `README.md` dosyası
durumları ve kaynak/yetki sınırlarını, CSV/JSON dosyaları da satır bazlı
inceleme kuyruğunu açıklar.

Mevcut ürün durumu: `candidate-only`

Release durumu: `release pending`

`production-ready`: established değil.

Bu pakette source-owner attestation, regulatory applicability, functional
assignment provisioning, Department Manager teknik onayı, publication,
deployment veya production onayı istenmiyor; bunlar gerçek yetkili kararları
gelene kadar `blocked` kalıyor.
