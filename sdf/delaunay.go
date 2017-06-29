//-----------------------------------------------------------------------------
/*

Dealunay Triangulation

See:
http://www.mathopenref.com/trianglecircumcircle.html
http://paulbourke.net/papers/triangulate/

*/
//-----------------------------------------------------------------------------

package sdf

import (
	"errors"
	"sort"
)

//-----------------------------------------------------------------------------

// 2d/3d triangle referencing a list of vertices
type TriangleI [3]int

// 2d/3d edge referencing a list of vertices
type EdgeI [2]int

//-----------------------------------------------------------------------------

// return the super triangle of the point set, ie: 3 vertices enclosing all points
func (s V2Set) SuperTriangle() ([]V2, error) {

	if len(s) == 0 {
		return nil, errors.New("no vertices")
	}

	var p V2
	var k float64

	if len(s) == 1 {
		// a single point
		p := s[0]
		k := p.MaxComponent() * 0.125
		if k == 0 {
			k = 1
		}
	} else {
		b := Box2{s.Min(), s.Max()}
		p = b.Center()
		k = b.Size().MaxComponent() * 2.0
	}

	p0 := p.Add(V2{-k, -k})
	p1 := p.Add(V2{0, k})
	p2 := p.Add(V2{k, -k})
	return []V2{p0, p1, p2}, nil
}

//-----------------------------------------------------------------------------

// Return the circumcenter of a triangle
func (t Triangle2) Circumcenter() (V2, error) {

	var m1, m2, mx1, mx2, my1, my2 float64
	var xc, yc float64

	x1 := t[0].X
	x2 := t[1].X
	x3 := t[2].X

	y1 := t[0].Y
	y2 := t[1].Y
	y3 := t[2].Y

	fabsy1y2 := Abs(y1 - y2)
	fabsy2y3 := Abs(y2 - y3)

	// Check for coincident points
	if fabsy1y2 < EPSILON && fabsy2y3 < EPSILON {
		return V2{}, errors.New("coincident points")
	}

	if fabsy1y2 < EPSILON {
		m2 = -(x3 - x2) / (y3 - y2)
		mx2 = (x2 + x3) / 2.0
		my2 = (y2 + y3) / 2.0
		xc = (x2 + x1) / 2.0
		yc = m2*(xc-mx2) + my2
	} else if fabsy2y3 < EPSILON {
		m1 = -(x2 - x1) / (y2 - y1)
		mx1 = (x1 + x2) / 2.0
		my1 = (y1 + y2) / 2.0
		xc = (x3 + x2) / 2.0
		yc = m1*(xc-mx1) + my1
	} else {
		m1 = -(x2 - x1) / (y2 - y1)
		m2 = -(x3 - x2) / (y3 - y2)
		mx1 = (x1 + x2) / 2.0
		mx2 = (x2 + x3) / 2.0
		my1 = (y1 + y2) / 2.0
		my2 = (y2 + y3) / 2.0
		xc = (m1*mx1 - m2*mx2 + my2 - my1) / (m1 - m2)
		if fabsy1y2 > fabsy2y3 {
			yc = m1*(xc-mx1) + my1
		} else {
			yc = m2*(xc-mx2) + my2
		}
	}

	return V2{xc, yc}, nil
}

// Return inside = true if the point is inside the circumcircle of the triangle.
// Return done = true if the vertex (and the subsequent x-ordered vertices) is outside the circumcircle.
func (t Triangle2) InCircumcircle(p V2) (inside, done bool) {
	c, err := t.Circumcenter()
	if err != nil {
		inside = false
		done = true
		return
	}

	// radius squared of circumcircle
	dx := t[0].X - c.X
	dy := t[0].Y - c.Y
	r2 := dx*dx + dy*dy

	// distance squared from circumcenter to point
	dx = p.X - c.X
	dy = p.Y - c.Y
	d2 := dx*dx + dy*dy

	// is the point within the circumcircle?
	inside = d2-r2 <= EPSILON

	// If this vertex has an x-value beyond the circumcenter and the distance based on the x-delta
	// is greater than the circumradius, then this triangle is done for this and all subsequent vertices
	// since the vertex list has been sorted by x-value.
	done = (dx > 0) && (dx*dx > r2)

	return
}

//-----------------------------------------------------------------------------

