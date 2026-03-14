<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xsl:analyze-string where @regex matches several overlapping substrings of different length.
				Cases tested: @regex contains a greedy quantifier
							  @regex contains a non-greedy quantifier 
							  @regex contains alternatives that both match different substrings-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:text>
</t:text>
         <E1>
            <t:analyze-string select="'banana'" regex="(an)*a">
  		           <t:matching-substring>
    		            <t:value-of select="'#'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E1>
         <t:text>
</t:text>
         <E2>
            <t:analyze-string select="'banana'" regex="(an)+?a">
  		           <t:matching-substring>
    		            <t:value-of select="'#'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E2>
         <t:text>
</t:text>
         <E3>
            <t:analyze-string select="'banana'" regex="a|ana">
  		           <t:matching-substring>
    		            <t:value-of select="'#'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E3>
         <t:text>
</t:text>
         <E4>
            <t:analyze-string select="'banana'" regex="ana|a">
  		           <t:matching-substring>
    		            <t:value-of select="'#'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E4>
      </out>
   </t:template>
</t:transform>
