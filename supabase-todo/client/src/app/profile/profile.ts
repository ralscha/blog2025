import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { SupabaseService } from '../supabase.service';
import { AvatarComponent } from './avatar';

@Component({
  selector: 'app-profile',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, AvatarComponent],
  templateUrl: './profile.html',
})
export class ProfileComponent implements OnInit {
  private readonly supabase = inject(SupabaseService);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder);

  loading = signal(false);
  signingOut = signal(false);
  errorMessage = signal<string | null>(null);
  successMessage = signal<string | null>(null);
  userEmail = signal('');
  avatarUrl = signal<string | null>(null);

  form = this.fb.group({});

  async ngOnInit() {
    const user = await this.supabase.getUser();
    if (!user) return;

    this.userEmail.set(user.email ?? '');

    const { data, error } = await this.supabase.profile();
    if (error) {
      this.errorMessage.set(error.message);
      return;
    }
    if (data) {
      this.avatarUrl.set(data.avatar_url ?? null);
    }
  }

  onAvatarUploaded(filePath: string) {
    this.avatarUrl.set(filePath);
    this.saveProfile();
  }

  async updateProfile() {
    if (this.form.invalid) return;
    await this.saveProfile();
  }

  private async saveProfile() {
    this.loading.set(true);
    this.errorMessage.set(null);
    this.successMessage.set(null);

    const user = await this.supabase.getUser();
    if (!user) {
      this.loading.set(false);
      this.errorMessage.set('Not authenticated');
      return;
    }

    const { error } = await this.supabase.updateProfile(
      {
        avatar_url: this.avatarUrl() ?? '',
      },
      user.id,
    );

    this.loading.set(false);

    if (error) {
      this.errorMessage.set(error.message);
      return;
    }

    this.successMessage.set('Profile updated!');
    setTimeout(() => this.successMessage.set(null), 3000);
  }

  async signOut() {
    this.signingOut.set(true);
    await this.supabase.signOut();
    this.router.navigate(['/sign-in']);
  }
}
