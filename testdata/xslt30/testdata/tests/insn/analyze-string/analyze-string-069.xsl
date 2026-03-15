<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Testcase with xsl:analyze-string using case-sensitive mode (@flags is set to 'i').-->


   <t:template match="doc">

      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="str1|str2">
      <r name="{name()}">
         <e1>
            <t:analyze-string select="." regex="GHIJ">
               <t:matching-substring>
                  <t:value-of select="'!!!!'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </e1>

         <E1>
            <t:analyze-string select="." regex="GHIJ" flags="i">
               <t:matching-substring>
                  <t:value-of select="'!!!!'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </E1>

         <e2>
            <t:analyze-string select="." regex="efGhijk">
               <t:matching-substring>
                  <t:value-of select="'1234567'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </e2>

         <E2>
            <t:analyze-string select="." regex="efGhijk" flags="i">
               <t:matching-substring>
                  <t:value-of select="'1234567'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </E2>

         <e3>
            <t:analyze-string select="." regex="e([a-z]*)H([A-Z]*)k">
               <t:matching-substring>
                  <t:value-of select="'#'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </e3>

         <E3>
            <t:analyze-string select="." regex="e([a-z]*)H([A-Z]*)k" flags="i">
               <t:matching-substring>
                  <t:value-of select="'#'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </E3>
      </r>
   </t:template>
</t:transform>
