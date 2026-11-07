// Member 07: renders the API's insertion order, without sorting by transaction date.
const recentInstances = new WeakMap();

function recentFormatMoney(cents) {
  const value = BigInt(cents);
  return (value / 100n).toString() + "." + (value % 100n).toString().padStart(2, "0");
}

function recentValidateRecords(records) {
  if (!Array.isArray(records)) throw new Error("记录接口未返回约定的数组。");
  for (const record of records) {
    if (!record || typeof record.date !== "string" || !["income", "expense"].includes(record.type) ||
        !Number.isSafeInteger(record.amountCents) || record.amountCents <= 0 ||
        typeof record.category !== "string" || (record.note !== undefined && typeof record.note !== "string")) {
      throw new Error("记录格式或金额无效，无法安全显示。");
    }
  }
  return records;
}

export function initRecent(container) {
  if (!container || typeof container.replaceChildren !== "function") throw new TypeError("最近记录容器无效");
  if (recentInstances.has(container)) return recentInstances.get(container);
  const wrapper = document.createElement("div");
  wrapper.className = "recent-view";
  wrapper.innerHTML = `
    <div class="recent-toolbar">
      <label class="recent-limit-label">显示条数
        <select class="recent-limit" aria-label="最近记录条数"><option value="5">5 笔</option><option value="10" selected>10 笔</option><option value="20">20 笔</option></select>
      </label>
      <button type="button" class="recent-refresh">刷新记录</button>
    </div>
    <p class="recent-feedback" role="status" aria-live="polite"></p>
    <div class="recent-content"></div>`;
  container.replaceChildren(wrapper);
  const limit = wrapper.querySelector(".recent-limit");
  const refreshButton = wrapper.querySelector(".recent-refresh");
  const feedback = wrapper.querySelector(".recent-feedback");
  const content = wrapper.querySelector(".recent-content");
  let sequence = 0;
  let activeRequest = null;
  let destroyed = false;
  let hasResult = false;

  function render(records) {
    content.replaceChildren();
    if (!records.length) {
      const empty = document.createElement("p");
      empty.className = "recent-empty";
      empty.textContent = "还没有记录，先记下第一笔收支吧。";
      content.append(empty);
      return;
    }
    const scroll = document.createElement("div");
    scroll.className = "recent-table-scroll";
    scroll.tabIndex = 0;
    scroll.setAttribute("role", "region");
    scroll.setAttribute("aria-label", "最近账目，窄屏可横向滚动");
    const table = document.createElement("table");
    table.className = "recent-table";
    const caption = table.createCaption();
    caption.textContent = "按录入顺序倒序显示";
    const head = table.createTHead().insertRow();
    for (const label of ["日期", "类型", "金额（元）", "类别", "备注"]) {
      const th = document.createElement("th");
      th.scope = "col";
      th.textContent = label;
      head.append(th);
    }
    const body = table.createTBody();
    for (const record of records) {
      const row = body.insertRow();
      row.className = "recent-row recent-" + record.type;
      const values = [record.date, record.type === "income" ? "收入" : "支出",
        (record.type === "income" ? "+" : "−") + recentFormatMoney(record.amountCents),
        record.category, record.note || "—"];
      values.forEach((value, index) => {
        const cell = row.insertCell();
        cell.className = ["recent-date", "recent-type", "recent-amount", "recent-category", "recent-note"][index];
        cell.textContent = value;
      });
    }
    scroll.append(table);
    content.append(scroll);
  }
  async function load() {
    if (destroyed) return;
    const current = ++sequence;
    activeRequest?.abort();
    activeRequest = new AbortController();
    refreshButton.disabled = true;
    wrapper.setAttribute("aria-busy", "true");
    feedback.textContent = "正在读取最近记录…";
    feedback.dataset.state = "loading";
    try {
      const response = await fetch("/api/records/recent?limit=" + encodeURIComponent(limit.value),
        {signal: activeRequest.signal});
      let result;
      try { result = await response.json(); }
      catch { throw new Error("服务返回了无法识别的响应。"); }
      if (!response.ok) throw new Error(typeof result?.error === "string" ? result.error : "读取账本失败。");
      const records = recentValidateRecords(result);
      if (destroyed || current !== sequence) return;
      render(records);
      hasResult = true;
      feedback.textContent = records.length ? "已更新，显示 " + records.length + " 笔记录。" : "账本为空。";
      feedback.dataset.state = "success";
    } catch (error) {
      if (destroyed || current !== sequence || error.name === "AbortError") return;
      feedback.textContent = (error instanceof TypeError ? "网络连接失败。" : error.message) +
        (hasResult ? " 当前显示上次读取的结果。" : "") + " 请点击刷新重试。";
      feedback.dataset.state = "error";
    } finally {
      if (!destroyed && current === sequence) {
        refreshButton.disabled = false;
        wrapper.removeAttribute("aria-busy");
      }
    }
  }
  const onChanged = () => { void load(); };
  refreshButton.addEventListener("click", onChanged);
  limit.addEventListener("change", onChanged);
  window.addEventListener("ledger:changed", onChanged);
  const controller = {
    refresh: load,
    destroy() {
      destroyed = true;
      ++sequence;
      activeRequest?.abort();
      window.removeEventListener("ledger:changed", onChanged);
      wrapper.remove();
      recentInstances.delete(container);
    },
  };
  recentInstances.set(container, controller);
  void load();
  return controller;
}

