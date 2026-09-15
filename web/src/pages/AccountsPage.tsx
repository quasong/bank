import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { closeAccount, errorMessage, freezeAccount, openAccount, unfreezeAccount } from "../api";
import { CURRENCIES, auditAmount, auditLabel, copyText, formatAccountNumber, openedLabel, recentWhen } from "../format";
import { useAccounts, useAudit, useSelectedAccount, useToast } from "../hooks";
import { MoneySheet } from "../moneyflow";
import { PeoplePanel } from "../people";
import { AccountHero, Banner, CurrencyChoices, EmptyState, IconArrow, IconMinus, IconPlus, IconSwap, Page, PageSkeleton, Sheet, Toast, Wallets } from "../ui";

type MoneyKind = "fund" | "withdraw";

export function AccountsPage() {
  const { accounts, error, setError, reload } = useAccounts();
  const { selected, select } = useSelectedAccount(accounts);
  const { events, reload: reloadAudit } = useAudit();
  const { text: toast, show } = useToast();
  const [pending, setPending] = useState(false);
  const [form, setForm] = useState<MoneyKind | null>(null);
  const [closing, setClosing] = useState(false);
  const [freezing, setFreezing] = useState(false);
  const [adding, setAdding] = useState(false);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const missing = CURRENCIES.filter((c) => !(accounts ?? []).some((a) => a.currency === c));
  const ownNumbers = (accounts ?? []).map((a) => a.account_number);

  useEffect(() => {
    const action = searchParams.get("action");
    if (action === "fund" || action === "withdraw") {
      setForm(action);
      setSearchParams({}, { replace: true });
    }
  }, [searchParams, setSearchParams]);

  async function onOpen(currency?: string) {
    setError("");
    setPending(true);
    try {
      const res = await openAccount(currency);
      await reload();
      select(res.account.id);
      show(`${res.account.currency} opened`);
      setAdding(false);
      await reloadAudit().catch(() => undefined);
    } catch (err) {
      setError(errorMessage(err, "Could not open account"));
    } finally {
      setPending(false);
    }
  }

  async function onMoneySuccess(message: string) {
    setForm(null);
    show(message);
    await reload();
    await reloadAudit().catch(() => undefined);
  }

  async function runStatus(action: () => Promise<unknown>, message: string) {
    setError("");
    setPending(true);
    try {
      await action();
      setClosing(false);
      setFreezing(false);
      show(message);
      await reload();
      await reloadAudit().catch(() => undefined);
    } catch (err) {
      setError(errorMessage(err, "Could not update account"));
    } finally {
      setPending(false);
    }
  }

  if (!accounts) {
    return <PageSkeleton />;
  }

  const acct = selected;

  return (
    <Page title="Account" kicker={acct?.currency ?? "Balances"}>
      {error ? <Banner>{error}</Banner> : null}
      {!acct ? (
        <EmptyState
          title="No account yet"
          body="Open a USD account to hold a balance. You can add euros and pounds next."
          action={
            <button className="btn btn-primary" type="button" disabled={pending} onClick={() => void onOpen()}>
              {pending ? "Opening…" : "Open account"}
            </button>
          }
        />
      ) : (
        <>
          <Wallets
            accounts={accounts}
            selectedId={acct.id}
            onSelect={select}
            onAdd={missing.length > 0 ? () => setAdding(true) : undefined}
          />
          <AccountHero account={acct} onCopied={() => show("Copied")} />
          {acct.status === "active" ? (
            <div className="quicks">
              <button className="quick" type="button" onClick={() => setForm("fund")}>
                <span className="quick-icon">
                  <IconPlus />
                </span>
                Add
              </button>
              <button className="quick" type="button" onClick={() => setForm("withdraw")}>
                <span className="quick-icon">
                  <IconMinus />
                </span>
                Withdraw
              </button>
              <button className="quick" type="button" onClick={() => navigate("/transfers")}>
                <span className="quick-icon">
                  <IconArrow />
                </span>
                Send
              </button>
              <button className="quick" type="button" onClick={() => navigate("/convert")}>
                <span className="quick-icon">
                  <IconSwap />
                </span>
                Convert
              </button>
            </div>
          ) : null}
          {acct.status === "frozen" ? (
            <div className="manage">
              <button
                className="btn btn-primary"
                type="button"
                disabled={pending}
                onClick={() => runStatus(() => unfreezeAccount(acct.id), "Account unfrozen")}
              >
                Unfreeze
              </button>
              <button className="btn btn-danger" type="button" disabled={pending} onClick={() => setClosing(true)}>
                Close account
              </button>
            </div>
          ) : null}
          {acct.status === "closed" ? <p className="kicker">This account is closed and cannot move money.</p> : null}
          <section className="panel facts">
            <button
              type="button"
              className="fact"
              onClick={() => {
                void copyText(acct.account_number);
                show("Copied account number");
              }}
            >
              <span>{acct.currency === "EUR" ? "IBAN" : "Account number"}</span>
              <strong>{formatAccountNumber(acct.account_number)}</strong>
            </button>
            {acct.details?.routing_number ? (
              <button
                type="button"
                className="fact"
                onClick={() => {
                  void copyText(acct.details?.routing_number ?? "");
                  show("Copied routing number");
                }}
              >
                <span>ACH routing</span>
                <strong>{acct.details.routing_number}</strong>
              </button>
            ) : null}
            {acct.details?.sort_code ? (
              <button
                type="button"
                className="fact"
                onClick={() => {
                  void copyText(acct.details?.sort_code ?? "");
                  show("Copied sort code");
                }}
              >
                <span>Sort code</span>
                <strong>{acct.details.sort_code}</strong>
              </button>
            ) : null}
            {acct.details?.account && acct.currency !== "USD" ? (
              <button
                type="button"
                className="fact"
                onClick={() => {
                  void copyText(acct.details?.account ?? "");
                  show("Copied account");
                }}
              >
                <span>Local account</span>
                <strong>{acct.details.account}</strong>
              </button>
            ) : null}
            {acct.details?.bic ? (
              <button
                type="button"
                className="fact"
                onClick={() => {
                  void copyText(acct.details?.bic ?? "");
                  show("Copied BIC");
                }}
              >
                <span>BIC</span>
                <strong>{acct.details.bic}</strong>
              </button>
            ) : null}
            <div className="fact">
              <span>Opened</span>
              <strong>{openedLabel(acct.opened_at)}</strong>
            </div>
          </section>
          <PeoplePanel
            ownAccountNumbers={ownNumbers}
            currency={acct.currency}
            onToast={show}
            onSendTo={(number) => navigate(`/transfers?to=${number}`)}
          />
          {events && events.length > 0 ? (
            <section className="panel facts">
              <p className="day-label">Account log</p>
              {events.slice(0, 12).map((ev) => {
                const amount = auditAmount(ev.metadata);
                return (
                  <div className="fact" key={ev.id}>
                    <span>
                      {auditLabel(ev.action)}
                      {ev.metadata?.note ? ` · ${ev.metadata.note}` : ""}
                    </span>
                    <strong>
                      {amount ? `${amount} · ` : ""}
                      {recentWhen(ev.created_at)}
                    </strong>
                  </div>
                );
              })}
            </section>
          ) : null}
          {acct.status === "active" ? (
            <div className="manage">
              <button className="btn btn-quiet" type="button" disabled={pending} onClick={() => setFreezing(true)}>
                Freeze account
              </button>
              <button className="btn btn-quiet" type="button" disabled={pending} onClick={() => setClosing(true)}>
                Close account
              </button>
            </div>
          ) : null}
        </>
      )}

      {form && acct ? (
        <MoneySheet kind={form} account={acct} onClose={() => setForm(null)} onSuccess={onMoneySuccess} />
      ) : null}

      {adding ? (
        <Sheet title="Add a currency" onClose={() => setAdding(false)}>
          <p className="sheet-copy">Each currency gets its own local details, like Wise or Revolut.</p>
          <CurrencyChoices currencies={missing} pending={pending} onPick={(ccy) => void onOpen(ccy)} />
        </Sheet>
      ) : null}

      {freezing && acct ? (
        <Sheet title="Freeze this account?" onClose={() => setFreezing(false)}>
          <p className="sheet-copy">You won't be able to add, withdraw, send, or convert until you unfreeze it.</p>
          <div className="sheet-actions">
            <button className="btn btn-secondary" type="button" onClick={() => setFreezing(false)}>
              Keep it active
            </button>
            <button
              className="btn btn-primary"
              type="button"
              disabled={pending}
              onClick={() => runStatus(() => freezeAccount(acct.id), "Account frozen")}
            >
              Freeze
            </button>
          </div>
        </Sheet>
      ) : null}

      {closing && acct ? (
        <Sheet title="Close this account?" onClose={() => setClosing(false)}>
          <p className="sheet-copy">You can only close it when the balance is zero. This cannot be undone.</p>
          <div className="sheet-actions">
            <button className="btn btn-secondary" type="button" onClick={() => setClosing(false)}>
              Keep it
            </button>
            <button
              className="btn btn-danger"
              type="button"
              disabled={pending}
              onClick={() => runStatus(() => closeAccount(acct.id), "Account closed")}
            >
              Close account
            </button>
          </div>
        </Sheet>
      ) : null}
      <Toast text={toast} />
    </Page>
  );
}
