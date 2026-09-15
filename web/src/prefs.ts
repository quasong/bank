import { useCallback, useEffect, useState } from "react";

const HIDE_KEY = "tb.hide-balance";
const EMAIL_KEY = "tb.last-email";
const PREFS_EVENT = "tb-prefs";

function readHide(): boolean {
  try {
    return localStorage.getItem(HIDE_KEY) === "1";
  } catch {
    return false;
  }
}

export function useHideBalance() {
  const [hidden, setHidden] = useState(readHide);

  useEffect(() => {
    function sync() {
      setHidden(readHide());
    }
    window.addEventListener("storage", sync);
    window.addEventListener(PREFS_EVENT, sync);
    return () => {
      window.removeEventListener("storage", sync);
      window.removeEventListener(PREFS_EVENT, sync);
    };
  }, []);

  const toggle = useCallback(() => {
    const next = !readHide();
    try {
      localStorage.setItem(HIDE_KEY, next ? "1" : "0");
    } catch {
      /* ignore quota */
    }
    window.dispatchEvent(new Event(PREFS_EVENT));
  }, []);

  return { hidden, toggle };
}

export function readLastEmail(): string {
  try {
    return localStorage.getItem(EMAIL_KEY) ?? "";
  } catch {
    return "";
  }
}

export function rememberEmail(email: string) {
  try {
    localStorage.setItem(EMAIL_KEY, email.trim());
  } catch {
    /* ignore quota */
  }
}
