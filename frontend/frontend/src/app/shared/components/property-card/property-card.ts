import { Component, EventEmitter, Input, Output, Signal } from '@angular/core';
import { PropertyResponseModel } from '../../models/properties-model';

@Component({
  selector: 'app-property-card',
  imports: [],
  templateUrl: './property-card.html',
  styleUrl: './property-card.css',
})
export class PropertyCard {
  @Input({ required: true }) propertyData!: PropertyResponseModel;
  @Output() back = new EventEmitter<boolean>();
}
