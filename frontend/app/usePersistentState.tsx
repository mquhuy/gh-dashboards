import { useState, useEffect } from 'react';

function usePersistentState(key, initialValue) {
  const [state, setState] = useState(() => {
    if (typeof window != 'undefined') {
      const storedState = localStorage.getItem(key);
      return storedState ? JSON.parse(storedState) : initialValue;
    }
    return initialValue;
  });

  useEffect(() => {
    if (typeof window != 'undefined') {
      localStorage.setItem(key, JSON.stringify(state));
    }
  }, [key, state]);

  return [state, setState];
}

export default usePersistentState;
