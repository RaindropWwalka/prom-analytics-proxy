import { Button } from "@/components/ui/button";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { LegacyColumn } from "@tanstack/react-table/legacy";
import { DataTableColumnHeaderProps, TableData } from "./types";

export function DataTableColumnHeader({
  column,
  title,
  className,
}: DataTableColumnHeaderProps) {
  // Type assertion for column as it comes from the interface as unknown
  const tableColumn = column as LegacyColumn<TableData, unknown>;

  if (!tableColumn.getCanSort()) {
    return <div className={className}>{title}</div>;
  }

  return (
    <Button
      variant="ghost"
      // -ml-2 cancels the button's own padding, so the title lines up with
      // the cells below; the sort icon sits right after the title rather
      // than at the column's edge, where it read as the next column's.
      className={`-ml-2 px-2 h-8 font-semibold flex items-center justify-start gap-1 ${className || ""}`}
      onClick={() => tableColumn.toggleSorting()}
    >
      <span>{title}</span>
      {tableColumn.getIsSorted() === "asc" ? (
        <ArrowUp className="h-4 w-4 shrink-0" />
      ) : tableColumn.getIsSorted() === "desc" ? (
        <ArrowDown className="h-4 w-4 shrink-0" />
      ) : (
        <ArrowUpDown className="h-4 w-4 shrink-0 text-muted-foreground" />
      )}
    </Button>
  );
}
