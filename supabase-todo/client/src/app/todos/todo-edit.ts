import { Component, ElementRef, inject, input, OnInit, signal, viewChild } from '@angular/core';
import { FormField, FormRoot, form, minLength, required } from '@angular/forms/signals';
import { Router } from '@angular/router';
import { SupabaseService, Todo } from '../supabase.service';

@Component({
  selector: 'app-todo-edit',
  imports: [FormField, FormRoot],
  templateUrl: './todo-edit.html',
})
export class TodoEditComponent implements OnInit {
  private readonly supabase = inject(SupabaseService);
  private readonly router = inject(Router);

  id = input<string>();

  isNew = signal(false);
  pageLoading = signal(false);
  saving = signal(false);
  deleting = signal(false);
  errorMessage = signal<string | null>(null);
  submitted = signal(false);

  private deleteModal = viewChild<ElementRef<HTMLDialogElement>>('deleteModal');

  readonly todoModel = signal({
    title: '',
    description: '',
    priority: 'medium' as Todo['priority'],
    due_date: '',
  });
  readonly form = form(this.todoModel, (path) => {
    required(path.title);
    minLength(path.title, 1);
  });

  private existingIsComplete = false;

  async ngOnInit() {
    const id = this.id();
    if (!id || id === 'new') {
      this.isNew.set(true);
      return;
    }

    const numId = parseInt(id, 10);
    if (isNaN(numId)) {
      this.router.navigate(['/todos']);
      return;
    }

    this.pageLoading.set(true);
    const { data, error } = await this.supabase.getTodo(numId);
    this.pageLoading.set(false);

    if (error || !data) {
      this.router.navigate(['/todos']);
      return;
    }

    const todo = data;
    this.existingIsComplete = todo.is_complete;
    this.todoModel.set({
      title: todo.title,
      description: todo.description ?? '',
      priority: todo.priority,
      due_date: todo.due_date ?? '',
    });
  }

  async onSubmit() {
    this.submitted.set(true);
    if (!this.form().valid()) return;

    this.saving.set(true);
    this.errorMessage.set(null);

    const { title, description, priority, due_date } = this.todoModel();

    const payload = {
      title,
      description: description || null,
      priority: (priority ?? 'medium') as Todo['priority'],
      due_date: due_date || null,
      is_complete: this.isNew() ? false : this.existingIsComplete,
    };

    let error: unknown;

    if (this.isNew()) {
      const result = await this.supabase.addTodo(payload);
      error = result.error;
    } else {
      const result = await this.supabase.updateTodo(parseInt(this.id()!, 10), payload);
      error = result.error;
    }

    this.saving.set(false);

    if (error) {
      this.errorMessage.set((error as Error).message);
      return;
    }

    this.router.navigate(['/todos']);
  }

  deleteTodo() {
    this.deleteModal()?.nativeElement.showModal();
  }

  cancelDelete() {
    this.deleteModal()?.nativeElement.close();
  }

  async confirmDelete() {
    this.deleting.set(true);
    const { error } = await this.supabase.deleteTodo(parseInt(this.id()!, 10));
    this.deleting.set(false);

    if (error) {
      this.deleteModal()?.nativeElement.close();
      this.errorMessage.set(error.message);
      return;
    }

    this.router.navigate(['/todos']);
  }

  goBack() {
    this.router.navigate(['/todos']);
  }
}
