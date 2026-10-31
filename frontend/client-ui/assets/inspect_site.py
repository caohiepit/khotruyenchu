
from html.parser import HTMLParser

class MyParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.links = []

    def handle_starttag(self, tag, attrs):
        attr_dict = dict(attrs)
        if tag == 'a' and 'href' in attr_dict:
            self.links.append((attr_dict['href'], attr_dict.get('title', '')))

with open('d:/MMO/frontend/client-ui/assets/truyenfulldich_home.html', encoding='utf-8') as hf:
    p = MyParser()
    p.feed(hf.read())

for l, t in p.links[:25]:
    print('Link:', l, '| Title:', t)
