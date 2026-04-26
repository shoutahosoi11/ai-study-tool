import { createElement } from "react";

function IconBase(props, paths) {
  const {
    color = "currentColor",
    size = 24,
    strokeWidth = 2,
    children,
    ...rest
  } = props;

  return createElement(
    "svg",
    {
      xmlns: "http://www.w3.org/2000/svg",
      width: size,
      height: size,
      viewBox: "0 0 24 24",
      fill: "none",
      stroke: color,
      strokeWidth,
      strokeLinecap: "round",
      strokeLinejoin: "round",
      ...rest,
    },
    ...paths,
    children,
  );
}

export function Home(props) {
  return IconBase(props, [
    createElement("path", { key: "roof", d: "M3 10.5 12 3l9 7.5" }),
    createElement("path", { key: "house", d: "M5 9.5V21h14V9.5" }),
    createElement("path", { key: "door", d: "M10 21v-6h4v6" }),
  ]);
}

export function PencilLine(props) {
  return IconBase(props, [
    createElement("path", { key: "line", d: "M12 20h9" }),
    createElement("path", { key: "body", d: "m16.5 3.5 4 4L8 20l-5 1 1-5Z" }),
  ]);
}

export function User(props) {
  return IconBase(props, [
    createElement("path", { key: "head", d: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z" }),
    createElement("path", { key: "body", d: "M4 21a8 8 0 0 1 16 0" }),
  ]);
}
