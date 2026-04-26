import type { SVGProps } from "react";

export type LucideProps = SVGProps<SVGSVGElement> & {
  color?: string;
  size?: string | number;
  strokeWidth?: string | number;
};

export declare function Home(props: LucideProps): JSX.Element;
export declare function PencilLine(props: LucideProps): JSX.Element;
export declare function User(props: LucideProps): JSX.Element;
