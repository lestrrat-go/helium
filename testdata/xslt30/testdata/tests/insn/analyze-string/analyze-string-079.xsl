<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test to verify that the value of current captured substrings is unaffected through a call to xsl:call-template.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="doc">
      <out>
         <t:analyze-string select="str1" regex="(a)B(c)D(e)F(g)H(i)J(k)L(m)N(o)P(q)">
            <t:matching-substring>
               <t:call-template name="str1"/>
            </t:matching-substring>
      
            <t:non-matching-substring>
               <t:value-of select="."/>
            </t:non-matching-substring>
         </t:analyze-string>
      </out>
   </t:template>

   <t:template name="str1">
 	    <t:value-of select="regex-group(1)"/>
   </t:template>
</t:transform>
