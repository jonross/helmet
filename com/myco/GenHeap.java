/*
 * Copyright (C) 2012, 2013 by Jonathan Ross (jonross@alum.mit.edu)
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
import java.util.Random;

public class GenHeap
{
    private Map<Integer,Object> m = new HashMap<>();

    class Thing {
        Integer value;
    }
    
    {
        Random random = new Random();
        List<Thing> list = new ArrayList<>();
        
        for (int i = 0; i < 1000; i++) {
            Thing t = new Thing();
            t.value = i;
            list.add(t);
            if (random.nextInt(10) == 0) {
                m.put(i, list);
                list = new ArrayList<>();
            }
        }
        
        /*
        for (int i = 0; i < 10000; i++) {
            m.put(i, String.valueOf(i));
        }
        */
    }
    
    public static void main(String[] args) throws Exception {
        GenHeap g = new GenHeap();
        Thread.sleep(60000);
    }
}
