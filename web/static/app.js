// Pocketa App Client JS

function showToast(message, type = 'success') {
  let container = document.getElementById('toast-container');
  if (!container) {
    container = document.createElement('div');
    container.id = 'toast-container';
    container.className = 'toast-container';
    document.body.appendChild(container);
  }

  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.innerText = message;

  container.appendChild(toast);

  setTimeout(() => {
    toast.remove();
  }, 3500);
}

function formatPaise(paise) {
  const isNegative = paise < 0;
  if (isNegative) paise = -paise;
  const rupees = Math.floor(paise / 100);
  const remPaise = (paise % 100).toString().padStart(2, '0');

  let rStr = rupees.toString();
  let formattedRupees = '';
  if (rStr.length <= 3) {
    formattedRupees = rStr;
  } else {
    const last3 = rStr.substring(rStr.length - 3);
    let rest = rStr.substring(0, rStr.length - 3);
    const parts = [];
    while (rest.length > 2) {
      parts.unshift(rest.substring(rest.length - 2));
      rest = rest.substring(0, rest.length - 2);
    }
    if (rest.length > 0) parts.unshift(rest);
    formattedRupees = parts.join(',') + ',' + last3;
  }

  return (isNegative ? '-' : '') + '₹' + formattedRupees + '.' + remPaise;
}

function toggleBalance() {
  const el = document.getElementById('balance-display');
  const eye = document.getElementById('eye-icon');
  if (!el) return;

  if (el.getAttribute('data-hidden') === 'true') {
    el.innerText = el.getAttribute('data-real');
    el.setAttribute('data-hidden', 'false');
    if (eye) eye.innerText = '👁️';
  } else {
    el.setAttribute('data-real', el.innerText);
    el.innerText = '••••••••';
    el.setAttribute('data-hidden', 'true');
    if (eye) eye.innerText = '🙈';
  }
}

async function handleLogout() {
  try {
    const res = await fetch('/api/v1/auth/logout', { method: 'POST' });
    if (res.ok) {
      window.location.href = '/login';
    }
  } catch (err) {
    window.location.href = '/login';
  }
}

// Live Handle Check on Signup
let handleTimer = null;
function checkHandleAvailability(input) {
  clearTimeout(handleTimer);
  const statusEl = document.getElementById('handle-status');
  if (!statusEl) return;

  let val = input.value.trim();
  if (!val.startsWith('@')) val = '@' + val;
  input.value = val.toLowerCase();

  if (val.length < 3) {
    statusEl.innerText = '';
    return;
  }

  handleTimer = setTimeout(async () => {
    try {
      const res = await fetch(`/api/v1/users/lookup?handle=${encodeURIComponent(val)}`);
      if (res.status === 404) {
        statusEl.innerText = '✓ Handle available';
        statusEl.style.color = 'var(--primary)';
      } else {
        statusEl.innerText = '✕ Handle taken';
        statusEl.style.color = 'var(--danger)';
      }
    } catch (e) {
      statusEl.innerText = '';
    }
  }, 400);
}
