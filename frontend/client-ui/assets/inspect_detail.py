
from html.parser import HTMLParser

class DetailParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.chapters = []
        self.covers = []

    def handle_starttag(self, tag, attrs):
        attr_dict = dict(attrs)
        if tag == 'a' and 'href' in attr_dict:
            href = attr_dict['href']
            if '/chuong-' in href:
                self.chapters.append((href, attr_dict.get('title', '')))
        if tag == 'img':
            src = attr_dict.get('src') or attr_dict.get('data-src') or ''
            if src:
                self.covers.append(src)

with open('d:/MMO/frontend/client-ui/assets/sample_detail.html', encoding='utf-8') as f:
    p = DetailParser()
    p.feed(f.read())

print('Cover images found:', p.covers[:5])
print('Total chapters found:', len(p.chapters))
print('Sample chapters:', p.chapters[:5])
