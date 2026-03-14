<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of single AVT in @regex of xsl:analyze-string. 
				Cases for AVT: no literal element, LE at start and end
				AVT value comes from a node or a variable.-->

   <t:variable name="var" select="/doc/str1"/>

   <t:template match="/doc">
      <out>
         <t:text>
</t:text>
         <node>
            <t:analyze-string select="'banana'" regex="{str1}">
		             <t:matching-substring>
			               <t:value-of select="'#'"/>
		             </t:matching-substring>
		             <t:non-matching-substring>
			               <t:value-of select="."/>
		             </t:non-matching-substring>
	           </t:analyze-string>
            <t:text> * </t:text>
            <t:analyze-string select="'banana'" regex="n{str1}n">
		             <t:matching-substring>
			               <t:value-of select="'#'"/>
		             </t:matching-substring>
		             <t:non-matching-substring>
			               <t:value-of select="."/>
		             </t:non-matching-substring>
	           </t:analyze-string>
         </node>
         <t:text>
</t:text>
         <variable>
            <t:analyze-string select="'banana'" regex="{$var}">
		             <t:matching-substring>
			               <t:value-of select="'#'"/>
		             </t:matching-substring>
		             <t:non-matching-substring>
			               <t:value-of select="."/>
		             </t:non-matching-substring>
	           </t:analyze-string>
            <t:text> * </t:text>
            <t:analyze-string select="'banana'" regex="n{$var}n">
		             <t:matching-substring>
			               <t:value-of select="'#'"/>
		             </t:matching-substring>
		             <t:non-matching-substring>
			               <t:value-of select="."/>
		             </t:non-matching-substring>
	           </t:analyze-string>
         </variable>
      </out>
   </t:template>
</t:transform>
