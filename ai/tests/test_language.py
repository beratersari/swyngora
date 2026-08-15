from swyngora_ai.graph.desk import _packet
from swyngora_ai.graph.facts import extract_market_facts
from swyngora_ai.grounding import apply_grounding
from swyngora_ai.language import (
    detect_reply_lang,
    disclaimer_for,
    empty_reply_for,
    has_advice_disclaimer,
)


def test_detects_turkish_ascii_and_letters():
    assert detect_reply_lang("BTC nedir bugun?") == "tr"
    assert detect_reply_lang("ETH fiyatı nedir?") == "tr"
    assert detect_reply_lang("What is BTC doing today?") == "en"


def test_disclaimer_and_empty_match_user_lang():
    assert "yatırım tavsiyesi" in disclaimer_for("BTC analizi yap")
    assert "not financial advice" in disclaimer_for("BTC analysis please")
    assert "Yanıt üretemedim" in empty_reply_for("nedir bu?")
    assert "could not produce" in empty_reply_for("what is this?")


def test_has_advice_disclaimer_tr_and_en():
    assert has_advice_disclaimer("... Not financial advice.")
    assert has_advice_disclaimer("Yalnızca bilgilendirme — yatırım tavsiyesi değildir.")
    assert not has_advice_disclaimer("BTC looks heavy.")


def test_packet_instructs_turkish_reply():
    text = _packet({"message": "BTC nedir?", "facts": {}})
    assert "TÜRKÇE" in text or "Türkçe" in text
    assert "Sadece soruyu" in text
    assert "Bottom line" in text


def test_grounding_note_is_turkish_when_user_is():
    facts = extract_market_facts('{"lastPrice":"100.25"}')
    out = apply_grounding(
        "Son fiyat 99999.12",
        facts,
        ['{"lastPrice":"100.25"}'],
        user_message="BTC fiyatı nedir?",
    )
    assert "Doğrulanmamış" in out
    assert "Unverified" not in out
