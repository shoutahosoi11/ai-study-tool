import { theme } from "../../theme";

type InputProps = {
  label?: string;
  error?: string;
  type?: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
};

export function Input({
  label,
  error,
  type = "text",
  value,
  onChange,
  placeholder,
  disabled,
}: InputProps) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: theme.spacing.xs }}>
      {label && (
        <label style={{ fontSize: theme.fontSize.sm, fontWeight: 600, color: theme.colors.primary }}>
          {label}
        </label>
      )}
      <input
        type={type}
        value={value}
        onChange={function (e) {
          onChange(e.target.value);
        }}
        placeholder={placeholder}
        disabled={disabled}
        style={{
          padding: `${theme.spacing.sm} ${theme.spacing.md}`,
          border: `1px solid ${error ? theme.colors.danger : theme.colors.border}`,
          borderRadius: theme.radius.sm,
          fontSize: theme.fontSize.base,
          outline: "none",
          background: theme.colors.background,
          color: theme.colors.primary,
          width: "100%",
        }}
      />
      {error && (
        <span style={{ fontSize: theme.fontSize.xs, color: theme.colors.danger }}>
          {error}
        </span>
      )}
    </div>
  );
}
