import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { DataTablePagination } from "./data-table-pagination";

describe("DataTablePagination", () => {
  it("offers a rows-per-page picker when the table can change it", () => {
    render(
      <DataTablePagination
        currentPage={1}
        totalPages={3}
        onPageChange={() => {}}
        pageSize={20}
        onPageSizeChange={vi.fn()}
      />,
    );
    expect(screen.getByText("Rows per page")).toBeInTheDocument();
    expect(
      screen.getByRole("combobox", { name: "Rows per page" }),
    ).toHaveTextContent("20");
  });

  it("hides the picker otherwise", () => {
    render(
      <DataTablePagination
        currentPage={1}
        totalPages={3}
        onPageChange={() => {}}
      />,
    );
    expect(screen.queryByText("Rows per page")).not.toBeInTheDocument();
  });

  it("can't go past the last page", () => {
    render(
      <DataTablePagination
        currentPage={1}
        totalPages={0}
        onPageChange={() => {}}
      />,
    );
    expect(screen.getByRole("button", { name: "Next page" })).toBeDisabled();
  });
});
