/*
 * Copyright (C) 2012, 2013, 2014 by Jonathan Ross (jonross@alum.mit.edu)
 * 
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 * 
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 * 
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package com.myco;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class GenHeap
{
    private Map<Integer,Object> m1 = new MyHashMap<>();
    private Map<Integer,Object> m2 = new HashMap<>();
    private MyDom myDom = new MyDom();
    
    private int passes;

    GenHeap(int passes) {
        this.passes = passes;
    }
    
    void gen() throws Exception {
        
         // Generate custom map instance for simple skip testing.
        for (int i = 0; i < 10000; i++) {
            m1.put(i, String.valueOf(i));
        }

        List<Thing1> list = new ArrayList<>();
        
        for (int i = 1; i <= passes; i++) {
            Thing1 t = new Thing1(new Thing2[]{new Thing2(i-1), new Thing2(i), new Thing2(i+1)});
            list.add(t);
            if (i % 10 == 0) {
                m2.put(i, list);
                list = new ArrayList<>();
            }
        }
        
        System.err.println("Ready to dump, sleeping");
        Thread.sleep(60000);
    }
    
    public static void main(String[] args) throws Exception {
        GenHeap g = new GenHeap(Integer.parseInt(args[0]));
        g.gen();
    }
}

/**
 * Custom hashmap subclass we can easily target for query testing.
 */

@SuppressWarnings("serial")
class MyHashMap<K,V> extends HashMap<K,V>
{
}

/**
 * Other classes for same
 */

class Thing1 {
    private final static List<Long> extras = new ArrayList<>();
    static {
        for (long i = 0; i < 100; i++)
            extras.add(i);
    }
    Thing2[] things;
    Thing1(Thing2[] t) {
        things = t;
    }
}
 
class Thing2 {
    private final static List<Long> extras = new ArrayList<>();
    static {
        for (long i = 0; i < 200; i++)
            extras.add(i);
    }
    Integer value;
    Thing2(int v) {
        value = v;
    }
}

/**
 * For testing dominator tree; 10% of the held Long boxes should be dominated by
 * MyDom, not the map cells.  Make the values large enough to evade the autobox cache.
 */
 
class MyDom {
    Map<Long,Long> m = new HashMap<>();
    Long[] longs = new Long[100];
    
    {
        for (int x = 0; x < 1000; x++) {
            Long lx = (long) (x + 1000000);
            m.put(lx, lx);
            if (x < longs.length) {
                longs[x] = lx;
            }
        }
    }
}
