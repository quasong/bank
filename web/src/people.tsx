import { FormEvent, useState } from "react";
import { createPayee, deletePayee, errorMessage, updatePayee, type Payee } from "./api";
import { digitsOnly, formatAccountNumber, maskAccountInput, payeeLabel } from "./format";
import { usePayees } from "./hooks";
import { Banner, Sheet } from "./ui";

function nickname(p: Payee): string {
  return p.display_name === p.account_number ? "" : p.display_name;
}

export function SavePersonSheet({
  ownAccountNumber,
  initialNumber = "",
  initialName = "",
  onClose,
  onSaved,
}: {
  ownAccountNumber?: string;
  initialNumber?: string;
  initialName?: string;
  onClose: () => void;
  onSaved: (payee: Payee) => void;
}) {
  const [toNumber, setToNumber] = useState(digitsOnly(initialNumber));
  const [name, setName] = useState(initialName.slice(0, 40));
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const ownAccount = Boolean(ownAccountNumber) && toNumber === ownAccountNumber;
  const ready = toNumber.length === 8 && !ownAccount;

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!ready) {
      setError(ownAccount ? "That's your own account" : "Destination is an 8-digit account number");
      return;
    }
    setError("");
    setPending(true);
    try {
      const res = await createPayee(toNumber, name);
      onSaved(res.payee);
    } catch (err) {
      setError(errorMessage(err, "Could not save this person"));
    } finally {
      setPending(false);
    }
  }

  return (
    <Sheet title="Save a person" onClose={onClose}>
      <form className="stack" onSubmit={onSubmit}>
        {error ? <Banner>{error}</Banner> : null}
        <label>
          Account
          <input
            className={`digits${ownAccount ? " input-warn" : toNumber.length === 8 ? " input-ok" : ""}`}
            value={maskAccountInput(toNumber)}
            onChange={(e) => setToNumber(digitsOnly(e.target.value))}
            inputMode="numeric"
            maxLength={11}
            placeholder="0000 · 0000"
            autoComplete="off"
            autoFocus
            aria-label="Account number to save"
          />
        </label>
        <label>
          Name
          <input
            value={name}
            onChange={(e) => setName(e.target.value.slice(0, 40))}
            maxLength={40}
            placeholder="Optional"
            autoComplete="off"
            aria-label="Payee name"
          />
        </label>
        <p className={`avail${ownAccount ? " avail-warn" : ""}`}>
          {ownAccount ? "That's your own account" : `${toNumber.length}/8 digits`}
        </p>
        <button className="btn btn-primary btn-block" type="submit" disabled={pending || !ready}>
          {pending ? "Saving…" : "Save"}
        </button>
      </form>
    </Sheet>
  );
}

export function PeoplePanel({
  ownAccountNumber,
  onToast,
  onSendTo,
}: {
  ownAccountNumber?: string;
  onToast: (text: string) => void;
  onSendTo?: (accountNumber: string, displayName: string) => void;
}) {
  const { payees, reload } = usePayees();
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<Payee | null>(null);
  const [removing, setRemoving] = useState<Payee | null>(null);
  const [name, setName] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const saved = (payees ?? []).filter((p) => p.account_number !== ownAccountNumber);

  async function onRename(e: FormEvent) {
    e.preventDefault();
    if (!editing) return;
    setError("");
    setPending(true);
    try {
      await updatePayee(editing.id, name);
      await reload();
      onToast("Name updated");
      setEditing(null);
    } catch (err) {
      setError(errorMessage(err, "Could not update name"));
    } finally {
      setPending(false);
    }
  }

  async function onRemove() {
    if (!removing) return;
    setError("");
    setPending(true);
    try {
      await deletePayee(removing.id);
      await reload();
      onToast("Removed");
      setRemoving(null);
      setEditing(null);
    } catch (err) {
      setError(errorMessage(err, "Could not remove this person"));
    } finally {
      setPending(false);
    }
  }

  if (payees == null) {
    return null;
  }

  return (
    <>
      <section className="panel facts">
        <div className="panel-h">
          <h2>Saved people</h2>
          <button type="button" className="text-link" onClick={() => setAdding(true)}>
            Add
          </button>
        </div>
        {saved.length === 0 ? (
          <p className="panel-empty">Save someone to send faster next time.</p>
        ) : (
          saved.map((p) => (
            <button
              key={p.id}
              type="button"
              className="fact"
              onClick={() => {
                setError("");
                setName(nickname(p));
                setEditing(p);
              }}
            >
              <span>{formatAccountNumber(p.account_number)}</span>
              <strong>{nickname(p) || "Saved"}</strong>
            </button>
          ))
        )}
      </section>

      {adding ? (
        <SavePersonSheet
          ownAccountNumber={ownAccountNumber}
          onClose={() => setAdding(false)}
          onSaved={async () => {
            await reload();
            setAdding(false);
            onToast("Saved");
          }}
        />
      ) : null}

      {editing && !removing ? (
        <Sheet title={payeeLabel(editing.account_number, editing.display_name)} onClose={() => setEditing(null)}>
          <form className="stack" onSubmit={onRename}>
            {error ? <Banner>{error}</Banner> : null}
            <p className="sheet-copy">{formatAccountNumber(editing.account_number)}</p>
            <label>
              Name
              <input
                value={name}
                onChange={(e) => setName(e.target.value.slice(0, 40))}
                maxLength={40}
                placeholder="Optional"
                autoComplete="off"
                autoFocus
                aria-label="Payee name"
              />
            </label>
            <button className="btn btn-primary btn-block" type="submit" disabled={pending}>
              {pending ? "Saving…" : "Save name"}
            </button>
            {onSendTo ? (
              <button
                className="btn btn-secondary btn-block"
                type="button"
                disabled={pending}
                onClick={() => onSendTo(editing.account_number, nickname(editing))}
              >
                Send to this person
              </button>
            ) : null}
            <button
              className="btn btn-quiet btn-block"
              type="button"
              disabled={pending}
              onClick={() => {
                setError("");
                setRemoving(editing);
              }}
            >
              Remove
            </button>
          </form>
        </Sheet>
      ) : null}

      {removing ? (
        <Sheet title="Remove this person?" onClose={() => setRemoving(null)}>
          <p className="sheet-copy">
            {payeeLabel(removing.account_number, removing.display_name)} will leave your saved list. Past payments stay as they are.
          </p>
          {error ? <Banner>{error}</Banner> : null}
          <div className="sheet-actions">
            <button className="btn btn-secondary" type="button" disabled={pending} onClick={() => setRemoving(null)}>
              Keep
            </button>
            <button className="btn btn-danger" type="button" disabled={pending} onClick={() => void onRemove()}>
              {pending ? "Removing…" : "Remove"}
            </button>
          </div>
        </Sheet>
      ) : null}
    </>
  );
}