func (vs V2Set) Delaunay2d() ([]TriangleI, error) {

	// number of vertices
	n := len(vs)

	// sort the vertices by x value
	sort.Sort(V2SetByX(vs))

	// work out the super triangle
	t, err := vs.SuperTriangle()
	if err != nil {
		return nil, err
	}
	// add the super triangle to the vertex set
	vs = append(vs, t...)

	// allocate the triangles
	ts := make([]TriangleI, 0, n)
	done := make([]bool, 0, n)

	// set the super triangle as the 0th triangle
	ts = append(ts, TriangleI{n, n + 1, n + 2})
	done = append(done, false)

	// Add the vertices one at a time into the mesh
	// Note: we don't iterate over the super triangle vertices
	for i := 0; i < n; i++ {
		v := vs[i]

		// Create the edge buffer.
		// If the vertex lies inside the circumcircle of the triangle
		// then the three edges of that triangle are added to the edge
		// buffer and that triangle is removed.
		es := make([]EdgeI, 0, 32)
		nt := len(ts)
		for j := 0; j < nt; j++ {

			if done[j] {
				continue
			}

			t := Triangle2{vs[ts[j][0]], vs[ts[j][1]], vs[ts[j][2]]}
			inside, complete := t.InCircumcircle(v)
			done[j] = complete

			if inside {
				// add the triangle edges to the edge set
				es = append(es, EdgeI{ts[j][0], ts[j][1]})
				es = append(es, EdgeI{ts[j][1], ts[j][2]})
				es = append(es, EdgeI{ts[j][2], ts[j][0]})
				// remove the triangle (copy in the tail)
				ts[j] = ts[nt-1]
				done[j] = done[nt-1]
				nt -= 1
				j -= 1
			}
		}

		// re-size the triangle/done sets
		ts = ts[:nt]
		done = done[:nt]

		// Tag multiple edges. If all triangles are specified anticlockwise
		// then all interior edges are opposite pointing in direction.
		for j := 0; j < len(es)-1; j++ {
			for k := j + 1; k < len(es); k++ {
				if (es[j][0] == es[k][1]) && (es[j][1] == es[k][0]) {
					es[j] = EdgeI{-1, -1}
					es[k] = EdgeI{-1, -1}
				}
				// Shouldn't need the following, see note above
				if (es[j][0] == es[k][0]) && (es[j][1] == es[k][1]) {
					es[j] = EdgeI{-1, -1}
					es[k] = EdgeI{-1, -1}
				}
			}
		}

		// Form new triangles for the current point skipping over any tagged edges.
		// All edges are arranged in clockwise order.
		for _, e := range es {
			if e[0] < 0 || e[1] < 0 {
				continue
			}
			ts = append(ts, TriangleI{e[0], e[1], i})
			done = append(done, false)
		}

	}

	// remove any triangles with vertices from the super triangle
	nt := len(ts)
	for j := 0; j < nt; j++ {
		t := ts[j]
		if t[0] >= n || t[1] >= n || t[2] >= n {
			// remove the triangle (copy in the tail)
			ts[j] = ts[nt-1]
			nt -= 1
			j -= 1
		}
	}
	// re-size the triangle set
	ts = ts[:nt]

	// done
	return ts, nil
}

//-----------------------------------------------------------------------------

/*

typedef struct {
   int p1,p2,p3;
} ITRIANGLE;
typedef struct {
   int p1,p2;
} IEDGE;
typedef struct {
   double x,y,z;
} XYZ;

*/

/*
   Triangulation subroutine
   Takes as input NV vertices in array pxyz
   Returned is a list of ntri triangular faces in the array v
   These triangles are arranged in a consistent clockwise order.
   The triangle array 'v' should be malloced to 3 * nv
   The vertex array pxyz must be big enough to hold 3 more points
   The vertex array must be sorted in increasing x values say

   qsort(p,nv,sizeof(XYZ),XYZCompare);
      :
   int XYZCompare(void *v1,void *v2)
   {
      XYZ *p1,*p2;
      p1 = v1;
      p2 = v2;
      if (p1->x < p2->x)
         return(-1);
      else if (p1->x > p2->x)
         return(1);
      else
         return(0);
   }
*/

