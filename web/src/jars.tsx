import { FormEvent, useState } from "react";
import { closeAccount, errorMessage, moveMoney, openAccount, renameAccount, type BankAccount } from "./api";
import { formatMoney } from "./format";
import { jarName } from "./hooks";
import { centsToDollars, dollarsToCents } from "./money";
import { AmountChips, AmountField, Banner, MoneyText, Sheet } from "./ui";

const PRESETS = [1000, 2000, 5000, 10000];

export function JarsPanel({
  spend,
  jars,
  canMove,
  onAdd,
  onOpen,
}: {
  spend: BankAccount;
  jars: BankAccount[];
  canMove: boolean;
  onAdd?: () => void;
  onOpen: (jar: BankAccount) => void;
}) {
  const parked = jars.reduce((sum, j) => sum + j.balance_cents, 0);
  return (
    <section className="panel jars">
      <div className="panel-h">
        <h2>Jars</h2>
        {canMove && onAdd ? (
          <button type="button" className="text-link" onClick={onAdd}>
            Add
          </button>
        ) : null}
      </div>
      {jars.length === 0 ? (
        <p className="panel-empty">Set {spend.currency} aside for rent, a trip, or anything else. It stays in this currency.</p>
      ) : (
        <>
          {parked > 0 ? <p className="jars-sum">{formatMoney(parked, spend.currency)} in jars</p> : null}
          {jars.map((jar) => (
            <button key={jar.id} type="button" className="jar" onClick={() => onOpen(jar)}>
              <span>
                <strong>{jarName(jar)}</strong>
                <em>{jar.status === "active" ? "Inside this currency" : jar.status}</em>
              </span>
              <MoneyText cents={jar.balance_cents} currency={jar.currency} />
            </button>
          ))}
        </>
      )}
    </section>
  );
}

export function AddJarSheet({
  spend,
  pending,
  onClose,
  onCreate,
}: {
  spend: BankAccount;
  pending?: boolean;
  onClose: () => void;
  onCreate: (label: string) => void;
}) {
  const [label, setLabel] = useState("");
  return (
    <Sheet title="New jar" onClose={onClose}>
      <form
        className="stack"
        onSubmit={(e) => {
          e.preventDefault();
          onCreate(label.trim());
        }}
      >
        <p className="sheet-copy">Jars hold {spend.currency} only. Move money in from this balance; they cannot send or convert.</p>
        <label>
          Name
          <input value={label} onChange={(e) => setLabel(e.target.value.slice(0, 20))} placeholder="Rent, travel…" autoComplete="off" autoFocus />
        </label>
        <button className="btn btn-primary btn-block" type="submit" disabled={pending}>
          {pending ? "Opening…" : "Create jar"}
        </button>
      </form>
    </Sheet>
  );
}

export function MoveSheet({
  from,
  to,
  title,
  onClose,
  onSuccess,
}: {
  from: BankAccount;
  to: BankAccount;
  title: string;
  onClose: () => void;
  onSuccess: (message: string) => Promise<void> | void;
}) {
  const [amount, setAmount] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const cents = dollarsToCents(amount);
  const tooMuch = cents != null && cents > from.balance_cents;
  const ready = cents != null && cents > 0 && !tooMuch;
  const chips = PRESETS.filter((v) => v <= from.balance_cents);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (cents == null || cents <= 0) {
      setError("Enter an amount with at most two decimals");
      return;
    }
    if (tooMuch) {
      setError("You don't have that much available");
      return;
    }
    setError("");
    setPending(true);
    try {
      await moveMoney(from.id, to.id, cents, crypto.randomUUID());
      await onSuccess(`Moved ${formatMoney(cents, from.currency)}`);
    } catch (err) {
      setError(errorMessage(err, "Could not move that"));
    } finally {
      setPending(false);
    }
  }

  return (
    <Sheet title={title} onClose={onClose}>
      <form className="stack" onSubmit={onSubmit}>
        {error ? <Banner>{error}</Banner> : null}
        <p className="sheet-copy">
          From {from.product === "jar" ? jarName(from) : from.currency} to {to.product === "jar" ? jarName(to) : to.currency}.
        </p>
        <AmountField value={amount} onChange={setAmount} autoFocus currency={from.currency} />
        <AmountChips values={chips} onPick={(v) => setAmount(centsToDollars(v))} disabled={pending} currency={from.currency} />
        <p className="avail">
          Available {formatMoney(from.balance_cents, from.currency)}
          {from.balance_cents > 0 ? (
            <>
              {" · "}
              <button type="button" className="text-link" onClick={() => setAmount(centsToDollars(from.balance_cents))}>
                Max
              </button>
            </>
          ) : null}
        </p>
        <button className="btn btn-primary btn-block" type="submit" disabled={pending || !ready}>
          {pending ? "Moving…" : cents != null && ready ? `Move ${formatMoney(cents, from.currency)}` : "Move"}
        </button>
      </form>
    </Sheet>
  );
}

