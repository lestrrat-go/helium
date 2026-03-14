<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Testcase to verify that when @flags in xsl:analyze-string is omitted, 
   the effect is the same as supplying a zero-length string.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="var" select="/doc/str1"/>

   <t:template match="/">
      <out>
         <a>
            <t:analyze-string select="'Mary had a little lamb.'" regex="[^a-z]a">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </a>

         <b>
            <t:analyze-string select="'Mary had a little lamb.'" regex="[^a-z]a" flags="">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </b>

         <c>
            <t:analyze-string select="//str1" regex="[^a-z]a">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </c>

         <d>
            <t:analyze-string select="//str1" regex="[^a-z]a" flags="">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </d>
         <e>
            <t:analyze-string select="$var" regex="[^a-z]a">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </e>

         <f>
            <t:analyze-string select="$var" regex="[^a-z]a" flags="">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </f>

         <g>
            <t:analyze-string select="subsequence(('', 'Mary had a little lamb.', 'this'),2,1)"
               regex="[^a-z]a">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </g>

         <h>
            <t:analyze-string select="subsequence(('', 'Mary had a little lamb.', 'this'),2,1)"
               regex="[^a-z]a" flags="">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </h>

      </out>
   </t:template>
</t:transform>
