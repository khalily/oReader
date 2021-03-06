from lxml import etree
from urllib2 import urlopen, urlparse
from ..models import Feed, Item


def parse(db, source):
    parser = FeedParser()
    if not parser.parse(source):
        return

    feed = Feed(**parser.feed)
    db.session.add(feed)
    for item in parser.items:
        db.session.add(Item(**item))
    db.session.commit()


class FeedParser(object):
    def __init__(self):
        self._feed = self._items = []

    def parse(self, source):
        try:
            self._root = etree.parse(source).getroot()
            if self._root is None:
                return False
            self._remove_namespace()
            result = self._parse()
        except etree.LxmlError, e:
            print e
            return False
        return result

    @property
    def feed(self):
        return self._feed

    @property
    def items(self):
        return self._items

    def _parse(self):
        tag = self._root.tag
        if tag == 'rss':
            return self._parse_rss()
        elif tag == 'feed':
            return self._parse_atom()

    def _remove_namespace(self):
        for e in self._root.iter():
            if not hasattr(e.tag, 'find'): continue;
            i = e.tag.find('}')
            if i >= 0:
                e.tag = e.tag[i+1:]

    def _get_text(self, element):
        if element is not None and hasattr(element, 'text'):
            if element.text is not None:
                return element.text
        return ""

    def _get_img(self, link):
        print link
        url = urlparse.urlparse(link)
        new_url = url.scheme + '://' + url.netloc + '/favicon.ico'
        print new_url
        try:
            urlopen(new_url)
        except:
            return None
        return new_url

    def _remove_tags(self, tag_text):
        try:
            eh = etree.HTML(tag_text)
        except:
            print 'error'
            return ''
        return ''.join(eh.itertext())

    def _simplify_description(self, description):
        data = ''
        if len(description):
            data = self._remove_tags(description)
            if len(data) > 50:
                data = data[0:50]
            if len(data) > 0 and not data.endswith('.'):
                data += '...'
        return data

    def _parse_rss(self):
        channel = self._root.find('channel')

        if channel is None:
            return False
        self._feed = {
            'title': self._get_text(channel.find('title')),
            'description': self._simplify_description(self._get_text(channel.find('description'))),
            'last_build_date': self._get_text(channel.find('lastBuildDate'))
        }

        link = channel.find('link')
        while link is not None and link.text is None:
            link = link.getnext()
        if link is not None:
            self._feed['link'] = link.text
            self._feed['img'] = self._get_img(link.text)

        for item in channel.findall('item'):
            self._items.append(
                {
                    'title': self._get_text(item.find('title')),
                    'link': self._get_text(item.find('link')),
                    'description': self._simplify_description(self._get_text(item.find('description'))),
                    'pub_date': self._get_text(item.find('pubDate')),
                    'creator': self._get_text(item.find('creator')),
                    'content': self._get_text(item.find('encoded')),
                }
            )
        return True


    def _parse_atom(self):
        feed = self._root
        self._feed = {
            'title': self._get_text(feed.find('title')),
            'description': self._simplify_description(self._get_text(feed.find('subtitle'))),
            'last_build_date': self._get_text(feed.find('updated')),
        }

        link = feed.find('link')
        while link is not None:
            attrib = link.attrib
            if attrib.get('rel') != 'self':
                link = attrib.get('href')
                self._feed['link'] = link
                self._feed['img'] = self._get_img(link)
                break
            link = link.getnext()

        for entry in feed.findall('entry'):
            item = {
                    'title': self._get_text(entry.find('title')),
                    'description': self._simplify_description(self._get_text(entry.find('summary'))),
                    'content': self._get_text(entry.find('content')),
                    'pub_date': self._get_text(entry.find('published')),
                    'updated': self._get_text(entry.find('updated')),
            }
            link = entry.find('link')
            if link is not None:
                href = link.attrib.get('href')
                item['link'] = href if href is not None else ""
            author = entry.find('author')
            if author is not None:
                name = author.find('name')
                item['creator'] = name.text if name is not None else ""

            self._items.append(item)
        return True
