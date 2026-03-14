<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case in which use-when is used with an expression containing a local variable in a quantitative exp.  -->

   <t:template match="elem">
      <out>
         <t:apply-templates select="a | b"/>
      </out>
   </t:template>

   <t:template match="a" use-when="every $x in (1,2,3) satisfies $x=2">
      <success>
         <t:next-match/>
      </success>
   </t:template>

   <t:template match="b">
      <unsuccessful>
         <t:next-match use-when="every $x in (1,1,1) satisfies $x=1"/>
      </unsuccessful>
   </t:template>
</t:transform>
