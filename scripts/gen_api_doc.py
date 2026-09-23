#!/usr/bin/env python3
import json, re, collections

d = json.load(open('api/docs/swagger.json'))
defs = d['definitions']
paths = d['paths']

TAG_TITLES = {
    'auth': "Autentifikatsiya (Login/Register/Token)",
    'profile': "Foydalanuvchi profili",
    'user': "Foydalanuvchi (qo'shimcha)",
    'category': "Bo'limlar (Category)",
    'topic': "Mavzular (Topic)",
    'case': "Klinik Case'lar",
    'favorite': "Sevimlilar (Favorites)",
    'simulation': "Simulyatsiya dvigateli",
    'debrief': "Debriefing hisoboti",
    'setting': "Sozlamalar (limit, ovoz)",
    'promocode': "Promokodlar",
    'tariff': "Tariflar (Obuna/Coin paket)",
    'subscription': "Obuna",
    'notification': "Bildirishnomalar",
    'banner': "Bannerlar",
    'about': "Biz haqimizda",
    'faq': "FAQ",
    'contact': "Kontaktlar",
    'appRoute': "App Route (ilova ichki linklar)",
    'partner': "Hamkorlar",
    'study-plan': "O'quv rejasi (eslatma)",
    # /web (admin panel) uchun qo'shimcha guruhlar
    'admin': "Adminlar",
    'role': "Rollar (RBAC)",
    'custom-permission': "Maxsus ruxsatlar",
    'dashboard': "Dashboard/Analitika",
    'ai-prompt': "AI Promptlari va sinov",
    'coin-transaction': "Tangalar monitoringi",
    'order': "Buyurtmalar/To'lovlar",
    'level': "Levellar (XP)",
    'file-upload': "Fayl yuklash",
}

MOBILE_TAG_ORDER = ['auth','profile','user','category','topic','case','favorite','simulation',
    'debrief','setting','promocode','tariff','subscription','study-plan',
    'notification','banner','about','faq','contact','appRoute','partner']

WEB_TAG_ORDER = ['auth','admin','role','custom-permission','dashboard','category','topic','case',
    'level','ai-prompt','tariff','order','promocode','partner','coin-transaction','setting',
    'notification','banner','about','faq','contact','appRoute','user','file-upload']

def resolve_ref(ref):
    name = ref.split('/')[-1]
    return name, defs.get(name, {})

def type_str(schema, depth=0):
    if schema is None:
        return '—'
    if '$ref' in schema:
        name, sub = resolve_ref(schema['$ref'])
        return name.replace('models.', '')
    t = schema.get('type')
    if t == 'array':
        items = schema.get('items', {})
        return type_str(items, depth) + '[]'
    if t == 'object' and 'additionalProperties' in schema:
        return 'object (key->' + type_str(schema['additionalProperties'], depth) + ')'
    if t:
        fmt = schema.get('format')
        return t + (f'({fmt})' if fmt else '')
    return 'object'

def fields_table(schema, seen=None, depth=0):
    """schema -> markdown bullet list of field: type"""
    if seen is None:
        seen = set()
    if schema is None:
        return []
    if '$ref' in schema:
        name, sub = resolve_ref(schema['$ref'])
        if name in seen or depth > 1:
            return [f"_(→ {name.replace('models.','')})_"]
        seen = seen | {name}
        return fields_table(sub, seen, depth)
    if schema.get('type') == 'array':
        items = schema.get('items', {})
        inner = fields_table(items, seen, depth)
        return ['**[ ] massiv, har bir element:**'] + ['  ' + l for l in inner] if inner else [type_str(schema)]
    props = schema.get('properties')
    if not props:
        return []
    lines = []
    for name, sub in props.items():
        t = type_str(sub, depth+1)
        lines.append(f"- `{name}`: {t}")
    return lines

def param_line(p):
    req = ' (majburiy)' if p.get('required') else ''
    loc = p.get('in')
    if loc == 'body':
        name, sub = None, p.get('schema')
        lines = fields_table(sub)
        body = '\n'.join('  ' + l for l in lines) if lines else '  (bo\'sh)'
        return f"**Body** ({type_str(sub)}){req}:\n{body}"
    default = f", default=`{p['default']}`" if 'default' in p else ''
    return f"- `{p['name']}` ({loc}, {p.get('type','string')}{default}){req}: {p.get('description','')}"

