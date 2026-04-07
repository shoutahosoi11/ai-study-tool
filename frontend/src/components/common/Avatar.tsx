import { theme } from "../../theme";

type AvatarProps = {
  src?: string;
  name?: string;
  size?: number;
};

export function Avatar({ src, name, size = 40 }: AvatarProps) {
  const initial = name ? name.charAt(0).toUpperCase() : "?";

  if (src) {
    return (
      <img
        src={src}
        alt={name}
        onError={function (e) {
          e.currentTarget.style.display = "none";
        }}
        style={{
          width: size,
          height: size,
          borderRadius: theme.radius.full,
          objectFit: "cover",
        }}
      />
    );
  }

  return (
    <div
      style={{
        width: size,
        height: size,
        borderRadius: theme.radius.full,
        background: theme.colors.primary,
        color: theme.colors.background,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontSize: size > 32 ? theme.fontSize.base : theme.fontSize.xs,
        fontWeight: 700,
        flexShrink: 0,
      }}
    >
      {initial}
    </div>
  );
}
