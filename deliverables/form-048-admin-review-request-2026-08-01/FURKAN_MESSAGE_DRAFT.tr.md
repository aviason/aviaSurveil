# Furkan için Türkçe mesaj taslağı

Merhaba Furkan abi,

Governed AGA Checklist Intake çalışmasının Gate 0, Task 1-8 ve source-backed
handoff kapsamı yerel olarak tamamlandı ve doğrulandı. AGA arşivi Admin-only
intake sınırından stream/hash edilerek 53 PDF kaydı (1 register ve 52 form)
olarak doğrulandı. Ham ZIP/PDF byte'ları repository'ye kopyalanmadı; yeni
yetkilendirme kapsamında yalnızca hash'e bağlı 28 kaynak soru metni ve
provenance içeren türetilmiş paket oluşturuldu. Ürün durumu `candidate-only`.

Task 9'un yerel mekanizması çalışıyor; kalan tek kapı insan tarafından
verilecek gerçek Admin kararlarıdır. Form 048 için şunlara ihtiyaç var:

1. Kimlik çakışmasında seçilecek insan-okunabilir Form 048 kimliği ve karar
   gerekçesi.
2. 28 görünür soru sınırının her biri için actor-bound karar:
   `ACCEPT`, `SPLIT`, `MERGE`, `TRANSCRIBE` veya `EXCLUDE`; exact proposal
   ID/digest, PDF sayfa/locator ve gerekçe.
3. Soruları literal `SOURCE_MAPPING_REQUIRED` olarak bırakan tek bir
   immutable `EXISTING_CHECKLIST_CANDIDATE` oluşturma yetkisi.
4. Ek formlar için isimli, kapsamı ve batch limiti belirli Phase 2
   candidate-only genişletme yetkisi.

Ekli Türkçe PDF, AGA PDF düzenine benzer numaralı bir Admin karar formudur ve
Form 048'in hash ile doğrulanmış kaynak PDF'sindeki 28 literal protocol
question metnini, AGA kodlarını, PDF sayfalarını ve görünür NAMCAR/NAMCATS
referanslarını içerir. Extraction kararları hâlâ boş (`NOT_SUPPLIED`) bırakıldı;
her karar exact intake packet proposal ID/digest'ine bağlanmalıdır. Boş,
tahmini veya cross-batch kararlar sistem tarafından fail-closed reddedilir.

Bu pakette regulatory source mapping, source-owner attestation, Department
Manager teknik onayı, publication, functional-assignment yetkisi, deployment
ve production onayı istenmiyor. Bunlar ayrı kapılar olarak `blocked` kalıyor.

Kararlar sağlandığında sistem bunları exact parser/file/manifest/packet
digest'lerine bağlayarak gerçek Form 048 candidate/source-gap Draft'ını
oluşturacak ve Task 9'u bounded, stop-on-error candidate import olarak
çalıştıracak.

Mevcut ürün durumu: `candidate-only`

Release durumu: `release pending`

`production-ready`: established değil.