AUTH_OVERRIDE = {
    # (path, method) -> auth majburiymi. api/api.go'dagi haqiqiy route guruhlash (.Use(mid.NewAuth))
    # asosida - swagger @Security annotatsiyasi ba'zan (bitta handler ikkita /web va /mobile
    # route'ga xizmat qilganda) noto'g'ri/chalkash bo'ladi, shu yerda to'g'irlanadi.
    ('/mobile/app-route', 'get'): False,
    ('/mobile/app-route/{id}', 'get'): False,
}

def gen_tag(tag, ops):
    out = [f"## {TAG_TITLES.get(tag, tag)}\n"]
    for path, method, op in ops:
        auth = 'BearerAuth' in [list(s.keys())[0] for s in op.get('security', [])] if op.get('security') else False
        if (path, method) in AUTH_OVERRIDE:
            auth = AUTH_OVERRIDE[(path, method)]
        out.append(f"### `{method.upper()} {path}`")
        summary = op.get('summary', '')
        desc = op.get('description', '')
        if summary:
            out.append(f"**{summary}**")
        if desc:
            out.append(desc)
        out.append(f"- Auth: {'kerak (Bearer token)' if auth else 'kerak emas'}")
        params = op.get('parameters', [])
        non_body = [p for p in params if p.get('in') != 'body']
        body = [p for p in params if p.get('in') == 'body']
        if non_body:
            out.append("**Parametrlar:**")
            for p in non_body:
                out.append(param_line(p))
        if body:
            out.append(param_line(body[0]))
        responses = op.get('responses', {})
        ok = responses.get('200') or responses.get('201')
        if ok and ok.get('schema'):
            out.append(f"**Javob (200):**")
            lines = fields_table(ok['schema'])
            if lines:
                out.extend(lines)
            else:
                out.append(f"`{type_str(ok['schema'])}`")
        out.append("")
    return '\n'.join(out)

def build(prefix, outfile, title, tag_order):
    grouped = collections.defaultdict(list)
    for path, methods in paths.items():
        if not path.startswith(prefix):
            continue
        for method, op in methods.items():
            for tag in op.get('tags', ['other']):
                grouped[tag].append((path, method, op))

    login_note = ("login/OTP oqimi orqali (`POST /mobile/auth/google` yoki `/mobile/auth/user/otp/*`)"
        if prefix == '/mobile' else "admin login orqali (`POST /web/auth/admin/login`, email+parol)")
    header = f"""# {title}

> **Diqqat:** bu fayl `api/docs/swagger.json` asosida avtomatik generatsiya qilingan
> (`scripts/gen_api_doc.py` skripti orqali — qo'lda tahrirlamang, backend o'zgarsa qayta
> generatsiya qilinadi: `swag init -g api/api.go -o api/docs && python3 scripts/gen_api_doc.py`).
>
> - **Base URL:** deploy qilingan domen (masalan `https://api.tibsphereai.uz`), path'lar shu yerda ko'rsatilganidek qo'shiladi (masalan `{prefix}/category`).
> - **Auth:** {login_note} olingan `access_token`ni `Authorization: Bearer <access_token>` header bilan yuboring. Token muddati tugasa (`401 "access token expired"`), `POST /auth/token/refresh` ga `{{"refresh_token": "..."}}` yuborib yangi juft token oling (bu endpoint `{prefix}` ostida emas, root'da).
> - **Til:** ko'p javoblarda matnlar tilga bog'liq bo'lsa, `Accept-Language: uz|ru|en` header yuboriladi (default `uz`).
> - **Pagination:** ro'yxat endpointlari odatda `limit`, `page` query parametr oladi.
> - **Xatolik formati:** xatolik bo'lsa `4xx/5xx` status + `{{"error": "..."}}` shaklidagi JSON qaytadi.
> - Pastdagi "Auth" qatori shu endpoint uchun haqiqatda routerda auth middleware borligini bildiradi (`api/api.go`dagi guruhlashga qarab tekshirilgan, ba'zi joylarda swagger annotatsiyasidan ko'ra ishonchliroq).

---

"""
    body_parts = [header]
    for tag in tag_order:
        if tag in grouped:
            body_parts.append(gen_tag(tag, grouped[tag]))
    # any leftover tags not in tag_order
    for tag in grouped:
        if tag not in tag_order:
            body_parts.append(gen_tag(tag, grouped[tag]))

    with open(outfile, 'w') as f:
        f.write('\n'.join(body_parts))
    print('wrote', outfile, 'paths:', sum(len(v) for v in grouped.values()))

build('/mobile', 'API_MOBILE.md', "📱 Mobil ilova API — Frontend uchun qo'llanma", MOBILE_TAG_ORDER)
build('/web', 'API_WEB.md', "🖥️ Admin panel API — Frontend (web) uchun qo'llanma", WEB_TAG_ORDER)
