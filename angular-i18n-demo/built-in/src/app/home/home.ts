import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-home',
  imports: [CommonModule],
  templateUrl: './home.html'
})
export class Home {
  name = 'Alex';
  gender: 'male' | 'female' | 'other' = 'other';
  unreadCount = 1;
  itemCount = 2;
  today = new Date();
  price = 1234.56;
  percent = 0.42;
  items = [
    $localize`:fruitApples:Apples`,
    $localize`:fruitBananas:Bananas`,
    $localize`:fruitCherries:Cherries`
  ];
  greeting = $localize`Hello ${this.name}, you have ${this.itemCount} items in your cart.`;
}