/*

int Triangulate(int nv,XYZ *pxyz,ITRIANGLE *v,int *ntri)
{
   int *complete = NULL;
   IEDGE *edges = NULL;
   int nedge = 0;
   int trimax,emax = 200;
   int status = 0;

   int inside;
   int i,j,k;
   double xp,yp,x1,y1,x2,y2,x3,y3,xc,yc,r;
   double xmin,xmax,ymin,ymax,xmid,ymid;
   double dx,dy,dmax;

   // Allocate memory for the completeness list, flag for each triangle
   trimax = 4 * nv;
   if ((complete = malloc(trimax*sizeof(int))) == NULL) {
      status = 1;
      goto skip;
   }

   // Allocate memory for the edge list
   if ((edges = malloc(emax*(long)sizeof(EDGE))) == NULL) {
      status = 2;
      goto skip;
   }


   // Find the maximum and minimum vertex bounds.
   // This is to allow calculation of the bounding triangle

   xmin = pxyz[0].x;
   ymin = pxyz[0].y;
   xmax = xmin;
   ymax = ymin;
   for (i=1;i<nv;i++) {
      if (pxyz[i].x < xmin) xmin = pxyz[i].x;
      if (pxyz[i].x > xmax) xmax = pxyz[i].x;
      if (pxyz[i].y < ymin) ymin = pxyz[i].y;
      if (pxyz[i].y > ymax) ymax = pxyz[i].y;
   }
   dx = xmax - xmin;
   dy = ymax - ymin;
   dmax = (dx > dy) ? dx : dy;
   xmid = (xmax + xmin) / 2.0;
   ymid = (ymax + ymin) / 2.0;

   //  Set up the supertriangle
   //  This is a triangle which encompasses all the sample points.
   //  The supertriangle coordinates are added to the end of the
   //  vertex list. The supertriangle is the first triangle in
   //  the triangle list.

   pxyz[nv+0].x = xmid - 20 * dmax;
   pxyz[nv+0].y = ymid - dmax;
   pxyz[nv+0].z = 0.0;
   pxyz[nv+1].x = xmid;
   pxyz[nv+1].y = ymid + 20 * dmax;
   pxyz[nv+1].z = 0.0;
   pxyz[nv+2].x = xmid + 20 * dmax;
   pxyz[nv+2].y = ymid - dmax;
   pxyz[nv+2].z = 0.0;
   v[0].p1 = nv;
   v[0].p2 = nv+1;
   v[0].p3 = nv+2;
   complete[0] = FALSE;
   *ntri = 1;

   // Include each point one at a time into the existing mesh

   for (i=0;i<nv;i++) {

      xp = pxyz[i].x;
      yp = pxyz[i].y;
      nedge = 0;

      // Set up the edge buffer.
      // If the point (xp,yp) lies inside the circumcircle then the
      // three edges of that triangle are added to the edge buffer
      // and that triangle is removed.

      for (j=0;j<(*ntri);j++) {
         if (complete[j])
            continue;
         x1 = pxyz[v[j].p1].x;
         y1 = pxyz[v[j].p1].y;
         x2 = pxyz[v[j].p2].x;
         y2 = pxyz[v[j].p2].y;
         x3 = pxyz[v[j].p3].x;
         y3 = pxyz[v[j].p3].y;
         inside = CircumCircle(xp,yp,x1,y1,x2,y2,x3,y3,&xc,&yc,&r);
         if (xc < xp && ((xp-xc)*(xp-xc)) > r)
				complete[j] = TRUE;
         if (inside) {
            // Check that we haven't exceeded the edge list size
            if (nedge+3 >= emax) {
               emax += 100;
               if ((edges = realloc(edges,emax*(long)sizeof(EDGE))) == NULL) {
                  status = 3;
                  goto skip;
               }
            }
            edges[nedge+0].p1 = v[j].p1;
            edges[nedge+0].p2 = v[j].p2;
            edges[nedge+1].p1 = v[j].p2;
            edges[nedge+1].p2 = v[j].p3;
            edges[nedge+2].p1 = v[j].p3;
            edges[nedge+2].p2 = v[j].p1;
            nedge += 3;
            v[j] = v[(*ntri)-1];
            complete[j] = complete[(*ntri)-1];
            (*ntri)--;
            j--;
         }
      }

      // Tag multiple edges
      // Note: if all triangles are specified anticlockwise then all
      //   interior edges are opposite pointing in direction.

      for (j=0;j<nedge-1;j++) {
         for (k=j+1;k<nedge;k++) {
            if ((edges[j].p1 == edges[k].p2) && (edges[j].p2 == edges[k].p1)) {
               edges[j].p1 = -1;
               edges[j].p2 = -1;
               edges[k].p1 = -1;
               edges[k].p2 = -1;
            }
            // Shouldn't need the following, see note above
            if ((edges[j].p1 == edges[k].p1) && (edges[j].p2 == edges[k].p2)) {
               edges[j].p1 = -1;
               edges[j].p2 = -1;
               edges[k].p1 = -1;
               edges[k].p2 = -1;
            }
         }
      }

      // Form new triangles for the current point
      // Skipping over any tagged edges.
      // All edges are arranged in clockwise order.

      for (j=0;j<nedge;j++) {
         if (edges[j].p1 < 0 || edges[j].p2 < 0)
            continue;
         if ((*ntri) >= trimax) {
            status = 4;
            goto skip;
         }
         v[*ntri].p1 = edges[j].p1;
         v[*ntri].p2 = edges[j].p2;
         v[*ntri].p3 = i;
         complete[*ntri] = FALSE;
         (*ntri)++;
      }
   }

   // Remove triangles with supertriangle vertices
   // These are triangles which have a vertex number greater than nv

   for (i=0;i<(*ntri);i++) {
      if (v[i].p1 >= nv || v[i].p2 >= nv || v[i].p3 >= nv) {
         v[i] = v[(*ntri)-1];
         (*ntri)--;
         i--;
      }
   }

skip:
   free(edges);
   free(complete);
   return(status);
}

*/

//-----------------------------------------------------------------------------
