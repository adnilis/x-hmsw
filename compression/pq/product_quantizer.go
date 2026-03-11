package pq

import (
	"fmt"
	"math"
)

// ProductQuantizer 乘积量化器
type ProductQuantizer struct {
	dimension int
	m         int // 子量化器数量
	k         int // 每个子量化器的聚类数
	centroids []float32
	trained   bool
}

// NewProductQuantizer 创建乘积量化器
func NewProductQuantizer(dimension, m, k int) *ProductQuantizer {
	if dimension%m != 0 {
		return nil
	}
	return &ProductQuantizer{
		dimension: dimension,
		m:         m,
		k:         k,
		trained:   false,
	}
}

// Train 训练量化器
func (pq *ProductQuantizer) Train(vectors [][]float32) error {
	if len(vectors) == 0 {
		return fmt.Errorf("no vectors to train")
	}

	subDim := pq.dimension / pq.m
	pq.centroids = make([]float32, pq.m*pq.k*subDim)

	// 对每个子空间进行 K-means 聚类
	for i := 0; i < pq.m; i++ {
		// 提取子空间向量
		subVectors := make([][]float32, len(vectors))
		for j, vec := range vectors {
			subVectors[j] = make([]float32, subDim)
			copy(subVectors[j], vec[i*subDim:(i+1)*subDim])
		}

		// K-means 聚类
		centroids, err := kmeans(subVectors, pq.k)
		if err != nil {
			return err
		}

		// 存储质心
		for j, centroid := range centroids {
			copy(pq.centroids[i*pq.k*subDim+j*subDim:], centroid)
		}
	}

	pq.trained = true
	return nil
}

// Encode 编码向量
func (pq *ProductQuantizer) Encode(vector []float32) []byte {
	if !pq.trained {
		return nil
	}

	subDim := pq.dimension / pq.m
	encoded := make([]byte, pq.m)

	for i := 0; i < pq.m; i++ {
		subVec := vector[i*subDim : (i+1)*subDim]
		bestIdx := -1
		bestDist := float32(math.MaxFloat32)

		// 找到最近的质心
		for j := 0; j < pq.k; j++ {
			centroid := pq.centroids[i*pq.k*subDim+j*subDim : i*pq.k*subDim+(j+1)*subDim]
			dist := distance(subVec, centroid)
			if dist < bestDist {
				bestDist = dist
				bestIdx = j
			}
		}

		encoded[i] = byte(bestIdx)
	}

	return encoded
}

// Decode 解码向量
func (pq *ProductQuantizer) Decode(encoded []byte) []float32 {
	if !pq.trained {
		return nil
	}

	subDim := pq.dimension / pq.m
	decoded := make([]float32, pq.dimension)

	for i := 0; i < pq.m; i++ {
		idx := encoded[i]
		start := i*pq.k*subDim + int(idx)*subDim
		end := start + subDim
		copy(decoded[i*subDim:(i+1)*subDim], pq.centroids[start:end])
	}

	return decoded
}

// kmeans K-means 聚类
func kmeans(data [][]float32, k int) ([][]float32, error) {
	n := len(data)
	if n == 0 {
		return nil, fmt.Errorf("no data points")
	}
	if len(data[0]) == 0 {
		return nil, fmt.Errorf("invalid dimension")
	}
	dim := len(data[0])
	if n < k {
		return nil, fmt.Errorf("not enough data points")
	}

	centroids := make([][]float32, k)
	assignments := make([]int, n)

	// 随机初始化质心
	for i := 0; i < k; i++ {
		centroids[i] = make([]float32, dim)
		copy(centroids[i], data[i])
	}

	// 迭代
	for iter := 0; iter < 100; iter++ {
		changed := false

		// 分配
		for i := 0; i < n; i++ {
			bestK := 0
			bestDist := float32(math.MaxFloat32)

			for j := 0; j < k; j++ {
				dist := distance(data[i], centroids[j])
				if dist < bestDist {
					bestDist = dist
					bestK = j
				}
			}

			if assignments[i] != bestK {
				changed = true
				assignments[i] = bestK
			}
		}

		// 更新质心
		counts := make([]int, k)
		newCentroids := make([][]float32, k)
		for i := 0; i < k; i++ {
			newCentroids[i] = make([]float32, dim)
		}

		for i := 0; i < n; i++ {
			clusterIdx := assignments[i]
			counts[clusterIdx]++
			for j := 0; j < dim; j++ {
				newCentroids[clusterIdx][j] += data[i][j]
			}
		}

		for i := 0; i < k; i++ {
			if counts[i] > 0 {
				for j := 0; j < dim; j++ {
					newCentroids[i][j] /= float32(counts[i])
				}
			}
		}

		centroids = newCentroids

		if !changed {
			break
		}
	}

	return centroids, nil
}

// distance 欧几里得距离
func distance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return sum
}

// Codebook 码本
type Codebook struct {
	Centroids [][]float32
}

// NewCodebook 创建码本
func NewCodebook(centroids [][]float32) *Codebook {
	return &Codebook{Centroids: centroids}
}

// Encode 编码
func (cb *Codebook) Encode(vector []float32) int {
	bestIdx := -1
	bestDist := float32(math.MaxFloat32)

	for i, centroid := range cb.Centroids {
		dist := distance(vector, centroid)
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	return bestIdx
}

// Decode 解码
func (cb *Codebook) Decode(index int) []float32 {
	if index < 0 || index >= len(cb.Centroids) {
		return nil
	}
	return cb.Centroids[index]
}

// Quantize 量化向量
func Quantize(vectors [][]float32, m int, k int) (*ProductQuantizer, error) {
	if len(vectors) == 0 {
		return nil, fmt.Errorf("no vectors to quantize")
	}

	dimension := len(vectors[0])
	pq := NewProductQuantizer(dimension, m, k)
	if pq == nil {
		return nil, fmt.Errorf("invalid parameters")
	}

	if err := pq.Train(vectors); err != nil {
		return nil, err
	}

	return pq, nil
}
