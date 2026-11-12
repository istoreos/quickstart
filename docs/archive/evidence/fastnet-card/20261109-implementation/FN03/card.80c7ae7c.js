import { defineComponent as n, onMounted as o, onUnmounted as a, h as e } from "/luci-static/quickstart/card-runtime-v1.js";
const m = 1;
let r = 0, d = 0;
const f = n({ props: ["context"], setup(t) {
  return o(() => r++), a(() => d++), () => e("article", { "data-external-fixture": "", style: "padding:20px" }, [
    e("h2", "独立 FastNet 卡片构建样例"),
    e("p", { "data-card-theme": "" }, t.context.theme.preference + "/" + t.context.theme.effective),
    e("p", { "data-card-form": "" }, t.context.form)
  ]);
} });
export {
  m as apiVersion,
  f as default,
  r as mounts,
  d as unmounts
};
