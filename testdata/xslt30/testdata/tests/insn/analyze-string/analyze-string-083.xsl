<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Testcase with xsl:analyze-string to test context item, position and size inside xsl:(non)-matching-substring.-->

   <t:template match="/doc">
      <out>
         <t:analyze-string select="str[3]" regex="a">
            <t:matching-substring>
               <match>
                  <item>
                     <t:value-of select="current()"/>
                  </item>
                  <pos>
                     <t:value-of select="position()"/>
                  </pos>
                  <size>
                     <t:value-of select="last()"/>
                  </size>
               </match>
            </t:matching-substring>
            <t:non-matching-substring>
               <non-match>
                  <item>
                     <t:value-of select="current()"/>
                  </item>
                  <pos>
                     <t:value-of select="position()"/>
                  </pos>
                  <size>
                     <t:value-of select="last()"/>
                  </size>
               </non-match>
            </t:non-matching-substring>
         </t:analyze-string>
      </out>
   </t:template>
</t:transform>
