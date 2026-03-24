<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test two AVTs with literal element between them 
  				in @flags and single AVT in @regex of xsl:analyze-string. 
  				AVT value comes from a node or a variable.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="var1" select="/doc/str[1]"/>

   <t:variable name="var2" select="/doc/str[2]"/>

   <t:template match="/doc">
      <out>
         <E1>
            <t:analyze-string select="poem"
                              regex="({item})              (.*)               (gesehen,)$"
                              flags="{str[1]}i{str[2]}">
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
                              regex="({item})(.*) (krahen)"
                              flags="ss{str[1]}i{str[2]}xxm">
  	            <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E2>
         <E3>
            <t:analyze-string select="poem"
                              regex="({item})              (.*)               (gesehen,)$"
                              flags="{$var1}i{$var2}">
               <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E3>
         <E4>
            <t:analyze-string select="poem"
                              regex="({item})(.*) (krahen)"
                              flags="ss{$var1}i{$var2}xxm">
  	            <t:matching-substring>
    		            <t:value-of select="'*'"/>
    	          </t:matching-substring>
    	          <t:non-matching-substring>
    		            <t:value-of select="."/>
    	          </t:non-matching-substring>
    	       </t:analyze-string>
         </E4>
      </out>
   </t:template>
</t:transform>
