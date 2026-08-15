from swyngora_ai.config import Settings
from swyngora_ai.tools.market_http import build_market_tools
from swyngora_ai.tools.packs import ACCOUNT_TOOLS, BOOK_TOOLS, PAPER_TOOLS, TAPE_TOOLS


def test_pack_filter_splits_tools():
    settings = Settings(_env_file=None, api_base_url="http://test")
    tape = {t.name for t in build_market_tools(settings, pack="tape")}
    book = {t.name for t in build_market_tools(settings, pack="book")}
    paper = {t.name for t in build_market_tools(settings, pack="paper")}
    account = {t.name for t in build_market_tools(settings, pack="account")}
    assert tape == set(TAPE_TOOLS)
    assert book == set(BOOK_TOOLS)
    assert "place_portfolio_order" not in tape
    assert "get_ticker" not in paper
    assert "preview_import" in account
    assert "place_portfolio_oco_order" in paper
    assert "list_delist_schedule" in tape
    assert "amend_portfolio_order" in paper
    assert paper == set(PAPER_TOOLS)
    assert account == set(ACCOUNT_TOOLS)
    all_names = {t.name for t in build_market_tools(settings)}
    assert "get_portfolio_order" in all_names
    assert "cancel_all_portfolio_orders" in all_names
    assert "place_portfolio_bracket_order" in all_names
