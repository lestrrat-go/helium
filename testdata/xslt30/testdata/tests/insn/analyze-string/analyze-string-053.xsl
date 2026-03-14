<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xsl:analyze-string where the value of @select is a the result of a function call.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <E1>
            <t:analyze-string select="replace('banana','(an).*?a','#')" regex="#">
    	          <t:matching-substring>
    		            <t:value-of select="current()"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="1"/>
    	          </t:non-matching-substring>
            </t:analyze-string>
         </E1>
      </out>
   </t:template>
</t:transform>
