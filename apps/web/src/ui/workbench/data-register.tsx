import type { ReactNode } from "react";

import { MobileRecordCard } from "./mobile-record-card";

export interface DataRegisterColumn<Row> {
  key: keyof Row & string;
  header: string;
  render?: (row: Row) => ReactNode;
  mobileRender?: (row: Row) => ReactNode;
}

export interface DataRegisterProps<Row extends Record<string, ReactNode>> {
  caption: string;
  columns: readonly DataRegisterColumn<Row>[];
  rows: readonly Row[];
  rowKey(row: Row): string;
  mobileTitle?: (row: Row) => string;
}

function cellValue<Row extends Record<string, ReactNode>>(
  row: Row,
  column: DataRegisterColumn<Row>,
): ReactNode {
  return column.render ? column.render(row) : row[column.key];
}

export function DataRegister<Row extends Record<string, ReactNode>>({
  caption,
  columns,
  rows,
  rowKey,
  mobileTitle,
}: DataRegisterProps<Row>) {
  return (
    <div className="workbench-data-register">
      <table>
        <caption>{caption}</caption>
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={column.key} scope="col">
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={rowKey(row)}>
              {columns.map((column) => (
                <td key={column.key}>{cellValue(row, column)}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      <div aria-label={`${caption} records`} className="workbench-record-list">
        {rows.map((row) => (
          <MobileRecordCard
            fields={columns.map((column) => ({
              label: column.header,
              value: column.mobileRender ? column.mobileRender(row) : cellValue(row, column),
            }))}
            key={rowKey(row)}
            title={mobileTitle ? mobileTitle(row) : rowKey(row)}
          />
        ))}
      </div>
    </div>
  );
}
