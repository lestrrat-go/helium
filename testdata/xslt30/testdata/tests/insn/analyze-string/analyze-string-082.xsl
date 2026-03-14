<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test to verify that the value of current captured substring is unaffected by an expansion of named attribute sets.-->

   <t:attribute-set name="style">
      <t:attribute name="size">12</t:attribute>
      <t:attribute name="weight">500</t:attribute>
   </t:attribute-set>

   <t:template match="doc">
      <out>
         <t:analyze-string select="str1" regex="(a)B(c)D(e)F(g)H(i)J(k)L(m)N(o)P(q)">
            <t:matching-substring>
               <t:apply-templates select="doc('analyze-string-082.xml')//str1"/>
            </t:matching-substring>
            <t:non-matching-substring>
               <t:value-of select="."/>
            </t:non-matching-substring>
         </t:analyze-string>
      </out>
   </t:template>

   <t:template match="str1">
      <b t:use-attribute-sets="style">
         <t:value-of select="regex-group(1)"/>
      </b>
   </t:template>
</t:transform>
