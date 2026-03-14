<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xsl:analyze-string using whitespace-ignore mode (@flags is set to 'x').-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">     
      <out>         
         <E1>
            <t:analyze-string select="poem" regex="(Kaum)              (.* )                (dies)">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E1>
         
         <E2>
            <t:analyze-string select="poem"
                              regex="(Kaum)              (.*)                (dies)"
                              flags="x">
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
