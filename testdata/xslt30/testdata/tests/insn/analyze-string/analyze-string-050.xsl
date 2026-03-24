<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xsl:analyze-string on a literal string where @regex uses backreferences. 
				Verifies that backreferences match the exact string that was already matched by the capturing 
				subexpressions they refer to and not just its pattern.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:text>
</t:text>
         <E1>
            <t:analyze-string select="'kikikeriki!! Tak, tak, tak! - da kommen sie.'"
                              regex="(da)( )(kommen)\2(sie)">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E1>
         <t:text>
</t:text>
         <E2>
            <t:analyze-string select="'kikikeriki!! Tak, tak, tak! - da kommen sie.'"
                              regex="(ki){{2}}ke.*\1">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E2>
         <t:text>
</t:text>
         <E3>
            <t:analyze-string select="'kikikeriki!! Tak, tak, tak! - da kommen sie.'"
                              regex="(ki|ke)\1">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E3>
         <t:text>
</t:text>
         <E4>
            <t:analyze-string select="'kikikeriki!! Tak, tak, tak! - da kommen sie.'" regex="(!).*\1">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E4>
         <t:text>
</t:text>
         <E5>
            <t:analyze-string select="'kikikeriki2!! Tak, tak, tak! - da kommen sie.'"
                              regex="(ki){{2}}ke.*\12">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E5>
         <t:text>
</t:text>
         <E6>
            <t:analyze-string select="''" regex="(.)\1">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E6>
         <t:text>
</t:text>
         <E7>
            <t:analyze-string select="'123kikikeriki123'" regex="([0-9]+)([a-z]+)\1">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E7>
         <t:text>
</t:text>
         <E8>
            <t:analyze-string select="'123kikikeriki4566'"
                              regex="(1)(2)(3)(ki)(k)(i)(ke)(ri)(ki)(4)(5)(6)\12">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E8>
         <t:text>
</t:text>
         <E9>
            <t:analyze-string select="'kikikeriki!! Tak, tak, tak! - da kommen sie.'"
                              regex="(da)( )(kommen)\2(die)">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E9>
         <t:text>
</t:text>
         <E11>
            <t:analyze-string select="'kikikeriki!! Tak, tak, tak! - da kommen sie.'"
                              regex="(ki){{2}}ke.*\12">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E11>
         <t:text>
</t:text>
         <E12>
            <t:analyze-string select="'123kikikeriki456'" regex="([0-9]+)([a-z]+)\1">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E12>
         <t:text>
</t:text>
         <E13>
            <t:analyze-string select="' '" regex="( )\1">
    	          <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E13>
      </out>
   </t:template>
</t:transform>
