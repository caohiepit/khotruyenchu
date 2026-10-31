import urllib.request, ssl, json, os

ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE
opener = urllib.request.build_opener(urllib.request.HTTPSHandler(context=ctx))
opener.addheaders = [('User-Agent', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)')]
urllib.request.install_opener(opener)

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

# Download cover
cover_url = story['image']
cover_local = f'/covers/{slug}.jpg'
try:
    urllib.request.urlretrieve(cover_url, f'd:/MMO/frontend/client-ui/covers/{slug}.jpg')
    print('Cover saved locally at covers/' + slug + '.jpg')
except Exception as e:
    print('Cover download failed:', e)

novel_obj = {
    'id': slug,
    'title': story['name'],
    'slug': slug,
    'author_name': story['authors'][0]['name'] if story['authors'] else 'Đang cập nhật',
    'genres': [c['name'] for c in story['categories']],
    'description': story['description'].replace('<p>', '').replace('</p>', '\n').strip(),
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

# Crawl 5 chapters
for c_num in range(1, 6):
    chap_url = f'{novel_url}/chuong-{c_num}'
    c_html = urllib.request.urlopen(chap_url).read().decode('utf-8', errors='ignore')
    c_data = extract_inertia(c_html)
    chap = c_data['props']['chapterProp']
    title_str = f"{chap['subname']}: {chap['name']}".strip()
    c_obj = {
        'id': f'{slug}-chuong-{c_num}',
        'novel_id': slug,
        'chapter_number': c_num,
        'title': title_str,
        'content_text': chap['content'].replace('<br />', '\n').replace('<br/>', '\n').replace('<br>', '\n').strip(),
        'word_count': len(chap['content'].split()),
        'source_url': chap_url
    }
    with open(f'd:/MMO/backend/storage/{slug}/chapters/{c_num}.json', 'w', encoding='utf-8') as f:
        json.dump(c_obj, f, ensure_ascii=False, indent=2)
    with open(f'd:/MMO/storage/{slug}/chapters/{c_num}.json', 'w', encoding='utf-8') as f:
        json.dump(c_obj, f, ensure_ascii=False, indent=2)
    print(f'Saved chapter {c_num}')

print('DONE!')