export function JarSheet({
  spend,
  jar,
  pending,
  onClose,
  onReload,
  onToast,
  onError,
}: {
  spend: BankAccount;
  jar: BankAccount;
  pending?: boolean;
  onClose: () => void;
  onReload: () => Promise<unknown> | unknown;
  onToast: (text: string) => void;
  onError: (text: string) => void;
}) {
  const [label, setLabel] = useState(jar.label ?? "");
  const [saving, setSaving] = useState(false);
  const [move, setMove] = useState<"in" | "out" | null>(null);
  const canMove = spend.status === "active" && jar.status === "active";

  async function saveName() {
    const next = label.trim();
    if (next === (jar.label ?? "").trim()) return;
    setSaving(true);
    try {
      await renameAccount(jar.id, next);
      onToast("Jar renamed");
      await onReload();
    } catch (err) {
      onError(errorMessage(err, "Could not rename"));
    } finally {
      setSaving(false);
    }
  }

  async function onCloseJar() {
    try {
      await closeAccount(jar.id);
      onToast("Jar closed");
      await onReload();
      onClose();
    } catch (err) {
      onError(errorMessage(err, "Could not close jar"));
    }
  }

  if (move === "in") {
    return (
      <MoveSheet
        from={spend}
        to={jar}
        title={`Add to ${jarName(jar)}`}
        onClose={() => setMove(null)}
        onSuccess={async (message) => {
          await onReload();
          onToast(message);
          setMove(null);
        }}
      />
    );
  }
  if (move === "out") {
    return (
      <MoveSheet
        from={jar}
        to={spend}
        title={`Move to ${spend.currency}`}
        onClose={() => setMove(null)}
        onSuccess={async (message) => {
          await onReload();
          onToast(message);
          setMove(null);
        }}
      />
    );
  }

  return (
    <Sheet title={jarName(jar)} onClose={onClose}>
      <p className="detail-amt">
        <MoneyText cents={jar.balance_cents} currency={jar.currency} reveal />
      </p>
      {canMove ? (
        <div className="sheet-actions">
          <button className="btn btn-secondary" type="button" disabled={!spend.balance_cents} onClick={() => setMove("in")}>
            Move in
          </button>
          <button className="btn btn-primary" type="button" disabled={!jar.balance_cents} onClick={() => setMove("out")}>
            Move out
          </button>
        </div>
      ) : (
        <p className="sheet-copy">Unfreeze both balances to move money.</p>
      )}
      <label>
        Name
        <input
          value={label}
          onChange={(e) => setLabel(e.target.value.slice(0, 20))}
          onBlur={() => void saveName()}
          disabled={saving || jar.status === "closed"}
        />
      </label>
      {jar.balance_cents === 0 && jar.status === "active" ? (
        <button className="btn btn-quiet btn-block" type="button" disabled={pending} onClick={() => void onCloseJar()}>
          Close jar
        </button>
      ) : null}
    </Sheet>
  );
}

export async function createJar(currency: string, label: string) {
  return openAccount(currency, { product: "jar", label });
}
