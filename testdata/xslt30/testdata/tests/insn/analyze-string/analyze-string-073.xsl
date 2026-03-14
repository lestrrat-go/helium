<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with regex-group(0) to verify that the returned substring is the whole value of @regex. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="doc">
      <out>
         <t:analyze-string select="str1" regex="(a)B(c)D(e)F(g)H(i)J(k)L(m)N(o)P(q)">
            <t:matching-substring>
               <t:value-of select="regex-group(0)"/>
            </t:matching-substring>
            <t:non-matching-substring>
               <t:value-of select="'#'"/>
            </t:non-matching-substring>
         </t:analyze-string>
      </out>
   </t:template>
</t:transform>
