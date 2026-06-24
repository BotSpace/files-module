# Files Module

Botspace uchun Go file utility moduli. Runtime tomonidan beriladigan file API
orqali internetdan fayl yuklaydi, faylni o'chiradi, nusxalaydi, text sifatida
o'qiydi va fayl haqida oddiy info chiqaradi.

## Node'lar

### `files.DownloadURL`

Internet URL'dan fayl yuklab Botspace storage'ga upload qiladi.

State:

- `file_uuid`
- `file_name`
- `file_size_bytes`
- `file_content_type`
- `file_source_url`
- `file_error`

### `files.DeleteFile`

`file_uuid` bo'yicha faylni o'chiradi.

State:

- `deleted_file_uuid`
- `file_deleted`
- `file_error`

### `files.CopyFile`

Mavjud fayl UUID'sini o'qib, yangi fayl sifatida qayta upload qiladi.

State:

- `file_uuid`
- `source_file_uuid`
- `file_name`
- `file_size_bytes`
- `file_content_type`
- `file_error`

### `files.FileInfo`

Faylni UUID orqali o'qib, hajm va MIME tipini aniqlaydi.

State:

- `file_uuid`
- `file_size_bytes`
- `file_content_type`
- `file_error`

### `files.ReadText`

Fayl contentini text sifatida `file_text` state'ga yozadi. Katta fayllar uchun
`max_chars` bilan preview limit beriladi.

State:

- `file_text`
- `file_text_truncated`
- `file_size_bytes`
- `file_content_type`
- `file_error`

## Lokal ishga tushirish

```bash
go run .
```

```bash
curl http://localhost:8100/health
curl -X POST http://localhost:8100/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"describe","id":1}'
```

## Test

```bash
go test ./...
go build ./...
```

## Docker

```bash
docker build -t files-module .
docker run -p 8100:8100 files-module
```

## Eslatma

`UploadFile`, `GetFile`, `DeleteFile` faqat Botspace runtime file API berganda
ishlaydi. Oddiy lokal `go run .` paytida `describe`, `docs`, helper testlar
ishlaydi, lekin haqiqiy upload/get/delete uchun platforma konteksti kerak.

To'liq SDK kontrakti uchun [`SDK.md`](./SDK.md) faylini ko'ring.
