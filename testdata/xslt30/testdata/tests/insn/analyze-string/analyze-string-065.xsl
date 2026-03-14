<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xsl:analyze-string using 'dot-all' mode (@flags is set to 's').-->


   <t:template match="/">     
      <out>        
         <E1>
            <t:analyze-string select="//str1" regex="hello.*world">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
            </t:analyze-string>
         </E1>
         
         <E2>
            <t:analyze-string select="//str1" regex="hello.*world" flags="s">
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
