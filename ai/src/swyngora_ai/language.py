"""Reply-language helpers. Match the user's language (especially Turkish)."""

from __future__ import annotations

import re

DISCLAIMER_EN = (
    "Informational analysis only — not financial advice. "
    "Crypto markets are volatile; verify critical numbers via tools."
)
DISCLAIMER_TR = (
    "Yalnızca bilgilendirme amaçlıdır — yatırım tavsiyesi değildir. "
    "Kripto piyasaları oynaktır; kritik rakamları araçlarla doğrulayın."
)

EMPTY_REPLY_EN = (
    "I could not produce an answer. Try rephrasing or /ask with a clearer symbol (e.g. JUVUSDT)."
)
EMPTY_REPLY_TR = (
    "Yanıt üretemedim. Soruyu yeniden yazın veya daha net bir sembol kullanın (ör. JUVUSDT)."
)

LOW_CONF_EN = "[Low confidence: no tool-backed figures in this turn — verify via market tools.]"
LOW_CONF_TR = "[Düşük güven: bu turda araç destekli rakam yok — piyasa araçlarıyla doğrulayın.]"

UNVERIFIED_EN = "[Unverified figures not present in tool output: {shown}]"
UNVERIFIED_TR = "[Doğrulanmamış rakamlar (araç çıktısında yok): {shown}]"

# Turkish-specific letters are a strong signal even in short questions.
_TR_CHARS = re.compile(r"[çğıöşüÇĞİÖŞÜ]")
# Common function words / market phrasing (ascii-folded forms included).
_TR_WORDS = re.compile(
    r"\b("
    r"ve|bir|bu|şu|ne|nedir|nasıl|neden|niye|icin|için|"
    r"mi|mı|mu|mü|misin|mısın|musun|müsün|"
    r"fiyat|fiyati|fiyatı|analiz|analizi|"
    r"hakkinda|hakkında|bugun|bugün|yarin|yarın|"
    r"alayim|alayım|satayim|satayım|yukselir|yükselir|duser|düşer|"
    r"var|yok|lütfen|lutfen|merhaba|selam|"
    r"bana|benim|sence|sizce|acikla|açıkla|anlat|ozet|özet"
    r")\b",
    re.IGNORECASE,
)

_ADVICE_MARKERS = (
    "not financial advice",
    "yatırım tavsiyesi değildir",
    "yatirim tavsiyesi degildir",
)


def detect_reply_lang(text: str) -> str:
    """Return ``tr`` when the user wrote Turkish, else ``en``.

    Other languages still rely on the model matching the question; we only
    special-case Turkish for appended disclaimers / grounding lines.
    """
    raw = (text or "").strip()
    if not raw:
        return "en"
    if _TR_CHARS.search(raw) or _TR_WORDS.search(raw):
        return "tr"
    return "en"


def disclaimer_for(text: str) -> str:
    return DISCLAIMER_TR if detect_reply_lang(text) == "tr" else DISCLAIMER_EN


def empty_reply_for(text: str) -> str:
    return EMPTY_REPLY_TR if detect_reply_lang(text) == "tr" else EMPTY_REPLY_EN


def low_confidence_note(lang: str) -> str:
    return LOW_CONF_TR if lang == "tr" else LOW_CONF_EN


def unverified_note(lang: str, shown: str) -> str:
    tmpl = UNVERIFIED_TR if lang == "tr" else UNVERIFIED_EN
    return tmpl.format(shown=shown)


def has_advice_disclaimer(reply: str) -> bool:
    low = (reply or "").lower()
    return any(m in low for m in _ADVICE_MARKERS)
