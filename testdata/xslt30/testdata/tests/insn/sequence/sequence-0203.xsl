<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: test op:concatenate on nested sequences using for clauses -->


   <t:template match="/">
      <out>
         <t:text>(</t:text>
         <t:value-of select="for $i in ( for $i in (1, 2,  for $i in (3, 4,  for $i in (5, 6) return $i  )  return $i )  return $i,  7,  for $i in (8, 9) return $i,  10 )  return $i"
                     separator=", "/>
         <t:text>)</t:text>
      </out>
   </t:template>
</t:transform>
