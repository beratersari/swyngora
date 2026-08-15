from swyngora_ai.graph.route import classify_route


def test_named_coin_news_is_web_not_desk_note():
    r = classify_route("THY longest flight and fuel costs")
    assert r.web
    assert not r.desk_note
    assert not r.tape


def test_price_question_is_tape_not_brief():
    r = classify_route("What is BTC price on binance?")
    assert r.tape
    assert not r.desk_note


def test_nasdaq_filing_is_web_only():
    r = classify_route("AAPL 8-K filing")
    assert r.web
    assert not r.tape
    assert not r.desk_note


def test_bist_last_is_tape():
    r = classify_route("THYAO last on bist")
    assert r.tape


def test_book_keywords():
    r = classify_route("BTC order book walls and liquidations")
    assert r.book
    assert r.tape


def test_paper_and_account():
    assert classify_route("place a paper OCO on BTC").paper
    assert classify_route("add ETH to my watchlist").account
    assert classify_route("preview import of my export").account


def test_debate_only_when_asked():
    d = classify_route("Should I lean long BTC for a day or two?")
    assert d.debate
    assert d.tape
    assert d.web
    assert not classify_route("BTC RSI on binance 1h").debate


def test_social():
    r = classify_route("twitter sentiment on ETH")
    assert r.social


def test_greeting_does_not_force_desk():
    r = classify_route("hello")
    assert not r.desk_note
    assert not r.tape


def test_analysis_request_gets_desk_note():
    r = classify_route("BTC analizi yap 1-2 gün")
    assert r.desk_note
    assert r.tape
    assert r.web
