<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test to verify that the zero-length string is returned if there is no matched substring  with the number passed as an argument to regex-group().-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="doc">
      <out>
         <t:analyze-string select="str1" regex="(a)B(c)D(e)F(g)H(i)J(k)L(m)N(o)P(q)">
            <t:matching-substring>
               <t:value-of select="regex-group(10)"/>
            </t:matching-substring>
            <t:non-matching-substring>
               <t:value-of select="'#'"/>
            </t:non-matching-substring>
         </t:analyze-string>
      </out>
   </t:template>
</t:transform>
