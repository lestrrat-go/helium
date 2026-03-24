<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with whitespace text nodes as preceding-siblings of xsl:sort. 
  				Parent has @xml:space=preserve. Whitespace must be stripped. -->

   <t:template match="/doc">
      <t:variable name="out">
         <out>
            <t:for-each select="foo" xml:space="preserve">
	<t:sort/>  	 

<t:sort/>  	 
 &#xD;
</t:for-each>
         </out>
      </t:variable>
      <o xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:value-of select="string-to-codepoints($out/out)"/>
      </o>
   </t:template>
</t:transform>
