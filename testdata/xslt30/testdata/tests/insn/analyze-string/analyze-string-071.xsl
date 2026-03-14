<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Testcase to verify that @flags in xsl:analyze-string can be used to set multiple modes 
      simultaneously and repeats are allowed.-->

   <t:template match="/">
      <out>
         <E1>
            <t:analyze-string select="poem"
               regex="(kaum)              (.*)               (gesehen,)$" flags="mixxxm">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </E1>
         <E2>
            <t:analyze-string select="poem" regex="(kaum)(.*) (krahen)" flags="ssmixxxm">
               <t:matching-substring>
                  <t:value-of select="'*'"/>
               </t:matching-substring>
               <t:non-matching-substring>
                  <t:value-of select="."/>
               </t:non-matching-substring>
            </t:analyze-string>
         </E2>
      </out>
   </t:template>
</t:transform>
