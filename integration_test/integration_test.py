import pytest
import os
import subprocess
from pathlib import Path
from typing import Iterable, List

import dns.name
import dns.query
import dns.tsigkeyring
import dns.update
import dns.zone

UPDATER_BIN = Path(
    os.environ.get("UPDATER_BIN", str(Path(__file__).parents[1].joinpath("dnsupdater")))
)
DNS_SERVER = os.environ.get("TEST_DNS_SERVER", "127.0.0.1")
DNS_PORT = int(os.environ.get("TEST_DNS_PORT", "1053"))
DNS_TIMEOUT = 10
IGNORE_NAMES = ("@", "ns")
TEST_ZONE = "example.com"
TEST_ZONE2 = "example.net"
WANT_RECORDS = [
    "test 3600 IN A 192.0.2.1",
    "test 3600 IN AAAA 2001:db8::1",
    "test2 3600 IN A 192.0.2.2",
    "test2 3600 IN A 192.0.2.3",
    "test2 3600 IN AAAA 2001:db8::2",
    "test2 3600 IN AAAA 2001:db8::3",
    "test3 7200 IN CNAME test",
    'test4 3600 IN TXT "abcdef"',
    "test5 3600 IN MX 10 mx1.example.net.",
    "test5 3600 IN MX 15 mx2.example.net.",
    "test6 3600 IN SRV 10 20 80 www.example.net.",
]


@pytest.fixture(autouse=True, scope="session")
def set_krb5_config() -> None:
    os.environ["KRB5_CONFIG"] = str(
        Path(__file__).parent.joinpath("krb5.conf").absolute()
    )


@pytest.fixture(autouse=True, scope="function")
def clean_test_zone() -> None:
    clean_zone(TEST_ZONE)


def xfr(zone: str) -> dns.zone.Zone:
    return dns.zone.from_xfr(
        dns.query.xfr(DNS_SERVER, zone, port=DNS_PORT, timeout=DNS_TIMEOUT)
    )


def get_zone(zone: str) -> List[str]:
    z = xfr(zone)
    records: List[str] = []
    for n in z.nodes:
        if str(n) in IGNORE_NAMES:
            continue
        records.extend(z[n].to_text(n).splitlines())
    return sorted(records)


def rm_name(names: Iterable[dns.name.Name], zone: str, server: str, **kwargs) -> None:
    keyring = dns.tsigkeyring.from_text(
        {"admin-tsig-key.example.com": "bTueCg5wgjWkFsoX6n+p8WWUg5/tfyoBQEhnAjNx7RI="}
    )
    update = dns.update.Update(zone, keyring=keyring, keyalgorithm=dns.tsig.HMAC_SHA256)
    for n in names:
        update.delete(n)
    dns.query.tcp(update, server, **kwargs)


def clean_zone(zone: str) -> None:
    z = xfr(zone)
    rm_name(
        [n for n in z.nodes.keys() if str(n) not in IGNORE_NAMES],
        zone,
        DNS_SERVER,
        port=DNS_PORT,
        timeout=DNS_TIMEOUT,
    )


def test_insert() -> None:
    clean_zone(TEST_ZONE)
    config_file = Path(__file__).parent.joinpath("records.yml")
    assert (
        subprocess.call(
            [
                str(UPDATER_BIN.absolute()),
                "check",
                "--config",
                str(config_file.absolute()),
            ],
        )
        == 0
    )
    assert (
        subprocess.call(
            [
                str(UPDATER_BIN.absolute()),
                "insert",
                "--config",
                str(config_file.absolute()),
            ],
        )
        == 0
    )

    assert get_zone(TEST_ZONE) == WANT_RECORDS


@pytest.mark.parametrize("batch_size", range(1, len(WANT_RECORDS) + 2))
def test_insert_batch(batch_size: int) -> None:
    config_file = Path(__file__).parent.joinpath("records.yml")
    assert (
        subprocess.call(
            [
                str(UPDATER_BIN.absolute()),
                "insert",
                "--config",
                str(config_file.absolute()),
                f"--batch={batch_size}",
            ],
        )
        == 0
    )

    assert get_zone(TEST_ZONE) == WANT_RECORDS
