import { Component, inject, signal } from '@angular/core';
import { email, FormField, FormRoot, form, required } from '@angular/forms/signals';
import { Router, RouterLink } from '@angular/router';
import { SupabaseService } from '../supabase.service';

@Component({
  selector: 'app-sign-in',
  imports: [FormField, FormRoot, RouterLink],
  templateUrl: './sign-in.html',
})
export class SignInComponent {
  private readonly supabase = inject(SupabaseService);
  private readonly router = inject(Router);

  loading = signal(false);
  errorMessage = signal<string | null>(null);
  submitted = signal(false);

  readonly signInModel = signal({
    email: '',
    password: '',
  });
  readonly form = form(this.signInModel, (path) => {
    required(path.email);
    email(path.email);
    required(path.password);
  });

  async onSubmit() {
    this.submitted.set(true);
    if (!this.form().valid()) return;

    this.loading.set(true);
    this.errorMessage.set(null);

    const { email, password } = this.signInModel();
    const { error } = await this.supabase.signIn(email, password);

    this.loading.set(false);

    if (error) {
      this.errorMessage.set(error.message);
      return;
    }

    this.router.navigate(['/todos']);
  }
}
