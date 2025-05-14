export type Algorithm = 'bfs' | 'dfs' | 'bidire';
export type TreeType = 'best-recipes-tree' | 'multiple-recipes-tree';

export interface ElementData {
  name: string;
  imagePath?: string;
  localImage?: string;
  tier?: number;
  recipes?: ElementRecipe[];
}

export interface ElementRecipe {
  ingredients: string[];
}

export interface TreeData {
  name: string;
  imagePath?: string;
  isBaseElement?: boolean;
  isCircularReference?: boolean;
  noRecipe?: boolean;
  ingredients: TreeData[];
}

export interface PathNode {
  element: string;
  Element?: string;
  imagePath?: string;
  ImagePath?: string;
  ingredients?: string[];
  Ingredients?: string[];
}

export interface AlgorithmResult {
  paths?: PathNode[][];
  trees?: TreeData[];
  timeElapsed?: number;
  nodesVisited?: number;
}