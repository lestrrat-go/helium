<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xsl:analyze-string on a literal string where @regex uses reluctant quantifiers.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:text>
</t:text>
         <e1>
            <t:analyze-string select="'aaaa'" regex="a+">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </e1>
         <t:text>
</t:text>
         <E1>
            <t:analyze-string select="'aaaa'" regex="a+?">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E1>
         <t:text>
</t:text>
         <E2>
            <t:analyze-string select="'aaaaaa'" regex="a{{2}}?">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E2>
         <t:text>
</t:text>
         <e3>
            <t:analyze-string select="'aaaaaa'" regex="a{{2,}}">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </e3>
         <t:text>
</t:text>
         <E3>
            <t:analyze-string select="'aaaaaa'" regex="a{{2,}}?">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E3>
         <t:text>
</t:text>
         <e4>
            <t:analyze-string select="'aaaaaa'" regex="a{{2,3}}">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </e4>
         <t:text>
</t:text>
         <E4>
            <t:analyze-string select="'aaaaaa'" regex="a{{2,3}}?">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E4>
         <t:text>
</t:text>
         <E5>
            <t:analyze-string select="'a'" regex="a{{2,3}}?">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E5>
         <t:text>
</t:text>
         <E6>
            <t:analyze-string select="''" regex=".+?">
    	          <t:matching-substring>
    		            <t:value-of select="'b'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E6>
      </out>
   </t:template>
</t:transform>
