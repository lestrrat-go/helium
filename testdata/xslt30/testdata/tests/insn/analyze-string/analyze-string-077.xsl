<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://my_functions_example.org"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test to verify that the set of captured substrings is set to an empty sequence inside stylesheet function.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:function name="my:function">
	     <t:text>inside the function</t:text>
	     <b xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="regex-group(1)"/>
      </b>
   </t:function>

   <t:template match="doc">
      <out>
         <t:analyze-string select="str1" regex="(a)B(c)D(e)F(g)H(i)J(k)L(m)N(o)P(q)">
            <t:matching-substring>
               <t:value-of select="my:function()"/>
            </t:matching-substring>
            <t:non-matching-substring>
               <t:text>  </t:text>
               <t:value-of select="."/>
               <t:text>  </t:text>
            </t:non-matching-substring>
         </t:analyze-string>
      </out>
   </t:template>
</t:transform>
