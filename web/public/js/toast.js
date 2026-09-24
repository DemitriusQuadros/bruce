/* Toast utility */

export function showToast(msg, type = 'success') {
    const container = document.getElementById('toast-container');
    if (!container) return;
    const toast = document.createElement('div');
    toast.className = `toast toast--${type} toast--visible`;
    toast.setAttribute('role', 'status');
    toast.setAttribute('data-testid', 'toast');
    toast.textContent = msg;
    container.appendChild(toast);

    setTimeout(() => {
        toast.classList.remove('toast--visible');
        toast.remove();
    }, 3000);
}
