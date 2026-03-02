import { Pipe, PipeTransform } from '@angular/core';

@Pipe({ name: 'localeDate' })
export class LocaleDatePipe implements PipeTransform {
  transform(value: string | null | undefined): string {
    if (!value) return '';
    return new Date(`${value}T00:00:00`).toLocaleDateString();
  }
}
