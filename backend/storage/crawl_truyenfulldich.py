import urllib.request, ssl, json, os, re, html
from PIL import Image

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE
opener = urllib.request.build_opener(urllib.request.HTTPSHandler(context=ctx))
opener.addheaders = [('User-Agent', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)')]
urllib.request.install_opener(opener)

def clean_cover_watermark(image_path):
    """Xóa chữ kiểu truyenfull.vn ở phần dưới ảnh bìa"""
    try:
        with Image.open(image_path) as img:
            img = img.convert('RGB')
            w, h = img.size
            y1 = int(h * 0.88)
            y2 = h - 1
            x1 = int(w * 0.08)
            x2 = int(w * 0.92)
            for x in range(x1, x2):
                top_color = img.getpixel((x, max(0, y1 - 1)))
                bot_color = img.getpixel((x, min(h - 1, y2)))
                for y in range(y1, y2 + 1):
                    alpha = (y - y1) / max(1, (y2 - y1))
                    r = int(top_color[0] * (1 - alpha) + bot_color[0] * alpha)
                    g = int(top_color[1] * (1 - alpha) + bot_color[1] * alpha)
                    b = int(top_color[2] * (1 - alpha) + bot_color[2] * alpha)
                    img.putpixel((x, y), (r, g, b))
            img.save(image_path, 'JPEG', quality=92)
            print('✅ Đã xóa watermark truyenfull.vn khỏi ảnh:', image_path)
    except Exception as e:
        print('Lỗi xử lý xóa watermark ảnh:', e)

def clean_text_data(raw_text):
    """Clean data trước khi lưu: xóa tags HTML, link spam và watermark truyenfull"""
    if not raw_text: return ""
    text = raw_text.replace('<br />', '\n').replace('<br/>', '\n').replace('<br>', '\n')
    text = text.replace('<p>', '').replace('</p>', '\n')
    text = re.sub(r'(?i)<[^>]*>', '', text)
    text = html.unescape(text)

    watermarks = [
        r'(?i)truyenfull\.vn|truyenfull\.live|truyenfulldich\.top|truyenfull\.com|tangthuvien\.net',
        r'(?i)^\s*nguồn\s*:\s*.*',
        r'(?i)bạn đang (đọc|xem|theo dõi) truyện (tại|trên)\s*.*',
        r'(?i)truyện được (đăng|dịch|re-?up|copy) (duy nhất )?tại\s*.*',
        r'(?i)chúc bạn đọc truyện vui vẻ tại\s*.*',
        r'(?i)(like fanpage|tham gia group facebook|ủng hộ nhóm dịch).*',
        r'(?i)https?://[^\s]+',
    ]
    lines = []
    for line in text.split('\n'):
        line = line.strip()
        if not line:
            lines.append("")
            continue
        if any(re.search(pat, line) for pat in watermarks):
            continue
        lines.append(line)
    return re.sub(r'\n{3,}', '\n\n', '\n'.join(lines)).strip()

def extract_inertia(html):
    mark = 'type="application/json">'
    idx = html.find(mark)
    if idx == -1: return None
    s = idx + len(mark)
    end = html.find('</script>', s)
    return json.loads(html[s:end])

# Crawl 1 sample novel: Tao Hoa Chi Mon
novel_url = 'https://truyenfulldich.top/tao-hoa-chi-mon'
html = urllib.request.urlopen(novel_url).read().decode('utf-8', errors='ignore')
data = extract_inertia(html)
story = data['props']['storyProp']

slug = story['alias']
print('Story crawled:', slug)

os.makedirs(f'd:/MMO/backend/storage/{slug}/chapters', exist_ok=True)
os.makedirs(f'd:/MMO/storage/{slug}/chapters', exist_ok=True)

# Download cover & clean watermark
cover_url = story['image']
cover_local = f'/covers/{slug}.jpg'
cover_dest = f'd:/MMO/frontend/client-ui/covers/{slug}.jpg'
try:
    urllib.request.urlretrieve(cover_url, cover_dest)
    clean_cover_watermark(cover_dest)
    print('Cover saved locally and cleaned at covers/' + slug + '.jpg')
except Exception as e:
    print('Cover download failed:', e)

novel_obj = {
    'id': slug,
    'title': clean_text_data(story['name']),
    'slug': slug,
    'author_name': clean_text_data(story['authors'][0]['name']) if story['authors'] else 'Đang cập nhật',
    'genres': [clean_text_data(c['name']) for c in story['categories']],
    'description': clean_text_data(story['description']),
    'cover_url': cover_local,
    'status': 'Full' if story['status'] in [1, 2] else 'Đang ra',
    'total_chapters': story['total_chapters'],
    'actual_chapters': 5,
    'views_count': story['view'],
    'rating_score': 9.8,
    'source_url': novel_url
}

with open(f'd:/MMO/backend/storage/{slug}/novel.json', 'w', encoding='utf-8') as f:
    json.dump(novel_obj, f, ensure_ascii=False, indent=2)
with open(f'd:/MMO/storage/{slug}/novel.json', 'w', encoding='utf-8') as f:
    json.dump(novel_obj, f, ensure_ascii=False, indent=2)

# Crawl 5 chapters (clean data trước khi lưu)
for c_num in range(1, 6):
    chap_url = f'{novel_url}/chuong-{c_num}'
    c_html = urllib.request.urlopen(chap_url).read().decode('utf-8', errors='ignore')
    c_data = extract_inertia(c_html)
    chap = c_data['props']['chapterProp']
    title_str = clean_text_data(f"{chap['subname']}: {chap['name']}")
    clean_content = clean_text_data(chap['content'])
    c_obj = {
        'id': f'{slug}-chuong-{c_num}',
        'novel_id': slug,
        'chapter_number': c_num,
        'title': title_str,
        'content_text': clean_content,
        'word_count': len(clean_content.split()),
        'source_url': chap_url
    }
    with open(f'd:/MMO/backend/storage/{slug}/chapters/{c_num}.json', 'w', encoding='utf-8') as f:
        json.dump(c_obj, f, ensure_ascii=False, indent=2)
    with open(f'd:/MMO/storage/{slug}/chapters/{c_num}.json', 'w', encoding='utf-8') as f:
        json.dump(c_obj, f, ensure_ascii=False, indent=2)
    print(f'Saved chapter {c_num}')

print('DONE!')
