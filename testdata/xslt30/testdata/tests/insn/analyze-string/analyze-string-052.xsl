<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Testcase where the result from xsl:analyze-string is set as the value of a variable which is used for @select in another xsl:analyze-string element.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="var">
      <t:analyze-string select="'Mary had a little lamb.'" regex=".a">
         <t:matching-substring>
            <t:value-of select="."/>
         </t:matching-substring>
         <t:non-matching-substring>
            <t:value-of select="'*'"/>
         </t:non-matching-substring>
      </t:analyze-string>
   </t:variable>

   <t:template match="/">
      <out>
         <A>
            <t:value-of select="$var"/>
         </A>
         <B>
            <t:analyze-string select="$var" regex=".a">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="'!'"/>
               </t:non-matching-substring>
            </t:analyze-string>
         </B>
      </out>
   </t:template>
</t:transform>
