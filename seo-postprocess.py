#!/usr/bin/env python3
"""Stamp per-page SEO into the mdBook output.

mdBook's head.hbs has no variable for a page's own URL, so the canonical
link, og:url and the JSON-LD are added here, after `mdbook build`, from the
file name. Idempotent: a page already stamped is left alone.

    seo-postprocess.py build/html
"""
import html
import json
import pathlib
import re
import sys

BASE = "https://book.pflow.xyz/"
BOOK = "Petri Nets as a Universal Abstraction"
# GA4 measurement ID (G-…). Empty = no analytics injected. Set once the
# book.pflow.xyz property (or a second web stream on the pflow.xyz property)
# exists; measurement IDs are public, they ship in the HTML either way.
GA_MEASUREMENT_ID = ""
MARK = "<!-- seo-postprocess -->"
SKIP = {"404.html", "print.html"}


def page_url(name: str) -> str:
    return BASE if name == "index.html" else BASE + name


def stamp(path: pathlib.Path) -> bool:
    doc = path.read_text(encoding="utf-8")
    if MARK in doc or "</head>" not in doc:
        return False
    m = re.search(r"<title>(.*?)</title>", doc, re.S)
    title = html.unescape(m.group(1)).strip() if m else BOOK
    m = re.search(r'<meta name="description" content="(.*?)"', doc)
    desc = html.unescape(m.group(1)) if m else ""
    url = page_url(path.name)
    ld = {
        "@context": "https://schema.org",
        "@type": "WebPage",
        "url": url,
        "name": title,
        "description": desc,
        "isPartOf": {"@type": "Book", "name": BOOK, "url": BASE, "author": {"@type": "Person", "name": "Matt York", "url": "https://stackdump.com"}},
    }
    block = "\n".join([
        MARK,
        f'<link rel="canonical" href="{url}">',
        '<meta property="og:type" content="article">',
        f'<meta property="og:site_name" content="{html.escape(BOOK)}">',
        f'<meta property="og:title" content="{html.escape(title)}">',
        f'<meta property="og:description" content="{html.escape(desc)}">',
        f'<meta property="og:url" content="{url}">',
        f'<meta property="og:image" content="{BASE}figures/ch01-traffic-light.svg">',
        '<meta name="twitter:card" content="summary">',
        f'<meta name="twitter:title" content="{html.escape(title)}">',
        f'<meta name="twitter:description" content="{html.escape(desc)}">',
        f'<script type="application/ld+json">{json.dumps(ld, ensure_ascii=False)}</script>',
    ] + ([
        f'<script async src="https://www.googletagmanager.com/gtag/js?id={GA_MEASUREMENT_ID}"></script>',
        '<script>window.dataLayer=window.dataLayer||[];function gtag(){dataLayer.push(arguments);}'
        f"gtag('js',new Date());gtag('config','{GA_MEASUREMENT_ID}');</script>",
    ] if GA_MEASUREMENT_ID else []) + [
        "",
    ])
    path.write_text(doc.replace("</head>", block + "</head>", 1), encoding="utf-8")
    return True


def main(root: str) -> int:
    n = 0
    for p in sorted(pathlib.Path(root).glob("*.html")):
        if p.name in SKIP:
            continue
        n += stamp(p)
    print(f"seo-postprocess: stamped {n} pages under {root}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1] if len(sys.argv) > 1 else "build/html"))
