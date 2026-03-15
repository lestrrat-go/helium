<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with xsl:analyze-string on a literal string, node, context node(.), variable and a subsequence where @regex value is resolved at run-time.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="input" select="/doc/str4"/>

   <t:variable name="regex1" select="/doc/str7"/>

   <t:param name="regex2" select="/doc/str8"/>

   <t:template match="/">
	     <t:apply-templates select="doc/str1"/>
   </t:template>

   <t:template match="str1">
      <out>
         <t:text>
</t:text>
         <E1>
            <t:analyze-string select="." regex="{.}">
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
            <t:analyze-string select="//str1" regex="{.}">
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
            <t:analyze-string select="//doc" regex="{$regex1}">
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
            <t:analyze-string select="//str2" regex="{$regex1}">
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
            <t:analyze-string select="$input" regex="{//str9}\1">
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
            <t:analyze-string select="subsequence((//str1, '', //str5),3,1)" regex=".{$regex1}">
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
            <t:analyze-string select="//str5" regex="{$regex2}">
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
      </out>
   </t:template>
</t:transform>
